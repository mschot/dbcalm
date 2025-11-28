package utils

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Constants
const (
	HTTPTimeout            = 10 * time.Second
	BackupPollInterval     = 2 * time.Second
	DefaultBackupTimeout   = 300 * time.Second
	MySQLShutdownWait      = 2 * time.Second
	MariaDBStartWait       = 1 * time.Second
	MariaDBStatusCheckWait = 2 * time.Second
	ExpectedIncrementalUserCount = 3
)

// ProcessStatus represents the status response from the API
type ProcessStatus struct {
	Status     string                 `json:"status"`
	Error      string                 `json:"error,omitempty"`
	CommandID  string                 `json:"command_id"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// BackupResponse represents the response when creating a backup
type BackupResponse struct {
	PID string `json:"pid"`
}

// LoadSQLFile loads and executes SQL from a file
func LoadSQLFile(db *sql.DB, sqlFile string) error {
	sqlPath := filepath.Join("/tests", sqlFile)
	content, err := os.ReadFile(sqlPath)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}

	// Split by semicolon and execute each statement
	statements := strings.Split(string(content), ";")

	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	return nil
}

// ClearTestDatabase clears all test tables to prepare for a fresh test
func ClearTestDatabase(db *sql.DB) error {
	// Order matters due to foreign key constraints - delete child tables first
	tables := []string{"orders", "users"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}
	}
	return nil
}

// VerifyRowCount verifies row counts in tables match expected values
func VerifyRowCount(db *sql.DB, expectedCounts map[string]int) error {
	for table, expectedCount := range expectedCounts {
		var actualCount int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := db.QueryRow(query).Scan(&actualCount); err != nil {
			return fmt.Errorf("failed to count rows in %s: %w", table, err)
		}

		if actualCount != expectedCount {
			return fmt.Errorf("row count mismatch in %s: expected %d, got %d",
				table, expectedCount, actualCount)
		}
	}
	return nil
}

// VerifyDataIntegrity verifies data integrity for a specific dataset
func VerifyDataIntegrity(db *sql.DB, dataset string) error {
	// Check foreign key relationships
	rows, err := db.Query(`
		SELECT o.id, o.user_id, u.id as user_exists
		FROM orders o
		LEFT JOIN users u ON o.user_id = u.id
	`)
	if err != nil {
		return fmt.Errorf("failed to check foreign keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var orderID, userID int
		var userExists sql.NullInt64
		if err := rows.Scan(&orderID, &userID, &userExists); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		if !userExists.Valid {
			return fmt.Errorf("foreign key violation: order %d references non-existent user %d",
				orderID, userID)
		}
	}

	// Dataset-specific checks
	if dataset == "initial" {
		// Check we have exactly the initial users
		userRows, err := db.Query("SELECT username FROM users ORDER BY id")
		if err != nil {
			return fmt.Errorf("failed to query users: %w", err)
		}
		defer userRows.Close()

		var usernames []string
		for userRows.Next() {
			var username string
			if err := userRows.Scan(&username); err != nil {
				return fmt.Errorf("failed to scan username: %w", err)
			}
			usernames = append(usernames, username)
		}

		expected := []string{"alice", "bob", "charlie", "diana", "eve"}
		if len(usernames) != len(expected) {
			return fmt.Errorf("initial dataset mismatch: expected %d users, got %d",
				len(expected), len(usernames))
		}
		for i, username := range usernames {
			if username != expected[i] {
				return fmt.Errorf("initial dataset mismatch at position %d: expected %s, got %s",
					i, expected[i], username)
			}
		}
	} else if dataset == "full" {
		// Check we have all users including incremental
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM users
			WHERE username IN ('frank', 'grace', 'henry')
		`).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to count incremental users: %w", err)
		}
		if count != ExpectedIncrementalUserCount {
			return fmt.Errorf("incremental users not found in full dataset: expected %d, got %d",
				ExpectedIncrementalUserCount, count)
		}

		// Check updated email
		var email string
		err = db.QueryRow("SELECT email FROM users WHERE id = 3").Scan(&email)
		if err != nil {
			return fmt.Errorf("failed to query charlie's email: %w", err)
		}
		if email != "charlie.updated@example.com" {
			return fmt.Errorf("expected updated email for charlie, got %s", email)
		}
	}

	return nil
}

