package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	scheduleID int64
	backupID   string
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create backups",
	Long:  "Create full or incremental backups (typically used by cron)",
}

var backupFullCmd = &cobra.Command{
	Use:   "full",
	Short: "Create a full backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := initServices(cmd.Context())
		if err != nil {
			return err
		}
		defer services.Close()

		var backupIDPtr *string
		if backupID != "" {
			backupIDPtr = &backupID
		}
		var scheduleIDPtr *int64
		if scheduleID > 0 {
			scheduleIDPtr = &scheduleID
		}

		process, err := services.BackupService.CreateFullBackup(cmd.Context(), backupIDPtr, scheduleIDPtr)
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}

		fmt.Printf("Full backup started\n")
		fmt.Printf("Process ID: %d\n", process.ID)
		fmt.Printf("Command ID: %s\n", process.CommandID)
		if scheduleID > 0 {
			fmt.Printf("Schedule ID: %d\n", scheduleID)
		}

		return nil
	},
}

var backupIncrementalCmd = &cobra.Command{
	Use:   "incremental",
	Short: "Create an incremental backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := initServices(cmd.Context())
		if err != nil {
			return err
		}
		defer services.Close()

		var backupIDPtr *string
		if backupID != "" {
			backupIDPtr = &backupID
		}
		var scheduleIDPtr *int64
		if scheduleID > 0 {
			scheduleIDPtr = &scheduleID
		}

		process, err := services.BackupService.CreateIncrementalBackup(cmd.Context(), backupIDPtr, nil, scheduleIDPtr)
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}

		fmt.Printf("Incremental backup started\n")
		fmt.Printf("Process ID: %d\n", process.ID)
		fmt.Printf("Command ID: %s\n", process.CommandID)
		if scheduleID > 0 {
			fmt.Printf("Schedule ID: %d\n", scheduleID)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupFullCmd)
	backupCmd.AddCommand(backupIncrementalCmd)

	// Add flags
	backupFullCmd.Flags().Int64Var(&scheduleID, "schedule-id", 0, "Schedule ID (for cron jobs)")
	backupFullCmd.Flags().StringVar(&backupID, "backup-id", "", "Custom backup ID")

	backupIncrementalCmd.Flags().Int64Var(&scheduleID, "schedule-id", 0, "Schedule ID (for cron jobs)")
	backupIncrementalCmd.Flags().StringVar(&backupID, "backup-id", "", "Custom backup ID")
}
