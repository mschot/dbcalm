package cli

import (
	"fmt"

	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/spf13/cobra"
)

var cleanupScheduleID int64

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Run backup cleanup",
	Long:  "Run backup cleanup based on retention policies (typically used by cron)",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := initServices(cmd.Context())
		if err != nil {
			return err
		}
		defer services.Close()

		var process *domain.Process
		if cleanupScheduleID > 0 {
			// Cleanup for specific schedule
			process, err = services.CleanupService.CleanupBySchedule(cmd.Context(), cleanupScheduleID)
			if err != nil {
				return fmt.Errorf("failed to start cleanup: %w", err)
			}
			fmt.Printf("Cleanup started for schedule %d\n", cleanupScheduleID)
		} else {
			// Cleanup for all schedules
			process, err = services.CleanupService.CleanupAll(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to start cleanup: %w", err)
			}
			fmt.Println("Cleanup started for all schedules")
		}

		fmt.Printf("Process ID: %d\n", process.ID)
		fmt.Printf("Command ID: %s\n", process.CommandID)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.Flags().Int64Var(&cleanupScheduleID, "schedule-id", 0, "Schedule ID (cleanup specific schedule)")
}
