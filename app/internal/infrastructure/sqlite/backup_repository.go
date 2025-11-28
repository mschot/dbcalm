package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
)

type backupRepository struct {
	db *DB
}

func NewBackupRepository(db *DB) repository.BackupRepository {
	return &backupRepository{db: db}
}

func (r *backupRepository) Create(ctx context.Context, backup *domain.Backup) error {
	query := `
		INSERT INTO backup (id, from_backup_id, schedule_id, start_time, end_time, process_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	var endTime sql.NullTime
	if backup.EndTime != nil {
		endTime = sql.NullTime{Valid: true, Time: *backup.EndTime}
	}

	_, err := r.db.ExecContext(ctx, query,
		backup.ID,
		NullString(backup.FromBackupID),
		NullInt64(backup.ScheduleID),
		backup.StartTime,
		endTime,
		backup.ProcessID,
	)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	return nil
}

func (r *backupRepository) FindByID(ctx context.Context, id string) (*domain.Backup, error) {
	query := `
		SELECT id, from_backup_id, schedule_id, start_time, end_time, process_id
		FROM backup
		WHERE id = ?
	`
	return r.scanBackup(r.db.QueryRowContext(ctx, query, id))
}

func (r *backupRepository) Update(ctx context.Context, backup *domain.Backup) error {
	query := `
		UPDATE backup
		SET from_backup_id = ?, schedule_id = ?, end_time = ?
		WHERE id = ?
	`

	var endTime sql.NullTime
	if backup.EndTime != nil {
		endTime = sql.NullTime{Valid: true, Time: *backup.EndTime}
	}

	result, err := r.db.ExecContext(ctx, query,
		NullString(backup.FromBackupID),
		NullInt64(backup.ScheduleID),
		endTime,
		backup.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update backup: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("backup not found: %s", backup.ID)
	}

	return nil
}

func (r *backupRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM backup WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("backup not found: %s", id)
	}

	return nil
}

func (r *backupRepository) DeleteMany(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Build placeholders: ?, ?, ?
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM backup WHERE id IN (%s)", strings.Join(placeholders, ","))
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete backups: %w", err)
	}
	return nil
}

func (r *backupRepository) List(ctx context.Context, filter repository.BackupFilter) ([]*domain.Backup, error) {
	query := `
		SELECT id, from_backup_id, schedule_id, start_time, end_time, process_id
		FROM backup
		WHERE 1=1
	`
	args := []interface{}{}

	// Apply filters
	query, args = ApplyFilters(query, args, filter.Filters)

	// Apply ordering
	query = ApplyOrdering(query, filter.Order, "start_time DESC")

	// Apply pagination
	query, args = ApplyPagination(query, args, filter.Page, filter.PerPage)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}
	defer rows.Close()

	var backups []*domain.Backup
	for rows.Next() {
		backup, err := r.scanBackupRow(rows)
		if err != nil {
			return nil, err
		}
		backups = append(backups, backup)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating backups: %w", err)
	}

	return backups, nil
}

func (r *backupRepository) Count(ctx context.Context, filter repository.BackupFilter) (int, error) {
	query := `SELECT COUNT(*) FROM backup WHERE 1=1`
	args := []interface{}{}

	// Apply filters
	query, args = ApplyFilters(query, args, filter.Filters)

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count backups: %w", err)
	}

	return count, nil
}

func (r *backupRepository) FindLatestByScheduleAndType(ctx context.Context, scheduleID *int64, backupType domain.BackupType) (*domain.Backup, error) {
	query := `
		SELECT id, from_backup_id, schedule_id, start_time, end_time, process_id
		FROM backup
		WHERE end_time IS NOT NULL
	`
	args := []interface{}{}

	// Filter by type based on from_backup_id
	if backupType == domain.BackupTypeFull {
		query += " AND from_backup_id IS NULL"
	} else {
		query += " AND from_backup_id IS NOT NULL"
	}

	if scheduleID != nil {
		query += " AND schedule_id = ?"
		args = append(args, *scheduleID)
	} else {
		query += " AND schedule_id IS NULL"
	}

	query += " ORDER BY start_time DESC LIMIT 1"

	backup, err := r.scanBackup(r.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if err.Error() == "backup not found" {
			return nil, nil // No backup found is not an error
		}
		return nil, err
	}

	return backup, nil
}

func (r *backupRepository) FindChain(ctx context.Context, backupID string) ([]*domain.Backup, error) {
	// Walk backward from the given backup to find all backups in the chain
	var chain []*domain.Backup
	currentID := backupID

	for currentID != "" {
		backup, err := r.FindByID(ctx, currentID)
		if err != nil {
			return nil, fmt.Errorf("failed to find backup in chain: %w", err)
		}

		chain = append(chain, backup)

		// Move to parent backup (for incremental backups)
		if backup.FromBackupID != nil {
			currentID = *backup.FromBackupID
		} else {
			// Reached the full backup (root of chain)
			break
		}
	}

	// Reverse the chain so it goes from full backup to most recent
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	return chain, nil
}

func (r *backupRepository) FindBySchedule(ctx context.Context, scheduleID int64) ([]*domain.Backup, error) {
	query := `
		SELECT id, from_backup_id, schedule_id, start_time, end_time, process_id
		FROM backup
		WHERE schedule_id = ?
		ORDER BY start_time ASC
	`
	rows, err := r.db.QueryContext(ctx, query, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to find backups by schedule: %w", err)
	}
	defer rows.Close()

	var backups []*domain.Backup
	for rows.Next() {
		backup, err := r.scanBackupRow(rows)
		if err != nil {
			return nil, err
		}
		backups = append(backups, backup)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating backups: %w", err)
	}

	return backups, nil
}

func (r *backupRepository) scanBackup(row *sql.Row) (*domain.Backup, error) {
	var backup domain.Backup
	var fromBackupID sql.NullString
	var scheduleIDInt sql.NullInt64
	var endTime sql.NullTime

	err := row.Scan(
		&backup.ID,
		&fromBackupID,
		&scheduleIDInt,
		&backup.StartTime,
		&endTime,
		&backup.ProcessID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("backup not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan backup: %w", err)
	}

	// Infer type from from_backup_id: NULL = full, non-NULL = incremental
	if fromBackupID.Valid {
		backup.FromBackupID = &fromBackupID.String
		backup.Type = domain.BackupTypeIncremental
	} else {
		backup.Type = domain.BackupTypeFull
	}

	if scheduleIDInt.Valid {
		backup.ScheduleID = &scheduleIDInt.Int64
	}
	if endTime.Valid {
		backup.EndTime = &endTime.Time
	}

	return &backup, nil
}

func (r *backupRepository) scanBackupRow(rows *sql.Rows) (*domain.Backup, error) {
	var backup domain.Backup
	var fromBackupID sql.NullString
	var scheduleID sql.NullInt64
	var endTime sql.NullTime

	err := rows.Scan(
		&backup.ID,
		&fromBackupID,
		&scheduleID,
		&backup.StartTime,
		&endTime,
		&backup.ProcessID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan backup: %w", err)
	}

	// Infer type from from_backup_id: NULL = full, non-NULL = incremental
	if fromBackupID.Valid {
		backup.FromBackupID = &fromBackupID.String
		backup.Type = domain.BackupTypeIncremental
	} else {
		backup.Type = domain.BackupTypeFull
	}

	if scheduleID.Valid {
		backup.ScheduleID = &scheduleID.Int64
	}
	if endTime.Valid {
		backup.EndTime = &endTime.Time
	}

	return &backup, nil
}