// ClearMySQLDataDirectory clears MySQL data directory preserving only runtime socket/pid files
func ClearMySQLDataDirectory() error {
	dataDir := "/var/lib/mysql"

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read data directory: %w", err)
	}

	preserveExtensions := map[string]bool{
		".sock": true,
		".pid":  true,
	}

	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if preserveExtensions[ext] {
			continue
		}

		path := filepath.Join(dataDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			fmt.Printf("Warning: Failed to remove %s: %v\n", path, err)
		}
	}

	// Verify directory is empty (except runtime files)
	entries, err = os.ReadDir(dataDir)
	if err != nil {
		return fmt.Errorf("failed to verify data directory: %w", err)
	}

	var remaining []string
	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if !preserveExtensions[ext] {
			remaining = append(remaining, entry.Name())
		}
	}

	if len(remaining) > 0 {
		return fmt.Errorf("data directory %s is not empty after clearing. Remaining files: %v",
			dataDir, remaining)
	}

	return nil
}

// WaitForBackupCompletion waits for a backup process to complete by polling its status
func WaitForBackupCompletion(token, processID, apiBaseURL string, timeout time.Duration) (*ProcessStatus, error) {
	if timeout == 0 {
		timeout = DefaultBackupTimeout
	}

	client := &http.Client{
		Timeout: HTTPTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	startTime := time.Now()

	for time.Since(startTime) < timeout {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/status/%s", apiBaseURL, processID), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to poll status: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("status check failed with code %d: %s", resp.StatusCode, string(body))
		}

		var status ProcessStatus
		if err := json.Unmarshal(body, &status); err != nil {
			return nil, fmt.Errorf("failed to parse status: %w", err)
		}

		if status.Status == "success" {
			return &status, nil
		}
		if status.Status == "failed" {
			return nil, fmt.Errorf("backup process failed: %s", status.Error)
		}

		time.Sleep(BackupPollInterval)
	}

	return nil, fmt.Errorf("backup process %s did not complete within %v", processID, timeout)
}

// WaitForRestoreCompletion waits for a restore process to complete by polling its status
func WaitForRestoreCompletion(token, processID, apiBaseURL string, timeout time.Duration) (*ProcessStatus, error) {
	if timeout == 0 {
		timeout = DefaultBackupTimeout
	}

	client := &http.Client{
		Timeout: HTTPTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	startTime := time.Now()

	for time.Since(startTime) < timeout {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/status/%s", apiBaseURL, processID), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to poll status: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("status check failed with code %d: %s", resp.StatusCode, string(body))
		}

		var status ProcessStatus
		if err := json.Unmarshal(body, &status); err != nil {
			return nil, fmt.Errorf("failed to parse status: %w", err)
		}

		fmt.Printf("DEBUG: Restore status check - Full response: %+v\n", status)
		fmt.Printf("DEBUG: Status value: %s\n", status.Status)

		if status.Status == "success" {
			fmt.Printf("DEBUG: Restore succeeded! Final status: %+v\n", status)
			return &status, nil
		}
		if status.Status == "failed" {
			fmt.Printf("DEBUG: Restore failed! Status: %+v\n", status)
			return nil, fmt.Errorf("restore process failed: %s", status.Error)
		}

		time.Sleep(BackupPollInterval)
	}

	return nil, fmt.Errorf("restore process %s did not complete within %v", processID, timeout)
}

