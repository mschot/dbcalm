package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
)

type restoreRepository struct {
	db *DB
}

func NewRestoreRepository(db *DB) repository.RestoreRepository {
	return &restoreRepository{db: db}
}

func (r *restoreRepository) Create(ctx context.Context, restore *domain.Restore) error {
	query := `
		INSERT INTO restore (backup_id, backup_timestamp, target, target_path, start_time, end_time, process_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	var endTime sql.NullTime
	if restore.EndTime != nil {
		endTime = sql.NullTime{Valid: true, Time: *restore.EndTime}
	}

	result, err := r.db.ExecContext(ctx, query,
		restore.BackupID,
		restore.BackupTimestamp,
		restore.Target,
		restore.TargetPath,
		restore.StartTime,
		endTime,
		restore.ProcessID,
	)
	if err != nil {
		return fmt.Errorf("failed to create restore: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	restore.ID = id

	return nil
}

func (r *restoreRepository) FindByID(ctx context.Context, id int64) (*domain.Restore, error) {
	query := `
		SELECT id, backup_id, backup_timestamp, target, target_path, start_time, end_time, process_id
		FROM restore
		WHERE id = ?
	`
	return r.scanRestore(r.db.QueryRowContext(ctx, query, id))
}

func (r *restoreRepository) Update(ctx context.Context, restore *domain.Restore) error {
	query := `
		UPDATE restore
		SET backup_id = ?, backup_timestamp = ?, target = ?, target_path = ?, end_time = ?
		WHERE id = ?
	`

	var endTime sql.NullTime
	if restore.EndTime != nil {
		endTime = sql.NullTime{Valid: true, Time: *restore.EndTime}
	}

	result, err := r.db.ExecContext(ctx, query,
		restore.BackupID,
		restore.BackupTimestamp,
		restore.Target,
		restore.TargetPath,
		endTime,
		restore.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update restore: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("restore not found: %d", restore.ID)
	}

	return nil
}

func (r *restoreRepository) List(ctx context.Context, filter repository.RestoreFilter) ([]*domain.Restore, error) {
	query := `
		SELECT id, backup_id, backup_timestamp, target, target_path, start_time, end_time, process_id
		FROM restore
		WHERE 1=1
	`
	args := []interface{}{}

	if filter.BackupID != nil {
		query += " AND backup_id = ?"
		args = append(args, *filter.BackupID)
	}

	if filter.Target != nil {
		query += " AND target = ?"
		args = append(args, *filter.Target)
	}

	query += " ORDER BY start_time DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list restores: %w", err)
	}
	defer rows.Close()

	var restores []*domain.Restore
	for rows.Next() {
		restore, err := r.scanRestoreRow(rows)
		if err != nil {
			return nil, err
		}
		restores = append(restores, restore)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating restores: %w", err)
	}

	return restores, nil
}

func (r *restoreRepository) Count(ctx context.Context, filter repository.RestoreFilter) (int, error) {
	query := `SELECT COUNT(*) FROM restore WHERE 1=1`
	args := []interface{}{}

	if filter.BackupID != nil {
		query += " AND backup_id = ?"
		args = append(args, *filter.BackupID)
	}

	if filter.Target != nil {
		query += " AND target = ?"
		args = append(args, *filter.Target)
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count restores: %w", err)
	}

	return count, nil
}

func (r *restoreRepository) scanRestore(row *sql.Row) (*domain.Restore, error) {
	var restore domain.Restore
	var endTime sql.NullTime

	err := row.Scan(
		&restore.ID,
		&restore.BackupID,
		&restore.BackupTimestamp,
		&restore.Target,
		&restore.TargetPath,
		&restore.StartTime,
		&endTime,
		&restore.ProcessID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("restore not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan restore: %w", err)
	}

	if endTime.Valid {
		restore.EndTime = &endTime.Time
	}

	return &restore, nil
}

func (r *restoreRepository) scanRestoreRow(rows *sql.Rows) (*domain.Restore, error) {
	var restore domain.Restore
	var endTime sql.NullTime

	err := rows.Scan(
		&restore.ID,
		&restore.BackupID,
		&restore.BackupTimestamp,
		&restore.Target,
		&restore.TargetPath,
		&restore.StartTime,
		&endTime,
		&restore.ProcessID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan restore: %w", err)
	}

	if endTime.Valid {
		restore.EndTime = &endTime.Time
	}

	return &restore, nil
}
