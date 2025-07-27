package cmd

import (
	"fmt"

	"littlevsx/internal/extensions"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [EXTENSION_ID]",
	Short: "Deletes an extension from the database and all associated files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return runDelete(args[0])
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(extensionID string) error {
	extManager, err := extensions.New()
	if err != nil {
		return fmt.Errorf("error initializing extension manager: %w", err)
	}
	defer extManager.Close()

	ext, exists := extManager.GetByID(extensionID)
	if !exists {
		return fmt.Errorf("extension with ID %s not found", extensionID)
	}

	fmt.Printf("Found extension for deletion:\n")
	fmt.Printf("  ID: %s\n", ext.ID)
	fmt.Printf("  Name: %s\n", ext.DisplayName)
	fmt.Printf("  Publisher: %s\n", ext.Publisher)
	fmt.Printf("  Version: %s\n", ext.Version)
	fmt.Printf("  File: %s\n", ext.FilePath)

	fmt.Printf("\n⚠️  WARNING: This action will permanently delete the extension and all associated files!\n")
	fmt.Printf("Continue with deletion? (y/N): ")

	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" {
		fmt.Println("Deletion cancelled")
		return nil
	}

	if err := extManager.DeleteExtension(extensionID); err != nil {
		return fmt.Errorf("error deleting extension: %w", err)
	}

	return nil
}
