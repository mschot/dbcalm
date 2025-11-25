package service

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/martijn/dbcalm/internal/adapter/cmd"
	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
)

type CleanupService struct {
	backupRepo   repository.BackupRepository
	scheduleRepo repository.ScheduleRepository
	processServ  *ProcessService
	cmdClient    *cmd.Client
	backupDir    string
}

func NewCleanupService(
	backupRepo repository.BackupRepository,
	scheduleRepo repository.ScheduleRepository,
	processServ *ProcessService,
	cmdClient *cmd.Client,
	backupDir string,
) *CleanupService {
	return &CleanupService{
		backupRepo:   backupRepo,
		scheduleRepo: scheduleRepo,
		processServ:  processServ,
		cmdClient:    cmdClient,
		backupDir:    backupDir,
	}
}

// CleanupBySchedule runs cleanup for a specific schedule
func (s *CleanupService) CleanupBySchedule(ctx context.Context, scheduleID int64) (*domain.Process, error) {
	// Get schedule
	schedule, err := s.scheduleRepo.FindByID(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("schedule not found: %w", err)
	}

	// Check if schedule has retention policy
	if schedule.RetentionValue == nil || schedule.RetentionUnit == nil {
		return nil, fmt.Errorf("schedule does not have a retention policy")
	}

	// Create process record
	args := map[string]interface{}{
		"schedule_id": scheduleID,
	}

	process, err := s.processServ.CreateProcess(ctx,
		fmt.Sprintf("cleanup --schedule-id %d", scheduleID),
		domain.ProcessTypeCleanupBackups,
		args,
	)
	if err != nil {
		return nil, err
	}

	// Execute cleanup asynchronously
	go func() {
		bgCtx := context.Background()

		// Get expired backups for this schedule
		expiredBackups, err := s.getExpiredBackupsForSchedule(bgCtx, schedule)
		if err != nil {
			process.Fail(fmt.Sprintf("Failed to get expired backups: %v", err))
			_ = s.processServ.processRepo.Update(bgCtx, process)
			return
		}

		// Delete backups
		output, deleteErr := s.deleteBackups(bgCtx, expiredBackups)
		if deleteErr != nil {
			process.Fail(fmt.Sprintf("Cleanup partially failed: %v\nOutput: %s", deleteErr, output))
		} else {
			process.Complete(0, output, "")
		}

		_ = s.processServ.processRepo.Update(bgCtx, process)
	}()

	return process, nil
}

// CleanupAll runs cleanup for all schedules with retention policies
func (s *CleanupService) CleanupAll(ctx context.Context) (*domain.Process, error) {
	// Create process record
	args := map[string]interface{}{
		"all_schedules": true,
	}

	process, err := s.processServ.CreateProcess(ctx,
		"cleanup --all",
		domain.ProcessTypeCleanupBackups,
		args,
	)
	if err != nil {
		return nil, err
	}

	// Execute cleanup asynchronously
	go func() {
		bgCtx := context.Background()

		// Get all schedules
		schedules, err := s.scheduleRepo.List(bgCtx, repository.ScheduleFilter{})
		if err != nil {
			process.Fail(fmt.Sprintf("Failed to get schedules: %v", err))
			_ = s.processServ.processRepo.Update(bgCtx, process)
			return
		}

		var allExpiredBackups []*domain.Backup
		var output string

		// Get expired backups for each schedule with retention policy
		for _, schedule := range schedules {
			if schedule.RetentionValue == nil || schedule.RetentionUnit == nil {
				continue
			}

			expiredBackups, err := s.getExpiredBackupsForSchedule(bgCtx, schedule)
			if err != nil {
				output += fmt.Sprintf("Warning: Failed to get expired backups for schedule %d: %v\n", schedule.ID, err)
				continue
			}

			allExpiredBackups = append(allExpiredBackups, expiredBackups...)
		}

		// Delete backups
		deleteOutput, deleteErr := s.deleteBackups(bgCtx, allExpiredBackups)
		output += deleteOutput

		if deleteErr != nil {
			process.Fail(fmt.Sprintf("Cleanup partially failed: %v\nOutput: %s", deleteErr, output))
		} else {
			process.Complete(0, output, "")
		}

		_ = s.processServ.processRepo.Update(bgCtx, process)
	}()

	return process, nil
}

