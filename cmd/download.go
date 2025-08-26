package cmd

import (
	"fmt"

	"littlevsx/internal/config"
	"littlevsx/internal/database"
	"littlevsx/internal/extensions"
	"littlevsx/internal/marketplace"

	"github.com/spf13/cobra"
)

var (
	marketplaceType string
)

var downloadCmd = &cobra.Command{
	Use:   "download --type MARKETPLACE_TYPE EXTENSION_ID",
	Short: "Downloads an extension from specified marketplace",
	Long: `Downloads an extension from the specified marketplace.
	
Supported marketplaces:
- microsoft: Microsoft Marketplace
- open-vsx: Open VSX Registry (open-vsx.org)

Examples:
  littlevsx download --type microsoft ms-python.python
  littlevsx download --type open-vsx jeanp413.open-remote-ssh`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return runDownload(args[0])
	},
}

func init() {
	downloadCmd.Flags().StringVarP(&marketplaceType, "type", "t", "", "Marketplace type: microsoft, open-vsx (required)")
	downloadCmd.MarkFlagRequired("type")
	rootCmd.AddCommand(downloadCmd)
}

func runDownload(extensionID string) error {
	if marketplaceType == "" {
		return fmt.Errorf("marketplace type is required, use --type flag")
	}

	config := config.GetConfig()

	extManager, err := extensions.New()
	if err != nil {
		return fmt.Errorf("error initializing extension manager: %w", err)
	}
	defer extManager.Close()

	factory := marketplace.NewFactory()
	marketplaceTypeEnum := marketplace.MarketplaceType(marketplaceType)

	mp, err := factory.CreateByType(marketplaceTypeEnum)
	if err != nil {
		return fmt.Errorf("error creating marketplace provider: %w", err)
	}

	fmt.Printf("Using marketplace: %s\n", mp.GetName())
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
