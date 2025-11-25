package repository

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Restore struct {
	ID              int
	StartTime       time.Time
	EndTime         *time.Time
	Target          string
	TargetPath      string
	BackupID        string
	BackupTimestamp *time.Time
	ProcessID       int
}

type RestoreRepository struct {
	dbPath string
}

func NewRestoreRepository(dbPath string) *RestoreRepository {
	return &RestoreRepository{dbPath: dbPath}
}

func (r *RestoreRepository) getDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", r.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, nil
}

func (r *RestoreRepository) Create(restore *Restore) error {
	db, err := r.getDB()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO restore (start_time, end_time, target, target_path, backup_id, backup_timestamp, process_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, restore.StartTime, restore.EndTime, restore.Target, restore.TargetPath, restore.BackupID, restore.BackupTimestamp, restore.ProcessID)

	if err != nil {
		return fmt.Errorf("failed to create restore: %w", err)
	}

	return nil
}
