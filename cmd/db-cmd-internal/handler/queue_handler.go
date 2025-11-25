package handler

import (
	"log"
	"os"
	"path/filepath"

	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/builder"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/config"
	sharedProcess "github.com/martijn/dbcalm/shared/process"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/process"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/repository"
)

type QueueHandler struct {
	config    *config.Config
	backupRepo *repository.BackupRepository
	restoreRepo *repository.RestoreRepository
}

func NewQueueHandler(cfg *config.Config) *QueueHandler {
	return &QueueHandler{
		config:      cfg,
		backupRepo:  repository.NewBackupRepository(cfg.DatabasePath),
		restoreRepo: repository.NewRestoreRepository(cfg.DatabasePath),
	}
}

func (h *QueueHandler) Handle(processChan <-chan *sharedProcess.Process) {
	go func() {
		for proc := range processChan {
			h.handleProcess(proc)
		}
	}()
}

func (h *QueueHandler) handleProcess(proc *sharedProcess.Process) {
	if proc == nil {
		return
	}

	// Check if process failed
	if proc.ReturnCode != nil && *proc.ReturnCode != 0 {
		log.Printf("Process failed with return code %d: %s", *proc.ReturnCode, proc.Command)
		h.cleanupFailedProcess(proc)
		return
	}

	// Handle based on process type
	switch proc.Type {
	case process.TypeBackup:
		h.handleBackup(proc)
	case process.TypeRestore:
		h.handleRestore(proc)
	case process.TypeCleanupBackups:
		h.handleCleanupBackups(proc)
	default:
		log.Printf("Unknown process type: %s", proc.Type)
	}
}

func (h *QueueHandler) handleBackup(proc *sharedProcess.Process) {
	// Transform process to backup
	backup := &repository.Backup{
		ID:        proc.Args["id"].(string),
		StartTime: proc.StartTime,
		EndTime:   proc.EndTime,
		ProcessID: *proc.ID,
	}

	if fromBackupID, ok := proc.Args["from_backup_id"].(string); ok && fromBackupID != "" {
		backup.FromBackupID = &fromBackupID
	}

	if scheduleID, ok := proc.Args["schedule_id"].(float64); ok && scheduleID > 0 {
		sid := int(scheduleID)
		backup.ScheduleID = &sid
	}

	// Save to database
	err := h.backupRepo.Create(backup)
	if err != nil {
		log.Printf("Failed to create backup record: %v", err)
	} else {
		log.Printf("Backup created successfully: %s", backup.ID)
	}
}

func (h *QueueHandler) handleRestore(proc *sharedProcess.Process) {
	// Get id_list from args
	idListRaw, ok := proc.Args["id_list"]
	if !ok {
		log.Printf("Missing id_list in restore process args")
		return
	}

	var idList []string
	switch v := idListRaw.(type) {
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok {
				idList = append(idList, str)
			}
		}
	case []string:
		idList = v
	default:
		log.Printf("Invalid id_list type in restore process args")
		return
	}

	if len(idList) == 0 {
		log.Printf("Empty id_list in restore process args")
		return
	}

	backupID := idList[0] // Full backup
	latestBackupID := idList[len(idList)-1]

	// Get timestamp from latest backup
	latestBackup, err := h.backupRepo.Get(latestBackupID)
	if err != nil {
		log.Printf("Failed to get backup: %v", err)
	}

	// Create restore record
	restore := &repository.Restore{
		StartTime:  proc.StartTime,
		EndTime:    proc.EndTime,
		Target:     proc.Args["target"].(string),
		TargetPath: proc.Args["tmp_dir"].(string),
		BackupID:   backupID,
		ProcessID:  *proc.ID,
	}

	if latestBackup != nil {
		restore.BackupTimestamp = &latestBackup.StartTime
	}

	// Save to database
	err = h.restoreRepo.Create(restore)
	if err != nil {
		log.Printf("Failed to create restore record: %v", err)
	} else {
		log.Printf("Restore created successfully for backup: %s", backupID)
	}

	// Cleanup tmp folder for database restores
	if restore.Target == string(builder.RestoreTargetDatabase) {
		go h.removeTmpRestoreFolder(restore.TargetPath)
	}
}

func (h *QueueHandler) handleCleanupBackups(proc *sharedProcess.Process) {
	// TODO: Implement cleanup backups logic
	log.Printf("Cleanup backups completed")
}

func (h *QueueHandler) cleanupFailedProcess(proc *sharedProcess.Process) {
	// For failed backups, cleanup the backup folder if it exists
	if proc.Type == process.TypeBackup {
		if id, ok := proc.Args["id"].(string); ok {
			backupPath := filepath.Join(h.config.BackupDir, id)
			if _, err := os.Stat(backupPath); err == nil {
				log.Printf("Removing failed backup folder: %s", backupPath)
				if err := os.RemoveAll(backupPath); err != nil {
					log.Printf("Failed to remove backup folder: %v", err)
				}
			}
		}
	}
}

func (h *QueueHandler) removeTmpRestoreFolder(tmpPath string) {
	log.Printf("Removing temporary restore folder: %s", tmpPath)
	if err := os.RemoveAll(tmpPath); err != nil {
		log.Printf("Failed to remove temporary restore folder: %v", err)
	}
}
