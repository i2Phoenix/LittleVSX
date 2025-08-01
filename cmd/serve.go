package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"littlevsx/internal/config"
	"littlevsx/internal/extensions"
	"littlevsx/internal/server"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the HTTP server for the marketplace",
	Long:  `Starts the HTTP server that provides the API to fetch VS Code extensions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return runServe()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe() error {
	config := config.GetConfig()

	extManager, err := extensions.New()
	if err != nil {
		return fmt.Errorf("error initializing extension manager: %w", err)
	}
	defer extManager.Close()

	var srv *server.Server
	if config.UseHTTPS {
		srv = server.NewWithHTTPS(extManager, config.CertFile, config.KeyFile, config.BaseURL)
	} else {
		srv = server.New(extManager, config.BaseURL)
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	if config.UseHTTPS {
		fmt.Printf("Server started. Marketplace is available at: %s://%s\n", "https", addr)
	} else {
		fmt.Printf("Server started. Marketplace is available at: %s://%s\n", "http", addr)
	}
	fmt.Println("Press Ctrl+C to stop the server")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(addr); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case sig := <-sigChan:
		fmt.Printf("\nSignal received: %v. Stopping server...\n", sig)
	case err := <-errChan:
		return fmt.Errorf("server start error: %w", err)
	}

	fmt.Println("Performing graceful shutdown...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Error during server shutdown: %v\n", err)
		return err
	}

	fmt.Println("Server stopped successfully")
	return nil
}
