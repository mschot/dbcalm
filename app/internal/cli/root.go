package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/martijn/dbcalm/internal/adapter/cmd"
	"github.com/martijn/dbcalm/internal/adapter/dbcmd"
	"github.com/martijn/dbcalm/internal/core/repository"
	"github.com/martijn/dbcalm/internal/core/service"
	"github.com/martijn/dbcalm/internal/infrastructure/sqlite"
	"github.com/martijn/dbcalm/pkg/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "dbcalm",
	Short: "DBCalm - Database backup and restore management",
	Long: `DBCalm is a comprehensive backup and restore management system for MariaDB/MySQL databases.

It provides:
- Full and incremental backups using mariabackup
- Scheduled backups with cron integration
- Retention policy management
- Database and folder restore capabilities
- REST API for remote management
- OAuth2 authentication`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for commands that don't need it
		if cmd.Name() == "version" || cmd.Name() == "help" {
			return nil
		}

		// Load configuration
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		return nil
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/dbcalm/config.yml)")
}

// initServices initializes all services
func initServices(ctx context.Context) (*Services, error) {
	// Initialize database
	db, err := sqlite.New(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize repositories
	userRepo := sqlite.NewUserRepository(db)
	clientRepo := sqlite.NewClientRepository(db)
	authCodeRepo := sqlite.NewAuthCodeRepository(db)
	backupRepo := sqlite.NewBackupRepository(db)
	restoreRepo := sqlite.NewRestoreRepository(db)
	scheduleRepo := sqlite.NewScheduleRepository(db)
	processRepo := sqlite.NewProcessRepository(db)

	// Initialize socket clients
	dbClient := dbcmd.NewClient(cfg.MariaDBCmdSocketPath, 30*time.Second)
	cmdClient := cmd.NewClient(cfg.CmdSocketPath, 30*time.Second)

	// Initialize services
	authService := service.NewAuthService(userRepo, clientRepo, authCodeRepo, cfg.JWTSecretKey, cfg.JWTAlgorithm)
	processService := service.NewProcessService(processRepo)
	processService.Start() // Start process queue monitor

	backupService := service.NewBackupService(backupRepo, processService, dbClient)
	restoreService := service.NewRestoreService(restoreRepo, backupRepo, dbClient)
	scheduleService := service.NewScheduleService(scheduleRepo, backupRepo, processService, cmdClient, "/usr/bin/dbcalm", cfg.LogFile)
	cleanupService := service.NewCleanupService(backupRepo, scheduleRepo, processService, cmdClient, cfg.BackupDir)

	return &Services{
		DB:              db,
		UserRepo:        userRepo,
		ClientRepo:      clientRepo,
		ScheduleRepo:    scheduleRepo,
		BackupRepo:      backupRepo,
		AuthService:     authService,
		ProcessService:  processService,
		BackupService:   backupService,
		RestoreService:  restoreService,
		ScheduleService: scheduleService,
		CleanupService:  cleanupService,
	}, nil
}

// Services holds all initialized services
type Services struct {
	DB              *sqlite.DB
	UserRepo        repository.UserRepository
	ClientRepo      repository.ClientRepository
	ScheduleRepo    repository.ScheduleRepository
	BackupRepo      repository.BackupRepository
	AuthService     *service.AuthService
	ProcessService  *service.ProcessService
	BackupService   *service.BackupService
	RestoreService  *service.RestoreService
	ScheduleService *service.ScheduleService
	CleanupService  *service.CleanupService
}

// Close closes all resources
func (s *Services) Close() {
	if s.ProcessService != nil {
		s.ProcessService.Stop()
	}
	if s.DB != nil {
		s.DB.Close()
	}
}
