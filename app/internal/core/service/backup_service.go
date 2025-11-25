package service

import (
	"context"
	"fmt"
	"time"

	"github.com/martijn/dbcalm/internal/adapter/dbcmd"
	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
)

type BackupService struct {
	backupRepo repository.BackupRepository
	processServ *ProcessService
	dbClient   *dbcmd.Client
}

func NewBackupService(
	backupRepo repository.BackupRepository,
	processServ *ProcessService,
	dbClient *dbcmd.Client,
) *BackupService {
	return &BackupService{
		backupRepo:  backupRepo,
		processServ: processServ,
		dbClient:    dbClient,
	}
}

// CreateFullBackup creates a full backup via the socket service
func (s *BackupService) CreateFullBackup(ctx context.Context, backupID *string, scheduleID *int64) (*domain.Process, error) {
	// Generate backup ID if not provided
	if backupID == nil {
		id := time.Now().Format("20060102-150405")
		backupID = &id
	}

	// Build args for socket service
	args := map[string]interface{}{
		"id": *backupID,
	}
	if scheduleID != nil {
		args["schedule_id"] = *scheduleID
	}

	// Call socket service - it will create the process, build command, and execute
	response, err := s.dbClient.SendCommand(ctx, "full_backup", args)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate backup: %w", err)
	}

	// Check for 202 Accepted (async operation started)
	if response.Code != 202 {
		return nil, fmt.Errorf("backup failed: %s", response.Status)
	}

	// Return process stub - the actual process was created by socket service
	// The handler will need to look up the full process by command_id
	return &domain.Process{
		CommandID: response.ID,
		Status:    domain.ProcessStatusRunning,
	}, nil
}

// CreateIncrementalBackup creates an incremental backup via the socket service
func (s *BackupService) CreateIncrementalBackup(ctx context.Context, backupID *string, fromBackupID *string, scheduleID *int64) (*domain.Process, error) {
	// Find base backup if not specified
	if fromBackupID == nil {
		latestBackup, err := s.backupRepo.FindLatestByScheduleAndType(ctx, scheduleID, domain.BackupTypeFull)
		if err != nil {
			return nil, fmt.Errorf("failed to find base backup: %w", err)
		}
		if latestBackup == nil {
			return nil, fmt.Errorf("no full backup found to use as base")
		}
		fromBackupID = &latestBackup.ID
	}

	// Generate backup ID if not provided
	if backupID == nil {
		id := time.Now().Format("20060102-150405")
		backupID = &id
	}

	// Build args for socket service
	args := map[string]interface{}{
		"id":             *backupID,
		"from_backup_id": *fromBackupID,
	}
	if scheduleID != nil {
		args["schedule_id"] = *scheduleID
	}

	// Call socket service - it will create the process, build command, and execute
	response, err := s.dbClient.SendCommand(ctx, "incremental_backup", args)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate incremental backup: %w", err)
	}

	// Check for 202 Accepted (async operation started)
	if response.Code != 202 {
		return nil, fmt.Errorf("incremental backup failed: %s", response.Status)
	}

	// Return process stub - the actual process was created by socket service
	return &domain.Process{
		CommandID: response.ID,
		Status:    domain.ProcessStatusRunning,
	}, nil
}

// GetBackup retrieves a backup by ID
func (s *BackupService) GetBackup(ctx context.Context, id string) (*domain.Backup, error) {
	return s.backupRepo.FindByID(ctx, id)
}

// ListBackups lists backups with filtering
func (s *BackupService) ListBackups(ctx context.Context, filter repository.BackupFilter) ([]*domain.Backup, error) {
	return s.backupRepo.List(ctx, filter)
}

// CountBackups counts backups with filtering
func (s *BackupService) CountBackups(ctx context.Context, filter repository.BackupFilter) (int, error) {
	return s.backupRepo.Count(ctx, filter)
}

// GetBackupChain retrieves the full chain for a backup (for incrementals)
func (s *BackupService) GetBackupChain(ctx context.Context, backupID string) ([]*domain.Backup, error) {
	return s.backupRepo.FindChain(ctx, backupID)
}

// DeleteBackup deletes a backup
func (s *BackupService) DeleteBackup(ctx context.Context, id string) error {
	// Delete from filesystem via socket service
	delArgs := map[string]interface{}{"id": id}
	response, err := s.dbClient.SendCommand(ctx, "delete_backup", delArgs)
	if err != nil {
		return fmt.Errorf("failed to delete backup files: %w", err)
	}
	if response.Code != 200 {
		return fmt.Errorf("failed to delete backup files: %s", response.Status)
	}

	// Delete from database
	if err := s.backupRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete backup record: %w", err)
	}

	return nil
}
