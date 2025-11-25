package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/adapter"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/config"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/constants"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/handler"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/socket"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/validator"
	sharedProcess "github.com/martijn/dbcalm/shared/process"
	sharedSocket "github.com/martijn/dbcalm/shared/socket"
)

func main() {
	// Set up dual logging (file + stderr) during startup
	logFile, err := setupLogging()
	if err != nil {
		// If logging setup fails, report to stderr before exiting
		fmt.Fprintf(os.Stderr, "FATAL: Failed to setup logging: %v\n", err)
		os.Exit(1)
	}
	log.Println("Starting DBCalm Database Command Server")

	// Load configuration
	cfg, err := config.Load(constants.ConfigFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Loaded configuration: db_type=%s, backup_dir=%s", cfg.DbType, cfg.BackupDir)

	// Create process writer
	writer := sharedProcess.NewWriter(cfg.DatabasePath)

	// Create process runner
	runner := sharedProcess.NewRunner(writer)

	// Create adapter
	adptr, err := adapter.NewAdapter(cfg, runner)
	if err != nil {
		log.Fatalf("Failed to create adapter: %v", err)
	}

	// Create validator
	valid := validator.NewValidator(cfg)

	// Create queue handler
	queueHandler := handler.NewQueueHandler(cfg)

	// Create processor and socket server
	processor := socket.NewDbCommandProcessor(cfg, adptr, valid, queueHandler)
	server := sharedSocket.NewServer(constants.SocketPath, processor)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		log.Println("Shutting down gracefully...")
		os.Exit(0)
	}()

	// Start server
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	// Server started successfully, switch to file-only logging
	log.SetOutput(logFile)
	log.Println("Server started successfully, switched to file-only logging")
}

// setupLogging configures logging to write to both file and stderr during startup
// Returns the log file handle so main() can switch to file-only after successful startup
func setupLogging() (*os.File, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(constants.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", constants.LogDir, err)
	}

	// Open log file
	logFile, err := os.OpenFile(constants.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", constants.LogFile, err)
	}

	// Use MultiWriter to write to both file and stderr during startup
	// This ensures errors are visible in terminal/systemd journal
	multiWriter := io.MultiWriter(logFile, os.Stderr)
	log.SetOutput(multiWriter)

	// Set log flags with prefix
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("[db-cmd] ")

	return logFile, nil
}