// getExpiredBackupsForSchedule gets all expired backups for a schedule
func (s *CleanupService) getExpiredBackupsForSchedule(ctx context.Context, schedule *domain.Schedule) ([]*domain.Backup, error) {
	if schedule.RetentionValue == nil || schedule.RetentionUnit == nil {
		return nil, nil
	}

	// Calculate cutoff date
	cutoffDate := s.calculateCutoffDate(*schedule.RetentionValue, *schedule.RetentionUnit)

	// Get all backups for this schedule
	backups, err := s.backupRepo.FindBySchedule(ctx, schedule.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backups: %w", err)
	}

	// Group backups into chains
	chains := s.groupBackupsIntoChains(backups)

	// Find chains where ALL backups are older than cutoff
	var expiredBackups []*domain.Backup
	for _, chain := range chains {
		allExpired := true
		for _, backup := range chain {
			if backup.StartTime.After(cutoffDate) {
				allExpired = false
				break
			}
		}

		// Only delete complete chains
		if allExpired {
			expiredBackups = append(expiredBackups, chain...)
		}
	}

	return expiredBackups, nil
}

// calculateCutoffDate calculates the cutoff date for retention policy
func (s *CleanupService) calculateCutoffDate(retentionValue int, retentionUnit domain.RetentionUnit) time.Time {
	now := time.Now()

	switch retentionUnit {
	case domain.RetentionUnitDays:
		return now.AddDate(0, 0, -retentionValue)
	case domain.RetentionUnitWeeks:
		return now.AddDate(0, 0, -retentionValue*7)
	case domain.RetentionUnitMonths:
		return now.AddDate(0, -retentionValue, 0)
	default:
		return now
	}
}

// groupBackupsIntoChains groups backups into chains (full backup + its incrementals)
func (s *CleanupService) groupBackupsIntoChains(backups []*domain.Backup) [][]*domain.Backup {
	// Build a map of backup ID to backup
	backupMap := make(map[string]*domain.Backup)
	for _, backup := range backups {
		backupMap[backup.ID] = backup
	}

	// Find all full backups (root of chains)
	var fullBackups []*domain.Backup
	for _, backup := range backups {
		if backup.Type == domain.BackupTypeFull {
			fullBackups = append(fullBackups, backup)
		}
	}

	// Build chains
	var chains [][]*domain.Backup
	for _, fullBackup := range fullBackups {
		chain := []*domain.Backup{fullBackup}

		// Find all incrementals that depend on this full backup
		for _, backup := range backups {
			if backup.Type == domain.BackupTypeIncremental && backup.FromBackupID != nil {
				// Check if this incremental is part of this chain
				if s.isInChain(backup, fullBackup.ID, backupMap) {
					chain = append(chain, backup)
				}
			}
		}

		chains = append(chains, chain)
	}

	return chains
}

// isInChain checks if a backup is part of a chain starting from rootID
func (s *CleanupService) isInChain(backup *domain.Backup, rootID string, backupMap map[string]*domain.Backup) bool {
	if backup.FromBackupID == nil {
		return false
	}

	// Walk backward from this backup to find the root
	currentID := *backup.FromBackupID
	for {
		if currentID == rootID {
			return true
		}

		currentBackup, exists := backupMap[currentID]
		if !exists || currentBackup.FromBackupID == nil {
			return false
		}

		currentID = *currentBackup.FromBackupID
	}
}

// deleteBackups deletes a list of backups
func (s *CleanupService) deleteBackups(ctx context.Context, backups []*domain.Backup) (string, error) {
	if len(backups) == 0 {
		return "No backups to delete", nil
	}

	var output string
	output += fmt.Sprintf("Deleting %d backup(s)...\n", len(backups))

	// Build lists for cleanup via socket service
	var backupIDs []string
	var folders []string
	for _, backup := range backups {
		backupIDs = append(backupIDs, backup.ID)
		folders = append(folders, filepath.Join(s.backupDir, backup.ID))
	}

	// Call cleanup via socket service (matches Python architecture)
	cleanupArgs := map[string]interface{}{
		"backup_ids": backupIDs,
		"folders":    folders,
	}
	response, err := s.cmdClient.SendCommand(ctx, "cleanup_backups", cleanupArgs)
	if err != nil {
		return output, fmt.Errorf("failed to cleanup backups via socket service: %w", err)
	}
	if response.Code != 200 && response.Code != 202 {
		return output, fmt.Errorf("cleanup backups failed: %s", response.Status)
	}

	// Delete database records for all backups
	successCount := 0
	failCount := 0

	for _, backup := range backups {
		// Delete from database
		if err := s.backupRepo.Delete(ctx, backup.ID); err != nil {
			output += fmt.Sprintf("Warning: Failed to delete database record for %s: %v\n", backup.ID, err)
			failCount++
			continue
		}

		output += fmt.Sprintf("Successfully deleted backup %s\n", backup.ID)
		successCount++
	}

	output += fmt.Sprintf("\nSummary: %d deleted, %d failed\n", successCount, failCount)

	if failCount > 0 {
		return output, fmt.Errorf("%d backup(s) failed to delete", failCount)
	}

	return output, nil
}
