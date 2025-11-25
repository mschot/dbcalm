package process

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Writer struct {
	dbPath string
}

func NewWriter(dbPath string) *Writer {
	return &Writer{dbPath: dbPath}
}

func (w *Writer) getDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", w.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, nil
}

func (w *Writer) CreateProcess(command, commandID string, pid int, status, processType string, args map[string]interface{}, startTime time.Time) (int, error) {
	db, err := w.getDB()
	if err != nil {
		return 0, err
	}
	defer db.Close()

	argsJSON, err := json.Marshal(args)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal args: %w", err)
	}

	result, err := db.Exec(`
		INSERT INTO process (command, command_id, pid, status, output, error, return_code, start_time, end_time, type, args)
		VALUES (?, ?, ?, ?, NULL, NULL, NULL, ?, NULL, ?, ?)
	`, command, commandID, pid, status, startTime, processType, string(argsJSON))

	if err != nil {
		return 0, fmt.Errorf("failed to insert process: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return int(id), nil
}

func (w *Writer) UpdateProcessStatus(processID int, status string, output, errorMsg *string, returnCode *int, endTime *time.Time) error {
	db, err := w.getDB()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		UPDATE process 
		SET status = ?, output = ?, error = ?, return_code = ?, end_time = ?
		WHERE id = ?
	`, status, output, errorMsg, returnCode, endTime, processID)

	if err != nil {
		return fmt.Errorf("failed to update process: %w", err)
	}

	return nil
}

func (w *Writer) GetProcessByCommandID(commandID string) (*Process, error) {
	db, err := w.getDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var p Process
	var output, errorMsg sql.NullString
	var returnCode sql.NullInt64
	var endTime sql.NullTime
	var id sql.NullInt64

	err = db.QueryRow(`
		SELECT id, command, command_id, pid, status, output, error, return_code, start_time, end_time, type, args
		FROM process
		WHERE command_id = ?
	`, commandID).Scan(&id, &p.Command, &p.CommandID, &p.PID, &p.Status, &output, &errorMsg, &returnCode, &p.StartTime, &endTime, &p.Type, &p.ArgsJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query process: %w", err)
	}

	if id.Valid {
		idInt := int(id.Int64)
		p.ID = &idInt
	}
	if output.Valid {
		p.Output = &output.String
	}
	if errorMsg.Valid {
		p.Error = &errorMsg.String
	}
	if returnCode.Valid {
		rc := int(returnCode.Int64)
		p.ReturnCode = &rc
	}
	if endTime.Valid {
		p.EndTime = &endTime.Time
	}

	// Parse args JSON
	if p.ArgsJSON != "" {
		if err := json.Unmarshal([]byte(p.ArgsJSON), &p.Args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal args: %w", err)
		}
	}

	return &p, nil
}
