package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/martijn/dbcalm/internal/api/dto"
	"github.com/martijn/dbcalm/internal/core/service"
	"github.com/martijn/dbcalm/internal/infrastructure/sqlite"
)

// testEnv holds all test dependencies
type testEnv struct {
	db             *sqlite.DB
	router         *gin.Engine
	backupHandler  *BackupHandler
	restoreHandler *RestoreHandler
	processHandler *ProcessHandler
}

// setupTestEnv creates a test environment with in-memory SQLite database
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Use in-memory SQLite database
	db, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Create repositories
	backupRepo := sqlite.NewBackupRepository(db)
	restoreRepo := sqlite.NewRestoreRepository(db)
	processRepo := sqlite.NewProcessRepository(db)
	scheduleRepo := sqlite.NewScheduleRepository(db)

	// Create services (without dbClient since we're only testing list endpoints)
	processService := service.NewProcessService(processRepo)
	backupService := service.NewBackupService(backupRepo, processService, nil)
	restoreService := service.NewRestoreService(restoreRepo, backupRepo, nil)

	// Create handlers
	backupHandler := NewBackupHandler(backupService, scheduleRepo)
	restoreHandler := NewRestoreHandler(restoreService, backupRepo)
	processHandler := NewProcessHandler(processService)

	// Setup gin router in test mode
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register routes without auth middleware
	router.GET("/backups", backupHandler.ListBackups)
	router.GET("/restores", restoreHandler.ListRestores)
	router.GET("/processes", processHandler.ListProcesses)

	return &testEnv{
		db:             db,
		router:         router,
		backupHandler:  backupHandler,
		restoreHandler: restoreHandler,
		processHandler: processHandler,
	}
}

// cleanup closes the test database
func (env *testEnv) cleanup() {
	if env.db != nil {
		env.db.Close()
	}
}

