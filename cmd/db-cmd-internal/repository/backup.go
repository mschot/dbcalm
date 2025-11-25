package repository

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Backup struct {
	ID           string
	FromBackupID *string
	ScheduleID   *int
	StartTime    time.Time
	EndTime      *time.Time
	ProcessID    int
}

type BackupRepository struct {
	dbPath string
}

func NewBackupRepository(dbPath string) *BackupRepository {
	return &BackupRepository{dbPath: dbPath}
}

func (r *BackupRepository) getDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", r.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, nil
}

func (r *BackupRepository) Create(backup *Backup) error {
	db, err := r.getDB()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO backup (id, from_backup_id, schedule_id, start_time, end_time, process_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, backup.ID, backup.FromBackupID, backup.ScheduleID, backup.StartTime, backup.EndTime, backup.ProcessID)

	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	return nil
}

func (r *BackupRepository) Get(id string) (*Backup, error) {
	db, err := r.getDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var backup Backup
	var fromBackupID sql.NullString
	var scheduleID sql.NullInt64
	var endTime sql.NullTime

	err = db.QueryRow(`
		SELECT id, from_backup_id, schedule_id, start_time, end_time, process_id
		FROM backup
		WHERE id = ?
	`, id).Scan(&backup.ID, &fromBackupID, &scheduleID, &backup.StartTime, &endTime, &backup.ProcessID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get backup: %w", err)
	}

	if fromBackupID.Valid {
		backup.FromBackupID = &fromBackupID.String
	}
	if scheduleID.Valid {
		sid := int(scheduleID.Int64)
		backup.ScheduleID = &sid
	}
	if endTime.Valid {
		backup.EndTime = &endTime.Time
	}

	return &backup, nil
}

func (r *BackupRepository) RequiredBackups(backupID string) ([]string, error) {
	var required []string
	current := backupID

	for current != "" {
		backup, err := r.Get(current)
		if err != nil {
			return nil, err
		}
		if backup == nil {
			return nil, fmt.Errorf("backup not found: %s", current)
		}

		required = append(required, backup.ID)

		if backup.FromBackupID == nil {
			break
		}
		current = *backup.FromBackupID
	}

	// Reverse to get oldest to newest
	for i := 0; i < len(required)/2; i++ {
		j := len(required) - 1 - i
		required[i], required[j] = required[j], required[i]
	}

	return required, nil
}

func (r *BackupRepository) LatestBackup() (*Backup, error) {
	db, err := r.getDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var backup Backup
	var fromBackupID sql.NullString
	var scheduleID sql.NullInt64
	var endTime sql.NullTime

	err = db.QueryRow(`
		SELECT id, from_backup_id, schedule_id, start_time, end_time, process_id
		FROM backup
		ORDER BY start_time DESC
		LIMIT 1
	`).Scan(&backup.ID, &fromBackupID, &scheduleID, &backup.StartTime, &endTime, &backup.ProcessID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest backup: %w", err)
	}

	if fromBackupID.Valid {
		backup.FromBackupID = &fromBackupID.String
	}
	if scheduleID.Valid {
		sid := int(scheduleID.Int64)
		backup.ScheduleID = &sid
	}
	if endTime.Valid {
		backup.EndTime = &endTime.Time
	}

	return &backup, nil
}
