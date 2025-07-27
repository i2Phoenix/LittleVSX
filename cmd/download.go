package cmd

import (
	"fmt"

	"littlevsx/internal/config"
	"littlevsx/internal/database"
	"littlevsx/internal/extensions"
	"littlevsx/internal/marketplace"

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

	extManager, err := extensions.New()
	if err != nil {
		return fmt.Errorf("error initializing extension manager: %w", err)
	}
	defer extManager.Close()

	mp := marketplace.New()

	fmt.Println("Getting extension information...")
	info, err := mp.GetExtensionInfoByID(extensionID)
	if err != nil {
		return fmt.Errorf("error getting extension information: %w", err)
	}

	fmt.Printf("\nExtension information:\n")
	fmt.Printf("  ID: %s\n", info.ID)
	fmt.Printf("  Name: %s\n", info.DisplayName)
	fmt.Printf("  Publisher: %s\n", info.Publisher)
	fmt.Printf("  Version: %s\n", info.Version)
	if info.Description != "" {
		fmt.Printf("  Description: %s\n", info.Description)
	}

	fmt.Println("\nDownloading extension...")
	result, err := mp.DownloadExtension(info, config.ExtensionsDir)
	if err != nil {
		return fmt.Errorf("error downloading extension: %w", err)
	}

	if result.WasDownloaded {
		fmt.Printf("\n✅ Extension successfully downloaded: %s\n", result.FilePath)
		fmt.Println("Adding extension to database...")
		ext, err := extManager.ReadExtensionInfo(result.FilePath)
		if err != nil {
			return fmt.Errorf("error reading extension information: %w", err)
		}

		if ext.ReadmeContent != "" {
			fmt.Println("Processing README assets...")
			assetProcessor := extensions.NewAssetProcessor(config.AssetsDir, config.BaseURL)
			processedReadme, err := assetProcessor.ProcessReadme(ext.ReadmeContent, ext.ID)
			if err != nil {
				fmt.Printf("Warning: error processing assets: %v\n", err)
			} else {
				ext.ReadmeContent = processedReadme
				fmt.Println("✅ Assets processed")
			}
		}

		dbExt := database.ToDBExtension(ext)
		if err := extManager.GetDB().UpsertExtension(dbExt); err != nil {
			return fmt.Errorf("error saving extension to database: %w", err)
		}

		fmt.Printf("✅ Extension added to database: %s\n", ext.DisplayName)
		return nil
	} else {
		fmt.Printf("\nℹ️  Extension already exists: %s\n", result.FilePath)

		extensionID := info.ID
		existingExt, exists := extManager.GetByID(extensionID)

		if !exists {
			fmt.Println("Adding existing extension to database...")
			ext, err := extManager.ReadExtensionInfo(result.FilePath)
			if err != nil {
				return fmt.Errorf("error reading extension information: %w", err)
			}

			if ext.ReadmeContent != "" {
				fmt.Println("Processing README assets...")
				assetProcessor := extensions.NewAssetProcessor(config.AssetsDir, config.BaseURL)
				processedReadme, err := assetProcessor.ProcessReadme(ext.ReadmeContent, ext.ID)
				if err != nil {
					fmt.Printf("Warning: error processing assets: %v\n", err)
				} else {
					ext.ReadmeContent = processedReadme
					fmt.Println("✅ Assets processed")
				}
			}

			dbExt := database.ToDBExtension(ext)
			if err := extManager.GetDB().UpsertExtension(dbExt); err != nil {
				return fmt.Errorf("error saving extension to database: %w", err)
			}

			fmt.Printf("✅ Extension added to database: %s\n", ext.DisplayName)
			return nil
		} else {
			fmt.Printf("ℹ️  Extension already in database: %s\n", existingExt.DisplayName)
			return nil
		}
	}
}