// WaitForCleanupCompletion waits for a cleanup process to complete by polling its status
func WaitForCleanupCompletion(token, processID, apiBaseURL string, timeout time.Duration) (*ProcessStatus, error) {
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	client := &http.Client{
		Timeout: HTTPTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	fmt.Printf("[CLEANUP] Waiting for cleanup process %s to complete...\n", processID)
	startTime := time.Now()

	for time.Since(startTime) < timeout {
		elapsed := time.Since(startTime)

		req, err := http.NewRequest("GET", fmt.Sprintf("%s/status/%s", apiBaseURL, processID), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to poll status: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("status check failed with code %d: %s", resp.StatusCode, string(body))
		}

		var status ProcessStatus
		if err := json.Unmarshal(body, &status); err != nil {
			return nil, fmt.Errorf("failed to parse status: %w", err)
		}

		fmt.Printf("[CLEANUP] [%.1fs] Process %s status: %s\n", elapsed.Seconds(), processID, status.Status)

		if status.Status == "success" {
			fmt.Printf("[CLEANUP] Process %s completed successfully\n", processID)
			return &status, nil
		}
		if status.Status == "failed" {
			fmt.Printf("[CLEANUP] Process %s FAILED\n", processID)
			return nil, fmt.Errorf("cleanup process failed. Full status: %+v", status)
		}

		time.Sleep(BackupPollInterval)
	}

	return nil, fmt.Errorf("cleanup process %s did not complete within %v", processID, timeout)
}

// CreateBackup creates a backup and returns its process ID
func CreateBackup(token, backupType, fromBackupID, apiBaseURL string) (string, error) {
	client := &http.Client{
		Timeout: HTTPTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	payload := map[string]string{
		"type": backupType,
	}

	if backupType == "incremental" && fromBackupID != "" {
		payload["from_backup_id"] = fromBackupID
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/backups", apiBaseURL), strings.NewReader(string(payloadBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("backup creation failed with code %d: %s", resp.StatusCode, string(body))
	}

	var backupResp BackupResponse
	if err := json.Unmarshal(body, &backupResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return backupResp.PID, nil
}

// DatabaseService interface for managing database services
type DatabaseService interface {
	Start() error
	Stop() error
	IsRunning() bool
	EnsureRunning() error
}

// MariaDBService manages MariaDB service (without systemd)
type MariaDBService struct{}

// Start starts MariaDB service
func (m *MariaDBService) Start() error {
	// Create log file with correct ownership
	logFile := "/var/log/db-restart.log"
	if _, err := os.Create(logFile); err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	exec.Command("chown", "mysql:mysql", logFile).Run()
	exec.Command("chmod", "664", logFile).Run()

	// Start MariaDB in background
	cmd := exec.Command("mysqld_safe", "--log-error=/var/log/db-restart.log")
	fmt.Printf("DEBUG: Starting MariaDB with command: %v\n", cmd.Args)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MariaDB: %w", err)
	}

	fmt.Printf("DEBUG: MariaDB process started with PID: %d\n", cmd.Process.Pid)
	time.Sleep(MariaDBStartWait)

	// Wait for MariaDB to be ready (up to 30 seconds)
	for i := 0; i < 30; i++ {
		if m.IsRunning() {
			fmt.Printf("DEBUG: MariaDB is ready after %d seconds!\n", i+1)
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("failed to start MariaDB within 30 seconds")
}

// Stop stops MariaDB service
func (m *MariaDBService) Stop() error {
	exec.Command("mysqladmin", "-u", "root", "shutdown").Run()
	time.Sleep(MySQLShutdownWait)
	return nil
}

// IsRunning checks if MariaDB service is running
func (m *MariaDBService) IsRunning() bool {
	cmd := exec.Command("mysqladmin", "ping", "-h", "localhost", "--silent")
	return cmd.Run() == nil
}

// EnsureRunning ensures MariaDB is running
func (m *MariaDBService) EnsureRunning() error {
	if !m.IsRunning() {
		fmt.Println("WARNING: MariaDB is not running!")
		time.Sleep(MariaDBStatusCheckWait)
	}
	return nil
}

// MySQLService manages MySQL service (without systemd)
type MySQLService struct{}

// Start starts MySQL service
func (m *MySQLService) Start() error {
	// Debug: Check data directory ownership before starting
	fmt.Println("DEBUG: Checking /var/lib/mysql ownership before starting MySQL...")
	exec.Command("ls", "-la", "/var/lib/mysql").Run()

	// Fix ownership of data directory
	fmt.Println("DEBUG: Fixing ownership of /var/lib/mysql...")
	exec.Command("chown", "-R", "mysql:mysql", "/var/lib/mysql").Run()

	// Create log file with correct ownership
	logFile := "/var/log/db-restart.log"
	if _, err := os.Create(logFile); err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	exec.Command("chown", "mysql:mysql", logFile).Run()
	exec.Command("chmod", "664", logFile).Run()

	// Start MySQL in background
	cmd := exec.Command("mysqld", "--user=mysql", "--log-error=/var/log/db-restart.log")
	fmt.Printf("DEBUG: Starting MySQL with command: %v\n", cmd.Args)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MySQL: %w", err)
	}

	fmt.Printf("DEBUG: MySQL process started with PID: %d\n", cmd.Process.Pid)
	time.Sleep(MariaDBStartWait)

	// Wait for MySQL to be ready (up to 30 seconds)
	for i := 0; i < 30; i++ {
		if m.IsRunning() {
			fmt.Printf("DEBUG: MySQL is ready after %d seconds!\n", i+1)
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("failed to start MySQL within 30 seconds")
}

// Stop stops MySQL service
func (m *MySQLService) Stop() error {
	exec.Command("mysqladmin", "-u", "root", "shutdown").Run()
	time.Sleep(MySQLShutdownWait)
	return nil
}

// IsRunning checks if MySQL service is running
func (m *MySQLService) IsRunning() bool {
	cmd := exec.Command("mysqladmin", "ping", "-h", "localhost", "--silent")
	return cmd.Run() == nil
}

// EnsureRunning ensures MySQL is running
func (m *MySQLService) EnsureRunning() error {
	if !m.IsRunning() {
		fmt.Println("WARNING: MySQL is not running!")
		time.Sleep(MariaDBStatusCheckWait)
	}
	return nil
}

// GetDatabaseService returns appropriate database service based on DB_TYPE environment variable
func GetDatabaseService() DatabaseService {
	dbType := os.Getenv("DB_TYPE")
	if dbType == "mysql" {
		return &MySQLService{}
	}
	return &MariaDBService{}
}