// seedTestData populates the database with test data for filtering tests
func (env *testEnv) seedTestData(t *testing.T) {
	t.Helper()

	// Base time: Nov 1, 2025
	baseTime := time.Date(2025, 11, 1, 10, 0, 0, 0, time.UTC)

	// Create 10 processes first (backups and restores have FK to process)
	processes := []struct {
		commandID string
		command   string
		status    string
		procType  string
		startTime time.Time
		endTime   *time.Time
	}{
		{"proc-001", "mariabackup --backup", "success", "backup", baseTime, ptr(baseTime.Add(10 * time.Minute))},
		{"proc-002", "mariabackup --backup", "success", "backup", baseTime.Add(24 * time.Hour), ptr(baseTime.Add(24*time.Hour + 10*time.Minute))},
		{"proc-003", "mariabackup --backup", "success", "backup", baseTime.Add(5 * 24 * time.Hour), ptr(baseTime.Add(5*24*time.Hour + 10*time.Minute))},
		{"proc-004", "mariabackup --backup", "success", "backup", baseTime.Add(10 * 24 * time.Hour), ptr(baseTime.Add(10*24*time.Hour + 10*time.Minute))},
		{"proc-005", "mariabackup --backup", "success", "backup", baseTime.Add(15 * 24 * time.Hour), ptr(baseTime.Add(15*24*time.Hour + 10*time.Minute))},
		{"proc-006", "mariabackup --backup", "success", "backup", baseTime.Add(20 * 24 * time.Hour), ptr(baseTime.Add(20*24*time.Hour + 10*time.Minute))},
		{"proc-007", "mariabackup --backup", "running", "backup", baseTime.Add(25 * 24 * time.Hour), nil},
		{"proc-008", "mariabackup --prepare", "success", "restore", baseTime.Add(3 * 24 * time.Hour), ptr(baseTime.Add(3*24*time.Hour + 30*time.Minute))},
		{"proc-009", "mariabackup --prepare", "failed", "restore", baseTime.Add(8 * 24 * time.Hour), ptr(baseTime.Add(8*24*time.Hour + 5*time.Minute))},
		{"proc-010", "cleanup old backups", "success", "cleanup_backups", baseTime.Add(12 * 24 * time.Hour), ptr(baseTime.Add(12*24*time.Hour + 2*time.Minute))},
	}

	for _, p := range processes {
		var endTimeStr interface{}
		if p.endTime != nil {
			endTimeStr = p.endTime.Format(time.RFC3339)
		}
		_, err := env.db.Exec(`
			INSERT INTO process (command_id, command, pid, status, start_time, end_time, type, args)
			VALUES (?, ?, 12345, ?, ?, ?, ?, '{}')
		`, p.commandID, p.command, p.status, p.startTime.Format(time.RFC3339), endTimeStr, p.procType)
		if err != nil {
			t.Fatalf("failed to seed process %s: %v", p.commandID, err)
		}
	}

	// Create 10 backups - 5 full (from_backup_id=NULL), 5 incremental
	backups := []struct {
		id           string
		fromBackupID *string
		startTime    time.Time
		endTime      *time.Time
		processID    int
	}{
		// Full backups (from_backup_id = NULL)
		{"backup-001", nil, baseTime, ptr(baseTime.Add(10 * time.Minute)), 1},
		{"backup-002", nil, baseTime.Add(5 * 24 * time.Hour), ptr(baseTime.Add(5*24*time.Hour + 10*time.Minute)), 3},
		{"backup-003", nil, baseTime.Add(10 * 24 * time.Hour), ptr(baseTime.Add(10*24*time.Hour + 10*time.Minute)), 4},
		{"backup-004", nil, baseTime.Add(15 * 24 * time.Hour), ptr(baseTime.Add(15*24*time.Hour + 10*time.Minute)), 5},
		{"backup-005", nil, baseTime.Add(20 * 24 * time.Hour), ptr(baseTime.Add(20*24*time.Hour + 10*time.Minute)), 6},
		// Incremental backups (from_backup_id != NULL)
		{"backup-006", ptr("backup-001"), baseTime.Add(1 * 24 * time.Hour), ptr(baseTime.Add(1*24*time.Hour + 5*time.Minute)), 2},
		{"backup-007", ptr("backup-002"), baseTime.Add(6 * 24 * time.Hour), ptr(baseTime.Add(6*24*time.Hour + 5*time.Minute)), 3},
		{"backup-008", ptr("backup-003"), baseTime.Add(11 * 24 * time.Hour), ptr(baseTime.Add(11*24*time.Hour + 5*time.Minute)), 4},
		{"backup-009", ptr("backup-004"), baseTime.Add(16 * 24 * time.Hour), ptr(baseTime.Add(16*24*time.Hour + 5*time.Minute)), 5},
		{"backup-010", ptr("backup-005"), baseTime.Add(21 * 24 * time.Hour), ptr(baseTime.Add(21*24*time.Hour + 5*time.Minute)), 6},
	}

	for _, b := range backups {
		var endTimeStr interface{}
		if b.endTime != nil {
			endTimeStr = b.endTime.Format(time.RFC3339)
		}
		_, err := env.db.Exec(`
			INSERT INTO backup (id, from_backup_id, start_time, end_time, process_id)
			VALUES (?, ?, ?, ?, ?)
		`, b.id, b.fromBackupID, b.startTime.Format(time.RFC3339), endTimeStr, b.processID)
		if err != nil {
			t.Fatalf("failed to seed backup %s: %v", b.id, err)
		}
	}

	// Create 5 restores
	restores := []struct {
		backupID        string
		backupTimestamp time.Time
		target          string
		targetPath      string
		startTime       time.Time
		endTime         *time.Time
		processID       int
	}{
		{"backup-001", baseTime, "database", "/var/lib/mysql", baseTime.Add(3 * 24 * time.Hour), ptr(baseTime.Add(3*24*time.Hour + 30*time.Minute)), 8},
		{"backup-002", baseTime.Add(5 * 24 * time.Hour), "folder", "/tmp/restore-1", baseTime.Add(8 * 24 * time.Hour), ptr(baseTime.Add(8*24*time.Hour + 5*time.Minute)), 9},
		{"backup-003", baseTime.Add(10 * 24 * time.Hour), "database", "/var/lib/mysql", baseTime.Add(13 * 24 * time.Hour), ptr(baseTime.Add(13*24*time.Hour + 25*time.Minute)), 8},
		{"backup-004", baseTime.Add(15 * 24 * time.Hour), "folder", "/tmp/restore-2", baseTime.Add(18 * 24 * time.Hour), ptr(baseTime.Add(18*24*time.Hour + 15*time.Minute)), 9},
		{"backup-005", baseTime.Add(20 * 24 * time.Hour), "database", "/var/lib/mysql", baseTime.Add(23 * 24 * time.Hour), ptr(baseTime.Add(23*24*time.Hour + 20*time.Minute)), 8},
	}

	for _, r := range restores {
		var endTimeStr interface{}
		if r.endTime != nil {
			endTimeStr = r.endTime.Format(time.RFC3339)
		}
		_, err := env.db.Exec(`
			INSERT INTO restore (backup_id, backup_timestamp, target, target_path, start_time, end_time, process_id)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, r.backupID, r.backupTimestamp.Format(time.RFC3339), r.target, r.targetPath, r.startTime.Format(time.RFC3339), endTimeStr, r.processID)
		if err != nil {
			t.Fatalf("failed to seed restore: %v", err)
		}
	}
}

// makeRequest performs a GET request and returns the response
func (env *testEnv) makeRequest(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	return w
}

// parseBackupListResponse parses the response body into BackupListResponse
func parseBackupListResponse(t *testing.T, w *httptest.ResponseRecorder) dto.BackupListResponse {
	t.Helper()

	var resp dto.BackupListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nBody: %s", err, w.Body.String())
	}
	return resp
}

// parseRestoreListResponse parses the response body into RestoreListResponse
func parseRestoreListResponse(t *testing.T, w *httptest.ResponseRecorder) dto.RestoreListResponse {
	t.Helper()

	var resp dto.RestoreListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nBody: %s", err, w.Body.String())
	}
	return resp
}

// parseProcessListResponse parses the response body into ProcessListResponse
func parseProcessListResponse(t *testing.T, w *httptest.ResponseRecorder) dto.ProcessListResponse {
	t.Helper()

	var resp dto.ProcessListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nBody: %s", err, w.Body.String())
	}
	return resp
}

// parseErrorResponse parses the response body into ErrorResponse
func parseErrorResponse(t *testing.T, w *httptest.ResponseRecorder) dto.ErrorResponse {
	t.Helper()

	var resp dto.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v\nBody: %s", err, w.Body.String())
	}
	return resp
}

// ptr is a helper to create a pointer to a value
func ptr[T any](v T) *T {
	return &v
}
