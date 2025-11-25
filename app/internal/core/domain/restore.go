package domain

import "time"

type RestoreTarget string

const (
	RestoreTargetDatabase RestoreTarget = "database"
	RestoreTargetFolder   RestoreTarget = "folder"
)

type Restore struct {
	ID              int64         `db:"id"`
	BackupID        string        `db:"backup_id"`
	BackupTimestamp time.Time     `db:"backup_timestamp"`
	Target          RestoreTarget `db:"target"`
	TargetPath      string        `db:"target_path"`
	StartTime       time.Time     `db:"start_time"`
	EndTime         *time.Time    `db:"end_time"`
	ProcessID       int64         `db:"process_id"`
}

func NewRestore(backupID string, backupTimestamp time.Time, target RestoreTarget, targetPath string, processID int64) *Restore {
	return &Restore{
		BackupID:        backupID,
		BackupTimestamp: backupTimestamp,
		Target:          target,
		TargetPath:      targetPath,
		StartTime:       time.Now(),
		ProcessID:       processID,
	}
}

func (r *Restore) Complete(endTime time.Time) {
	r.EndTime = &endTime
}
