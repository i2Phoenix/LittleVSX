package cmd

import (
	"fmt"

	"littlevsx/internal/config"
	"littlevsx/internal/database"
	"littlevsx/internal/extensions"
	"littlevsx/internal/marketplace"
	"littlevsx/internal/models"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download [EXTENSION_ID]",
	Short: "Downloads an extension from Microsoft Marketplace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return runDownload(args[0])
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}

func runDownload(extensionID string) error {
	config := config.GetConfig()

	extManager, err := initializeComponents(config)
	if err != nil {
		return err
	}
	defer extManager.Close()

	mp := marketplace.New()

	info, err := getExtensionInfo(mp, extensionID)
	if err != nil {
		return err
	}

	printExtensionInfo(info)

	result, err := downloadExtension(mp, info, config.ExtensionsDir)
	if err != nil {
		return err
	}

	return processDownloadResult(result, extManager, config)
}

func initializeComponents(config config.Config) (*extensions.Manager, error) {
	extManager, err := extensions.New()
	if err != nil {
		return nil, err
	}

	return extManager, nil
}

func getExtensionInfo(mp *marketplace.Marketplace, extensionID string) (*marketplace.ExtensionInfo, error) {
	fmt.Println("Getting extension information...")
	info, err := mp.GetExtensionInfoByID(extensionID)
	if err != nil {
		return nil, fmt.Errorf("error getting extension information: %w", err)
	}
	return info, nil
}

func printExtensionInfo(info *marketplace.ExtensionInfo) {
	fmt.Printf("\nExtension information:\n")
	fmt.Printf("  ID: %s\n", info.ID)
	fmt.Printf("  Name: %s\n", info.DisplayName)
	fmt.Printf("  Publisher: %s\n", info.Publisher)
	fmt.Printf("  Version: %s\n", info.Version)
	if info.Description != "" {
		fmt.Printf("  Description: %s\n", info.Description)
	}
}

func downloadExtension(mp *marketplace.Marketplace, info *marketplace.ExtensionInfo, extensionsDir string) (*marketplace.DownloadResult, error) {
	fmt.Println("\nDownloading extension...")
	result, err := mp.DownloadExtension(info, extensionsDir)
	if err != nil {
		return nil, fmt.Errorf("error downloading extension: %w", err)
	}
	return result, nil
}

func processDownloadResult(result *marketplace.DownloadResult, extManager *extensions.Manager, config config.Config) error {
	if result.WasDownloaded {
		return processNewDownload(result, extManager, config)
	} else {
		return processExistingFile(result, extManager, config)
	}
}

func processNewDownload(result *marketplace.DownloadResult, extManager *extensions.Manager, config config.Config) error {
	fmt.Printf("\n✅ Extension successfully downloaded: %s\n", result.FilePath)

	fmt.Println("Adding extension to database...")
	return addExtensionToDatabase(result.FilePath, extManager, config)
}

func processExistingFile(result *marketplace.DownloadResult, extManager *extensions.Manager, config config.Config) error {
	fmt.Printf("\nℹ️  Extension already exists: %s\n", result.FilePath)

	extensionID := extractExtensionID(result.FilePath)
	existingExt, exists := extManager.GetByID(extensionID)

	if !exists {
		fmt.Println("Adding existing extension to database...")
		return addExtensionToDatabase(result.FilePath, extManager, config)
	} else {
		fmt.Printf("ℹ️  Extension already in database: %s\n", existingExt.DisplayName)
		return nil
	}
}

func addExtensionToDatabase(filePath string, extManager *extensions.Manager, config config.Config) error {
	ext, err := extManager.ReadExtensionInfo(filePath)
	if err != nil {
		return fmt.Errorf("error reading extension information: %w", err)
	}

	if err := processReadmeAssets(ext, config); err != nil {
		fmt.Printf("Warning: error processing assets: %v\n", err)
	}

	dbExt := database.ToDBExtension(ext)
	if err := extManager.GetDB().UpsertExtension(dbExt); err != nil {
		return fmt.Errorf("error saving extension to database: %w", err)
	}

	fmt.Printf("✅ Extension added to database: %s\n", ext.DisplayName)
	return nil
}

func processReadmeAssets(ext *models.Extension, config config.Config) error {
	if ext.ReadmeContent == "" {
		return nil
	}

	fmt.Println("Processing README assets...")
	assetProcessor := extensions.NewAssetProcessor(config.AssetsDir, config.BaseURL)

	processedReadme, err := assetProcessor.ProcessReadme(ext.ReadmeContent, ext.ID)
	if err != nil {
		return err
	}

	ext.ReadmeContent = processedReadme
	fmt.Println("✅ Assets processed")
	return nil
}

func extractExtensionID(filePath string) string {
	return ""
}
