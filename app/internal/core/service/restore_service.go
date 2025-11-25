package service

import (
	"context"
	"fmt"

	"github.com/martijn/dbcalm/internal/adapter/dbcmd"
	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
)

type RestoreService struct {
	restoreRepo repository.RestoreRepository
	backupRepo  repository.BackupRepository
	dbClient    *dbcmd.Client
}

func NewRestoreService(
	restoreRepo repository.RestoreRepository,
	backupRepo repository.BackupRepository,
	dbClient *dbcmd.Client,
) *RestoreService {
	return &RestoreService{
		restoreRepo: restoreRepo,
		backupRepo:  backupRepo,
		dbClient:    dbClient,
	}
}

// RestoreToDatabase restores a backup to the MySQL data directory
// Following Python's lean approach: validate, get backup chain, pass to db-cmd, return immediately
func (s *RestoreService) RestoreToDatabase(ctx context.Context, backupID string) (*domain.Process, error) {
	// Get backup chain (for incrementals) - returns list from oldest (full) to newest
	chain, err := s.backupRepo.FindChain(ctx, backupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup chain: %w", err)
	}

	// Build list of backup IDs for db-cmd service (matching Python's id_list)
	idList := make([]string, len(chain))
	for i, backup := range chain {
		idList[i] = backup.ID
	}

	// Send command to db-cmd service - it handles all the heavy lifting
	// (temp dirs, preparation, applying incrementals, cleanup, process updates, restore records)
	restoreArgs := map[string]interface{}{
		"id_list": idList,
		"target":  "database",
	}

	resp, err := s.dbClient.SendCommand(ctx, "restore_backup", restoreArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to send restore command: %w", err)
	}

	// Build process response from db-cmd response
	// The db-cmd service returns process info that we pass back to the handler
	// resp.ID contains the command_id (process identifier for status polling)
	process := &domain.Process{
		CommandID: resp.ID,
		Status:    domain.ProcessStatus(resp.Status),
	}

	return process, nil
}

// RestoreToFolder restores a backup to a folder for inspection
// Following Python's lean approach: validate, get backup chain, pass to db-cmd, return immediately
func (s *RestoreService) RestoreToFolder(ctx context.Context, backupID string) (*domain.Process, error) {
	// Get backup chain (for incrementals) - returns list from oldest (full) to newest
	chain, err := s.backupRepo.FindChain(ctx, backupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup chain: %w", err)
	}

	// Build list of backup IDs for db-cmd service (matching Python's id_list)
	idList := make([]string, len(chain))
	for i, backup := range chain {
		idList[i] = backup.ID
	}

	// Send command to db-cmd service - it handles all the heavy lifting
	// (directory creation, preparation, applying incrementals, process updates, restore records)
	restoreArgs := map[string]interface{}{
		"id_list": idList,
		"target":  "folder",
	}

	resp, err := s.dbClient.SendCommand(ctx, "restore_backup", restoreArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to send restore command: %w", err)
	}

	// Build process response from db-cmd response
	// The db-cmd service returns process info that we pass back to the handler
	// resp.ID contains the command_id (process identifier for status polling)
	process := &domain.Process{
		CommandID: resp.ID,
		Status:    domain.ProcessStatus(resp.Status),
	}

	return process, nil
}

// GetRestore retrieves a restore by ID
func (s *RestoreService) GetRestore(ctx context.Context, id int64) (*domain.Restore, error) {
	return s.restoreRepo.FindByID(ctx, id)
}

// ListRestores lists restores with filtering
func (s *RestoreService) ListRestores(ctx context.Context, filter repository.RestoreFilter) ([]*domain.Restore, error) {
	return s.restoreRepo.List(ctx, filter)
}

// CountRestores counts restores with filtering
func (s *RestoreService) CountRestores(ctx context.Context, filter repository.RestoreFilter) (int, error) {
	return s.restoreRepo.Count(ctx, filter)
}
