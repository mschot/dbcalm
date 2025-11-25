package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
)

type processRepository struct {
	db *DB
}

func NewProcessRepository(db *DB) repository.ProcessRepository {
	return &processRepository{db: db}
}

func (r *processRepository) Create(ctx context.Context, process *domain.Process) error {
	argsJSON, err := json.Marshal(process.Args)
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}

	query := `
		INSERT INTO process (command_id, command, pid, status, output, error, return_code, start_time, end_time, type, args)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var endTime sql.NullTime
	if process.EndTime != nil {
		endTime = sql.NullTime{Valid: true, Time: *process.EndTime}
	}

	result, err := r.db.ExecContext(ctx, query,
		process.CommandID,
		process.Command,
		NullInt(process.PID),
		process.Status,
		NullString(process.Output),
		NullString(process.Error),
		NullInt(process.ReturnCode),
		process.StartTime,
		endTime,
		process.Type,
		string(argsJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to create process: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	process.ID = id

	return nil
}

func (r *processRepository) FindByID(ctx context.Context, id int64) (*domain.Process, error) {
	query := `
		SELECT id, command_id, command, pid, status, output, error, return_code, start_time, end_time, type, args
		FROM process
		WHERE id = ?
	`
	return r.scanProcess(r.db.QueryRowContext(ctx, query, id))
}

func (r *processRepository) FindByCommandID(ctx context.Context, commandID string) (*domain.Process, error) {
	query := `
		SELECT id, command_id, command, pid, status, output, error, return_code, start_time, end_time, type, args
		FROM process
		WHERE command_id = ?
	`
	return r.scanProcess(r.db.QueryRowContext(ctx, query, commandID))
}

func (r *processRepository) Update(ctx context.Context, process *domain.Process) error {
	argsJSON, err := json.Marshal(process.Args)
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}

	query := `
		UPDATE process
		SET pid = ?, status = ?, output = ?, error = ?, return_code = ?, end_time = ?, args = ?
		WHERE id = ?
	`

	var endTime sql.NullTime
	if process.EndTime != nil {
		endTime = sql.NullTime{Valid: true, Time: *process.EndTime}
	}

	result, err := r.db.ExecContext(ctx, query,
		NullInt(process.PID),
		process.Status,
		NullString(process.Output),
		NullString(process.Error),
		NullInt(process.ReturnCode),
		endTime,
		string(argsJSON),
		process.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update process: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("process not found: %d", process.ID)
	}

	return nil
}

func (r *processRepository) List(ctx context.Context, filter repository.ProcessFilter) ([]*domain.Process, error) {
	query := `
		SELECT id, command_id, command, pid, status, output, error, return_code, start_time, end_time, type, args
		FROM process
		WHERE 1=1
	`
	args := []interface{}{}

	if filter.Type != nil {
		query += " AND type = ?"
		args = append(args, *filter.Type)
	}

	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, *filter.Status)
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
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}
	defer rows.Close()

	var processes []*domain.Process
	for rows.Next() {
		process, err := r.scanProcessRow(rows)
		if err != nil {
			return nil, err
		}
		processes = append(processes, process)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating processes: %w", err)
	}

	return processes, nil
}

func (r *processRepository) Count(ctx context.Context, filter repository.ProcessFilter) (int, error) {
	query := `SELECT COUNT(*) FROM processes WHERE 1=1`
	args := []interface{}{}

	if filter.Type != nil {
		query += " AND type = ?"
		args = append(args, *filter.Type)
	}

	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, *filter.Status)
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count processes: %w", err)
	}

	return count, nil
}

func (r *processRepository) FindRunning(ctx context.Context) ([]*domain.Process, error) {
	query := `
		SELECT id, command_id, command, pid, status, output, error, return_code, start_time, end_time, type, args
		FROM process
		WHERE status = ?
		ORDER BY start_time ASC
	`
	rows, err := r.db.QueryContext(ctx, query, domain.ProcessStatusRunning)
	if err != nil {
		return nil, fmt.Errorf("failed to find running processes: %w", err)
	}
	defer rows.Close()

	var processes []*domain.Process
	for rows.Next() {
		process, err := r.scanProcessRow(rows)
		if err != nil {
			return nil, err
		}
		processes = append(processes, process)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating running processes: %w", err)
	}

	return processes, nil
}

func (r *processRepository) scanProcess(row *sql.Row) (*domain.Process, error) {
	var process domain.Process
	var argsJSON string
	var pid, returnCode sql.NullInt64
	var output, errorOutput sql.NullString
	var endTime sql.NullTime

	err := row.Scan(
		&process.ID,
		&process.CommandID,
		&process.Command,
		&pid,
		&process.Status,
		&output,
		&errorOutput,
		&returnCode,
		&process.StartTime,
		&endTime,
		&process.Type,
		&argsJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("process not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan process: %w", err)
	}

	if pid.Valid {
		pidInt := int(pid.Int64)
		process.PID = &pidInt
	}
	if output.Valid {
		process.Output = &output.String
	}
	if errorOutput.Valid {
		process.Error = &errorOutput.String
	}
	if returnCode.Valid {
		rcInt := int(returnCode.Int64)
		process.ReturnCode = &rcInt
	}
	if endTime.Valid {
		process.EndTime = &endTime.Time
	}

	if err := json.Unmarshal([]byte(argsJSON), &process.Args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal args: %w", err)
	}

	return &process, nil
}

func (r *processRepository) scanProcessRow(rows *sql.Rows) (*domain.Process, error) {
	var process domain.Process
	var argsJSON string
	var pid, returnCode sql.NullInt64
	var output, errorOutput sql.NullString
	var endTime sql.NullTime

	err := rows.Scan(
		&process.ID,
		&process.CommandID,
		&process.Command,
		&pid,
		&process.Status,
		&output,
		&errorOutput,
		&returnCode,
		&process.StartTime,
		&endTime,
		&process.Type,
		&argsJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan process: %w", err)
	}

	if pid.Valid {
		pidInt := int(pid.Int64)
		process.PID = &pidInt
	}
	if output.Valid {
		process.Output = &output.String
	}
	if errorOutput.Valid {
		process.Error = &errorOutput.String
	}
	if returnCode.Valid {
		rcInt := int(returnCode.Int64)
		process.ReturnCode = &rcInt
	}
	if endTime.Valid {
		process.EndTime = &endTime.Time
	}

	if err := json.Unmarshal([]byte(argsJSON), &process.Args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal args: %w", err)
	}

	return &process, nil
}
