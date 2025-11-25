package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
)

type scheduleRepository struct {
	db *DB
}

func NewScheduleRepository(db *DB) repository.ScheduleRepository {
	return &scheduleRepository{db: db}
}

func (r *scheduleRepository) Create(ctx context.Context, schedule *domain.Schedule) error {
	query := `
		INSERT INTO schedule (backup_type, frequency, day_of_week, day_of_month, hour, minute,
			interval_value, interval_unit, retention_value, retention_unit, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var intervalUnit, retentionUnit sql.NullString
	if schedule.IntervalUnit != nil {
		intervalUnit = sql.NullString{String: string(*schedule.IntervalUnit), Valid: true}
	}
	if schedule.RetentionUnit != nil {
		retentionUnit = sql.NullString{String: string(*schedule.RetentionUnit), Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query,
		schedule.BackupType,
		schedule.Frequency,
		NullInt(schedule.DayOfWeek),
		NullInt(schedule.DayOfMonth),
		NullInt(schedule.Hour),
		NullInt(schedule.Minute),
		NullInt(schedule.IntervalValue),
		intervalUnit,
		NullInt(schedule.RetentionValue),
		retentionUnit,
		schedule.Enabled,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	schedule.ID = id

	return nil
}

func (r *scheduleRepository) FindByID(ctx context.Context, id int64) (*domain.Schedule, error) {
	query := `
		SELECT id, backup_type, frequency, day_of_week, day_of_month, hour, minute,
			interval_value, interval_unit, retention_value, retention_unit, enabled, created_at, updated_at
		FROM schedule
		WHERE id = ?
	`
	return r.scanSchedule(r.db.QueryRowContext(ctx, query, id))
}

func (r *scheduleRepository) Update(ctx context.Context, schedule *domain.Schedule) error {
	query := `
		UPDATE schedule
		SET backup_type = ?, frequency = ?, day_of_week = ?, day_of_month = ?, hour = ?, minute = ?,
			interval_value = ?, interval_unit = ?, retention_value = ?, retention_unit = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`

	var intervalUnit, retentionUnit sql.NullString
	if schedule.IntervalUnit != nil {
		intervalUnit = sql.NullString{String: string(*schedule.IntervalUnit), Valid: true}
	}
	if schedule.RetentionUnit != nil {
		retentionUnit = sql.NullString{String: string(*schedule.RetentionUnit), Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query,
		schedule.BackupType,
		schedule.Frequency,
		NullInt(schedule.DayOfWeek),
		NullInt(schedule.DayOfMonth),
		NullInt(schedule.Hour),
		NullInt(schedule.Minute),
		NullInt(schedule.IntervalValue),
		intervalUnit,
		NullInt(schedule.RetentionValue),
		retentionUnit,
		schedule.Enabled,
		schedule.UpdatedAt,
		schedule.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("schedule not found: %d", schedule.ID)
	}

	return nil
}

func (r *scheduleRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM schedule WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("schedule not found: %d", id)
	}

	return nil
}

func (r *scheduleRepository) List(ctx context.Context, filter repository.ScheduleFilter) ([]*domain.Schedule, error) {
	query := `
		SELECT id, backup_type, frequency, day_of_week, day_of_month, hour, minute,
			interval_value, interval_unit, retention_value, retention_unit, enabled, created_at, updated_at
		FROM schedule
		WHERE 1=1
	`
	args := []interface{}{}

	if filter.BackupType != nil {
		query += " AND backup_type = ?"
		args = append(args, *filter.BackupType)
	}

	if filter.Enabled != nil {
		query += " AND enabled = ?"
		args = append(args, *filter.Enabled)
	}

	query += " ORDER BY id ASC"

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
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*domain.Schedule
	for rows.Next() {
		schedule, err := r.scanScheduleRow(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schedules: %w", err)
	}

	return schedules, nil
}

func (r *scheduleRepository) Count(ctx context.Context, filter repository.ScheduleFilter) (int, error) {
	query := `SELECT COUNT(*) FROM schedule WHERE 1=1`
	args := []interface{}{}

	if filter.BackupType != nil {
		query += " AND backup_type = ?"
		args = append(args, *filter.BackupType)
	}

	if filter.Enabled != nil {
		query += " AND enabled = ?"
		args = append(args, *filter.Enabled)
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count schedules: %w", err)
	}

	return count, nil
}

func (r *scheduleRepository) FindEnabledFullSchedules(ctx context.Context) ([]*domain.Schedule, error) {
	query := `
		SELECT id, backup_type, frequency, day_of_week, day_of_month, hour, minute,
			interval_value, interval_unit, retention_value, retention_unit, enabled, created_at, updated_at
		FROM schedule
		WHERE backup_type = ? AND enabled = 1
		ORDER BY id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, domain.BackupTypeFull)
	if err != nil {
		return nil, fmt.Errorf("failed to find enabled full schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*domain.Schedule
	for rows.Next() {
		schedule, err := r.scanScheduleRow(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schedules: %w", err)
	}

	return schedules, nil
}

func (r *scheduleRepository) FindAllEnabled(ctx context.Context) ([]*domain.Schedule, error) {
	query := `
		SELECT id, backup_type, frequency, day_of_week, day_of_month, hour, minute,
			interval_value, interval_unit, retention_value, retention_unit, enabled, created_at, updated_at
		FROM schedule
		WHERE enabled = 1
		ORDER BY id ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find all enabled schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*domain.Schedule
	for rows.Next() {
		schedule, err := r.scanScheduleRow(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schedules: %w", err)
	}

	return schedules, nil
}

func (r *scheduleRepository) scanSchedule(row *sql.Row) (*domain.Schedule, error) {
	var schedule domain.Schedule
	var dayOfWeek, dayOfMonth, hour, minute, intervalValue, retentionValue sql.NullInt64
	var intervalUnit, retentionUnit sql.NullString

	err := row.Scan(
		&schedule.ID,
		&schedule.BackupType,
		&schedule.Frequency,
		&dayOfWeek,
		&dayOfMonth,
		&hour,
		&minute,
		&intervalValue,
		&intervalUnit,
		&retentionValue,
		&retentionUnit,
		&schedule.Enabled,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan schedule: %w", err)
	}

	if dayOfWeek.Valid {
		dow := int(dayOfWeek.Int64)
		schedule.DayOfWeek = &dow
	}
	if dayOfMonth.Valid {
		dom := int(dayOfMonth.Int64)
		schedule.DayOfMonth = &dom
	}
	if hour.Valid {
		h := int(hour.Int64)
		schedule.Hour = &h
	}
	if minute.Valid {
		m := int(minute.Int64)
		schedule.Minute = &m
	}
	if intervalValue.Valid {
		iv := int(intervalValue.Int64)
		schedule.IntervalValue = &iv
	}
	if intervalUnit.Valid {
		iu := domain.IntervalUnit(intervalUnit.String)
		schedule.IntervalUnit = &iu
	}
	if retentionValue.Valid {
		rv := int(retentionValue.Int64)
		schedule.RetentionValue = &rv
	}
	if retentionUnit.Valid {
		ru := domain.RetentionUnit(retentionUnit.String)
		schedule.RetentionUnit = &ru
	}

	return &schedule, nil
}

func (r *scheduleRepository) scanScheduleRow(rows *sql.Rows) (*domain.Schedule, error) {
	var schedule domain.Schedule
	var dayOfWeek, dayOfMonth, hour, minute, intervalValue, retentionValue sql.NullInt64
	var intervalUnit, retentionUnit sql.NullString

	err := rows.Scan(
		&schedule.ID,
		&schedule.BackupType,
		&schedule.Frequency,
		&dayOfWeek,
		&dayOfMonth,
		&hour,
		&minute,
		&intervalValue,
		&intervalUnit,
		&retentionValue,
		&retentionUnit,
		&schedule.Enabled,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan schedule: %w", err)
	}

	if dayOfWeek.Valid {
		dow := int(dayOfWeek.Int64)
		schedule.DayOfWeek = &dow
	}
	if dayOfMonth.Valid {
		dom := int(dayOfMonth.Int64)
		schedule.DayOfMonth = &dom
	}
	if hour.Valid {
		h := int(hour.Int64)
		schedule.Hour = &h
	}
	if minute.Valid {
		m := int(minute.Int64)
		schedule.Minute = &m
	}
	if intervalValue.Valid {
		iv := int(intervalValue.Int64)
		schedule.IntervalValue = &iv
	}
	if intervalUnit.Valid {
		iu := domain.IntervalUnit(intervalUnit.String)
		schedule.IntervalUnit = &iu
	}
	if retentionValue.Valid {
		rv := int(retentionValue.Int64)
		schedule.RetentionValue = &rv
	}
	if retentionUnit.Valid {
		ru := domain.RetentionUnit(retentionUnit.String)
		schedule.RetentionUnit = &ru
	}

	return &schedule, nil
}
