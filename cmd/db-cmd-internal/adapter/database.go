package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/builder"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/config"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/constants"
	sharedProcess "github.com/martijn/dbcalm/shared/process"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/process"
)

// DatabaseAdapter handles database backup and restore operations
// Works with both MariaDB (via mariabackup) and MySQL (via xtrabackup)
type DatabaseAdapter struct {
	config  *config.Config
	builder builder.Builder
	runner  *sharedProcess.Runner
}

// NewDatabaseAdapter creates a new database adapter that works with both MariaDB and MySQL
func NewDatabaseAdapter(cfg *config.Config, bldr builder.Builder, runner *sharedProcess.Runner) *DatabaseAdapter {
	return &DatabaseAdapter{
		config:  cfg,
		builder: bldr,
		runner:  runner,
	}
}

func (a *DatabaseAdapter) FullBackup(id string, scheduleID *int) (*sharedProcess.Process, chan *sharedProcess.Process, error) {
	// Build command
	cmd := a.builder.BuildFullBackupCmd(id)

	// Prepare args
	args := map[string]interface{}{
		"id": id,
	}
	if scheduleID != nil {
		args["schedule_id"] = *scheduleID
	}

	// Execute command
	proc, procChan := a.runner.Execute(cmd, process.TypeBackup, nil, args)

	return proc, procChan, nil
}

func (a *DatabaseAdapter) IncrementalBackup(id, fromBackupID string, scheduleID *int) (*sharedProcess.Process, chan *sharedProcess.Process, error) {
	// Build command
	cmd := a.builder.BuildIncrementalBackupCmd(id, fromBackupID)

	// Prepare args
	args := map[string]interface{}{
		"id":             id,
		"from_backup_id": fromBackupID,
	}
	if scheduleID != nil {
		args["schedule_id"] = *scheduleID
	}

	// Execute command
	proc, procChan := a.runner.Execute(cmd, process.TypeBackup, nil, args)

	return proc, procChan, nil
}

func (a *DatabaseAdapter) RestoreBackup(idList []string, target string) (*sharedProcess.Process, chan *sharedProcess.Process, error) {
	// Create temporary directory
	var tmpDir string
	if target == string(builder.RestoreTargetDatabase) {
		tmpDir = fmt.Sprintf("%s%s", constants.TempRestorePrefix, uuid.New().String())
	} else {
		tmpDir = filepath.Join(a.config.BackupDir, "restores", time.Now().Format("2006-01-02-15-04-05"))
	}

	// Create the directory before using it
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create temporary restore directory: %w", err)
	}

	// Build restore commands
	commands := a.builder.BuildRestoreCmds(tmpDir, idList, target)

	// Prepare args
	args := map[string]interface{}{
		"id_list": idList,
		"target":  target,
		"tmp_dir": tmpDir,
	}

	// Execute consecutive commands
	proc, procChan := a.runner.ExecuteConsecutive(commands, process.TypeRestore, args)

	return proc, procChan, nil
}
