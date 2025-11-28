package service

import (
	"context"
	"fmt"
	"os"
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

	// Get expired backups for this schedule (synchronously)
	expiredBackups, err := s.getExpiredBackupsForSchedule(ctx, schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired backups: %w", err)
	}

	// Build lists for cleanup via socket service
	var backupIDs []string
	var folders []string
	for _, backup := range expiredBackups {
		backupIDs = append(backupIDs, backup.ID)
		folders = append(folders, filepath.Join(s.backupDir, backup.ID))
	}

	// Call cleanup via socket service (it will create the process)
	// Even if no backups to delete, cmd service will create a process that succeeds immediately
	cleanupArgs := map[string]interface{}{
		"backup_ids": backupIDs,
		"folders":    folders,
	}
	response, err := s.cmdClient.SendCommand(ctx, "cleanup_backups", cleanupArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate cleanup: %w", err)
	}
	if response.Code != 202 {
		return nil, fmt.Errorf("cleanup failed: %s", response.Status)
	}

	// Start background goroutine to wait for completion and delete DB records
	if len(expiredBackups) > 0 {
		go s.waitAndDeleteRecords(response.ID, expiredBackups)
	}

	// Return process stub - actual process was created by socket service
	return &domain.Process{
		CommandID: response.ID,
		Status:    domain.ProcessStatusRunning,
	}, nil
}

// CleanupAll runs cleanup for all schedules with retention policies
func (s *CleanupService) CleanupAll(ctx context.Context) (*domain.Process, error) {
	// Get all schedules (synchronously)
	schedules, err := s.scheduleRepo.List(ctx, repository.ScheduleFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to get schedules: %w", err)
	}

	var allExpiredBackups []*domain.Backup

	// Get expired backups for each schedule with retention policy
	for _, schedule := range schedules {
		if schedule.RetentionValue == nil || schedule.RetentionUnit == nil {
			continue
		}

		expiredBackups, err := s.getExpiredBackupsForSchedule(ctx, schedule)
		if err != nil {
			// Log warning but continue with other schedules
			continue
		}

		allExpiredBackups = append(allExpiredBackups, expiredBackups...)
	}

	// Build lists for cleanup via socket service
	var backupIDs []string
	var folders []string
	for _, backup := range allExpiredBackups {
		backupIDs = append(backupIDs, backup.ID)
		folders = append(folders, filepath.Join(s.backupDir, backup.ID))
	}

	// Call cleanup via socket service (it will create the process)
	// Even if no backups to delete, cmd service will create a process that succeeds immediately
	cleanupArgs := map[string]interface{}{
		"backup_ids": backupIDs,
		"folders":    folders,
	}
	response, err := s.cmdClient.SendCommand(ctx, "cleanup_backups", cleanupArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate cleanup: %w", err)
	}
	if response.Code != 202 {
		return nil, fmt.Errorf("cleanup failed: %s", response.Status)
	}

	// Start background goroutine to wait for completion and delete DB records
	if len(allExpiredBackups) > 0 {
		go s.waitAndDeleteRecords(response.ID, allExpiredBackups)
	}

	// Return process stub - actual process was created by socket service
	return &domain.Process{
		CommandID: response.ID,
		Status:    domain.ProcessStatusRunning,
	}, nil
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

// waitAndDeleteRecords waits for the cmd service process to complete, then deletes DB records
func (s *CleanupService) waitAndDeleteRecords(commandID string, backups []*domain.Backup) {
	ctx := context.Background()

	// Poll for cmd service process completion
	for {
		proc, err := s.processServ.GetProcessByCommandID(ctx, commandID)
		if err != nil {
			// Process not found yet, keep waiting
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if proc.Status == domain.ProcessStatusSuccess || proc.Status == domain.ProcessStatusFailed {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Delete records for folders that are gone
	var idsToDelete []string
	for _, backup := range backups {
		folderPath := filepath.Join(s.backupDir, backup.ID)
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			idsToDelete = append(idsToDelete, backup.ID)
		}
	}

	// Delete all records in one query (avoids CASCADE race condition)
	if len(idsToDelete) > 0 {
		_ = s.backupRepo.DeleteMany(ctx, idsToDelete)
	}
}
