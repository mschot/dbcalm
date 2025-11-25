package domain

import "time"

type BackupType string

const (
	BackupTypeFull        BackupType = "full"
	BackupTypeIncremental BackupType = "incremental"
)

type Backup struct {
	ID           string     `db:"id"`
	Type         BackupType `db:"type"`
	FromBackupID *string    `db:"from_backup_id"` // For incremental backups
	ScheduleID   *int64     `db:"schedule_id"`  // For scheduled backups
	StartTime    time.Time  `db:"start_time"`
	EndTime      *time.Time `db:"end_time"`
	ProcessID    int64      `db:"process_id"`
	Size         *int64     `db:"size"` // In bytes
}

func NewBackup(id string, backupType BackupType, processID int64) *Backup {
	return &Backup{
		ID:        id,
		Type:      backupType,
		StartTime: time.Now(),
		ProcessID: processID,
	}
}

func (b *Backup) Complete(endTime time.Time, size *int64) {
	b.EndTime = &endTime
	b.Size = size
}
