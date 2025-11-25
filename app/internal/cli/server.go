package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/martijn/dbcalm/internal/api"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the API server",
	Long:  "Start the REST API server for remote management",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := initServices(cmd.Context())
		if err != nil {
			return err
		}
		defer services.Close()

		// Initialize Gin server
		server := api.NewServer(
			cfg,
			services.AuthService,
			services.ProcessService,
			services.BackupService,
			services.RestoreService,
			services.ScheduleService,
			services.CleanupService,
			services.ClientRepo,
			services.ScheduleRepo,
			services.BackupRepo,
		)

		// Start server in goroutine
		serverErr := make(chan error, 1)
		go func() {
			if err := server.Start(); err != nil && err != http.ErrServerClosed {
				serverErr <- err
			}
		}()

		// Wait for interrupt signal or server error
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		fmt.Println("Server is ready. Press Ctrl+C to stop.")

		select {
		case err := <-serverErr:
			return fmt.Errorf("server error: %w", err)
		case <-sigChan:
			fmt.Println("\nShutting down gracefully...")
		}

		// Graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}

		fmt.Println("Server stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
