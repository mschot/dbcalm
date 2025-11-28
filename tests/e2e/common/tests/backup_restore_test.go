package tests

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/martijn/dbcalm/tests/e2e/common/utils"
)

const (
	HTTPOK                  = 200
	HTTPAccepted            = 202
	HTTPServiceUnavailable  = 503
	HTTPTimeout             = 10
	MinExpectedBackups      = 2
)

// getAPIToken retrieves the API token from environment
func getAPIToken(t *testing.T) string {
	token := os.Getenv("API_TOKEN")
	if token == "" {
		t.Fatal("API_TOKEN environment variable not set")
	}
	return token
}

// getAPIBaseURL retrieves the API base URL from environment
func getAPIBaseURL(t *testing.T) string {
	url := os.Getenv("API_BASE_URL")
	if url == "" {
		url = "https://localhost:8335"
	}
	return url
}

// getDBConnection creates a database connection
func getDBConnection(t *testing.T) *sql.DB {
	db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=true")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	return db
}

// getDBService returns the appropriate database service
func getDBService() utils.DatabaseService {
	return utils.GetDatabaseService()
}

// httpClient returns a configured HTTP client
func httpClient() *http.Client {
	return &http.Client{
		Timeout: utils.HTTPTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

// TestFullBackupCreation tests creating a full backup
func TestFullBackupCreation(t *testing.T) {
	token := getAPIToken(t)
	apiURL := getAPIBaseURL(t)
	db := getDBConnection(t)
	defer db.Close()

	// Clear database before loading fixtures
	if err := utils.ClearTestDatabase(db); err != nil {
		t.Fatalf("Failed to clear test database: %v", err)
	}

	// Load initial dataset
	if err := utils.LoadSQLFile(db, "fixtures/initial_data.sql"); err != nil {
		t.Fatalf("Failed to load initial data: %v", err)
	}

	if err := utils.VerifyRowCount(db, map[string]int{"users": 5, "orders": 5}); err != nil {
		t.Fatalf("Row count verification failed: %v", err)
	}

	// Create full backup
	payload := map[string]string{"type": "full"}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/backups", apiURL), strings.NewReader(string(payloadBytes)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != HTTPOK && resp.StatusCode != HTTPAccepted {
		t.Fatalf("Failed to create backup: %s", string(body))
	}

	var backupData map[string]interface{}
	if err := json.Unmarshal(body, &backupData); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	processID, ok := backupData["pid"].(string)
	if !ok {
		t.Fatal("PID not found in response")
	}

	// Wait for backup to complete
	processStatus, err := utils.WaitForBackupCompletion(token, processID, apiURL, 0)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	// Get the actual backup ID from the process status
	backupID := processStatus.ResourceID
	if backupID == "" {
		t.Fatal("Backup ID not found in process status")
	}

	// Verify backup files exist
	backupDir := filepath.Join("/var/backups/dbcalm", backupID)
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Fatalf("Backup directory not found: %s", backupDir)
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("Backup directory is empty")
	}
}

// TestFullRestore tests restoring from a full backup
func TestFullRestore(t *testing.T) {
	token := getAPIToken(t)
	apiURL := getAPIBaseURL(t)
	db := getDBConnection(t)
	dbService := getDBService()

	// Ensure database is restarted even if test fails
	defer func() {
		if !dbService.IsRunning() {
			dbService.Start()
		}
	}()

	// Clear database before loading fixtures
	if err := utils.ClearTestDatabase(db); err != nil {
		t.Fatalf("Failed to clear test database: %v", err)
	}

	// Load initial dataset and create backup
	if err := utils.LoadSQLFile(db, "fixtures/initial_data.sql"); err != nil {
		t.Fatalf("Failed to load initial data: %v", err)
	}

	processID, err := utils.CreateBackup(token, "full", "", apiURL)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	processStatus, err := utils.WaitForBackupCompletion(token, processID, apiURL, 0)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	backupID := processStatus.ResourceID
	if backupID == "" {
		t.Fatal("Backup ID not found in process status")
	}

	// Add more data after backup
	if err := utils.LoadSQLFile(db, "fixtures/incremental_data.sql"); err != nil {
		t.Fatalf("Failed to load incremental data: %v", err)
	}

	if err := utils.VerifyRowCount(db, map[string]int{"users": 8, "orders": 7}); err != nil {
		t.Fatalf("Row count verification failed: %v", err)
	}

	// Close connection before restore
	db.Close()

	// Stop database and clear data directory
	if err := dbService.Stop(); err != nil {
		t.Fatalf("Failed to stop database: %v", err)
	}

	if err := utils.ClearMySQLDataDirectory(); err != nil {
		t.Fatalf("Failed to clear data directory: %v", err)
	}

	// Restore via API
	payload := map[string]string{
		"id":     backupID,
		"target": "database",
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/restore", apiURL), strings.NewReader(string(payloadBytes)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to start restore: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != HTTPOK && resp.StatusCode != HTTPAccepted {
		t.Fatalf("Failed to start restore: %s", string(body))
	}

	var restoreData map[string]interface{}
	if err := json.Unmarshal(body, &restoreData); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	restoreProcessID, ok := restoreData["pid"].(string)
	if !ok {
		t.Fatal("PID not found in restore response")
	}

	// Wait for restore to complete
	if _, err := utils.WaitForRestoreCompletion(token, restoreProcessID, apiURL, 0); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	time.Sleep(3 * time.Second)

	// Start database
	if err := dbService.Start(); err != nil {
		t.Fatalf("Failed to start database: %v", err)
	}

	// Reconnect and validate
	newDB := getDBConnection(t)
	defer newDB.Close()

	// Validate: Only initial data present (not incremental)
	if err := utils.VerifyRowCount(newDB, map[string]int{"users": 5, "orders": 5}); err != nil {
		t.Fatalf("Row count verification failed after restore: %v", err)
	}

	if err := utils.VerifyDataIntegrity(newDB, "initial"); err != nil {
		t.Fatalf("Data integrity verification failed: %v", err)
	}
}

// TestIncrementalBackupCreation tests creating an incremental backup
func TestIncrementalBackupCreation(t *testing.T) {
	token := getAPIToken(t)
	apiURL := getAPIBaseURL(t)
	db := getDBConnection(t)
	defer db.Close()

	// Clear database before loading fixtures
	if err := utils.ClearTestDatabase(db); err != nil {
		t.Fatalf("Failed to clear test database: %v", err)
	}

	// Load initial dataset and create full backup
	if err := utils.LoadSQLFile(db, "fixtures/initial_data.sql"); err != nil {
		t.Fatalf("Failed to load initial data: %v", err)
	}

	fullProcessID, err := utils.CreateBackup(token, "full", "", apiURL)
	if err != nil {
		t.Fatalf("Failed to create full backup: %v", err)
	}

	fullProcessStatus, err := utils.WaitForBackupCompletion(token, fullProcessID, apiURL, 0)
	if err != nil {
		t.Fatalf("Full backup failed: %v", err)
	}

	fullBackupID := fullProcessStatus.ResourceID
	if fullBackupID == "" {
		t.Fatal("Full backup ID not found in process status")
	}

	// Add incremental data
	if err := utils.LoadSQLFile(db, "fixtures/incremental_data.sql"); err != nil {
		t.Fatalf("Failed to load incremental data: %v", err)
	}

	if err := utils.VerifyRowCount(db, map[string]int{"users": 8, "orders": 7}); err != nil {
		t.Fatalf("Row count verification failed: %v", err)
	}

	// Create incremental backup
	payload := map[string]string{
		"type":           "incremental",
		"from_backup_id": fullBackupID,
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/backups", apiURL), strings.NewReader(string(payloadBytes)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to create incremental backup: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != HTTPOK && resp.StatusCode != HTTPAccepted {
		t.Fatalf("Failed to create incremental backup: %s", string(body))
	}

	var backupData map[string]interface{}
	if err := json.Unmarshal(body, &backupData); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	processID, ok := backupData["pid"].(string)
	if !ok {
		t.Fatal("PID not found in response")
	}

	// Wait for backup to complete
	processStatus, err := utils.WaitForBackupCompletion(token, processID, apiURL, 0)
	if err != nil {
		t.Fatalf("Incremental backup failed: %v", err)
	}

	incrementalBackupID := processStatus.ResourceID
	if incrementalBackupID == "" {
		t.Fatal("Incremental backup ID not found in process status")
	}

	// Verify backup files exist
	backupDir := filepath.Join("/var/backups/dbcalm", incrementalBackupID)
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Fatalf("Incremental backup directory not found: %s", backupDir)
	}
}

// TestIncrementalRestore tests restoring from incremental backup (includes full backup chain)
func TestIncrementalRestore(t *testing.T) {
	token := getAPIToken(t)
	apiURL := getAPIBaseURL(t)
	db := getDBConnection(t)
	dbService := getDBService()

	// Ensure database is restarted even if test fails
	defer func() {
		if !dbService.IsRunning() {
			dbService.Start()
		}
	}()

	// Clear database before loading fixtures
	if err := utils.ClearTestDatabase(db); err != nil {
		t.Fatalf("Failed to clear test database: %v", err)
	}

	// Load initial dataset and create full backup
	if err := utils.LoadSQLFile(db, "fixtures/initial_data.sql"); err != nil {
		t.Fatalf("Failed to load initial data: %v", err)
	}

	fullProcessID, err := utils.CreateBackup(token, "full", "", apiURL)
	if err != nil {
		t.Fatalf("Failed to create full backup: %v", err)
	}

	fullProcessStatus, err := utils.WaitForBackupCompletion(token, fullProcessID, apiURL, 0)
	if err != nil {
		t.Fatalf("Full backup failed: %v", err)
	}

	fullBackupID := fullProcessStatus.ResourceID
	if fullBackupID == "" {
		t.Fatal("Full backup ID not found in process status")
	}

	// Add incremental data and create incremental backup
	if err := utils.LoadSQLFile(db, "fixtures/incremental_data.sql"); err != nil {
		t.Fatalf("Failed to load incremental data: %v", err)
	}

	incrProcessID, err := utils.CreateBackup(token, "incremental", fullBackupID, apiURL)
	if err != nil {
		t.Fatalf("Failed to create incremental backup: %v", err)
	}

	incrProcessStatus, err := utils.WaitForBackupCompletion(token, incrProcessID, apiURL, 0)
	if err != nil {
		t.Fatalf("Incremental backup failed: %v", err)
	}

	incrementalBackupID := incrProcessStatus.ResourceID
	if incrementalBackupID == "" {
		t.Fatal("Incremental backup ID not found in process status")
	}

	// Close connection before restore
	db.Close()

	// Stop database and clear data directory
	if err := dbService.Stop(); err != nil {
		t.Fatalf("Failed to stop database: %v", err)
	}

	if err := utils.ClearMySQLDataDirectory(); err != nil {
		t.Fatalf("Failed to clear data directory: %v", err)
	}

	// Restore incremental backup (should restore full + incremental)
	payload := map[string]string{
		"id":     incrementalBackupID,
		"target": "database",
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/restore", apiURL), strings.NewReader(string(payloadBytes)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to start restore: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != HTTPOK && resp.StatusCode != HTTPAccepted {
		t.Fatalf("Failed to start restore: %s", string(body))
	}

	var restoreData map[string]interface{}
	if err := json.Unmarshal(body, &restoreData); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	restoreProcessID, ok := restoreData["pid"].(string)
	if !ok {
		t.Fatal("PID not found in restore response")
	}

	// Wait for restore to complete
	if _, err := utils.WaitForRestoreCompletion(token, restoreProcessID, apiURL, 0); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Wait a second before starting the database
	time.Sleep(1 * time.Second)

	// Start database
	if err := dbService.Start(); err != nil {
		t.Fatalf("Failed to start database: %v", err)
	}

	// Reconnect and validate
	newDB := getDBConnection(t)
	defer newDB.Close()

	// Validate: All data present (initial + incremental)
	if err := utils.VerifyRowCount(newDB, map[string]int{"users": 8, "orders": 7}); err != nil {
		t.Fatalf("Row count verification failed after restore: %v", err)
	}

	if err := utils.VerifyDataIntegrity(newDB, "full"); err != nil {
		t.Fatalf("Data integrity verification failed: %v", err)
	}
}

// TestBackupRequiresCredentialsFile tests that backup fails when credentials file is missing
func TestBackupRequiresCredentialsFile(t *testing.T) {
	token := getAPIToken(t)
	apiURL := getAPIBaseURL(t)

	credentialsFile := "/etc/dbcalm/credentials.cnf"
	var originalContent []byte

	// Read and remove credentials file
	if _, err := os.Stat(credentialsFile); err == nil {
		originalContent, _ = os.ReadFile(credentialsFile)
		os.Remove(credentialsFile)
	}

	defer func() {
		// Restore credentials file - ensure this runs before test returns
		// Use mode 0644 so mysql user can read it (dbcalm-db-cmd runs as mysql)
		if originalContent != nil {
			if err := os.WriteFile(credentialsFile, originalContent, 0644); err != nil {
				t.Errorf("Failed to restore credentials file: %v", err)
			}
		}
	}()

	// Attempt backup - should fail with 503
	payload := map[string]string{"type": "full"}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/backups", apiURL), strings.NewReader(string(payloadBytes)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != HTTPServiceUnavailable {
		t.Fatalf("Expected 503, got %d", resp.StatusCode)
	}

	bodyStr := strings.ToLower(string(body))
	if !strings.Contains(bodyStr, "credentials") {
		t.Fatalf("Expected error message to contain 'credentials', got: %s", string(body))
	}
}

// TestBackupRequiresClientDBCalmSection tests that backup fails when [client-dbcalm] section is missing
func TestBackupRequiresClientDBCalmSection(t *testing.T) {
	token := getAPIToken(t)
	apiURL := getAPIBaseURL(t)

	credentialsFile := "/etc/dbcalm/credentials.cnf"
	originalContent, err := os.ReadFile(credentialsFile)
	if err != nil {
		t.Fatalf("Failed to read credentials file: %v", err)
	}

	// Write credentials file without [client-dbcalm] section
	// Use mode 0644 so mysql user can read it (dbcalm-db-cmd runs as mysql)
	if err := os.WriteFile(credentialsFile, []byte("[client]\nuser=test\npassword=test\n"), 0644); err != nil {
		t.Fatalf("Failed to modify credentials file: %v", err)
	}

	defer func() {
		// Restore original credentials - ensure this runs before test returns
		// Use mode 0644 so mysql user can read it (dbcalm-db-cmd runs as mysql)
		if err := os.WriteFile(credentialsFile, originalContent, 0644); err != nil {
			t.Errorf("Failed to restore credentials file: %v", err)
		}
	}()

	// Attempt backup - should fail with 503
	payload := map[string]string{"type": "full"}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/backups", apiURL), strings.NewReader(string(payloadBytes)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != HTTPServiceUnavailable {
		t.Fatalf("Expected 503, got %d", resp.StatusCode)
	}

	bodyStr := strings.ToLower(string(body))
	if !strings.Contains(bodyStr, "client-dbcalm") {
		t.Fatalf("Expected error message to contain 'client-dbcalm', got: %s", string(body))
	}
}

// TestRestoreRequiresServerStopped tests that database restore fails when database server is running
func TestRestoreRequiresServerStopped(t *testing.T) {
	token := getAPIToken(t)
	apiURL := getAPIBaseURL(t)
	db := getDBConnection(t)
	defer db.Close()
	dbService := getDBService()

	// Clear database before loading fixtures
	if err := utils.ClearTestDatabase(db); err != nil {
		t.Fatalf("Failed to clear test database: %v", err)
	}

	// Create a backup to restore from
	if err := utils.LoadSQLFile(db, "fixtures/initial_data.sql"); err != nil {
		t.Fatalf("Failed to load initial data: %v", err)
	}

	processID, err := utils.CreateBackup(token, "full", "", apiURL)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	processStatus, err := utils.WaitForBackupCompletion(token, processID, apiURL, 0)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	backupID := processStatus.ResourceID
	if backupID == "" {
		t.Fatal("Backup ID not found in process status")
	}

	// Ensure database is running
	dbService.EnsureRunning()

	// Attempt restore - should fail with 503
	payload := map[string]string{
		"id":     backupID,
		"target": "database",
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/restore", apiURL), strings.NewReader(string(payloadBytes)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != HTTPServiceUnavailable {
		t.Fatalf("Expected 503, got %d", resp.StatusCode)
	}

	bodyStr := strings.ToLower(string(body))
	if !strings.Contains(bodyStr, "server") {
		t.Fatalf("Expected error message to contain 'server', got: %s", string(body))
	}
	if !strings.Contains(bodyStr, "stopped") && !strings.Contains(bodyStr, "dead") {
		t.Fatalf("Expected error message to contain 'stopped' or 'dead', got: %s", string(body))
	}
}

// TestRestoreRequiresEmptyDataDirectory tests that restore fails when data directory is not empty
func TestRestoreRequiresEmptyDataDirectory(t *testing.T) {
	token := getAPIToken(t)
	apiURL := getAPIBaseURL(t)
	db := getDBConnection(t)
	dbService := getDBService()

	// Ensure database is restarted even if test fails
	defer func() {
		if !dbService.IsRunning() {
			dbService.Start()
		}
	}()

	// Clear database before loading fixtures
	if err := utils.ClearTestDatabase(db); err != nil {
		t.Fatalf("Failed to clear test database: %v", err)
	}

	// Create a backup to restore from
	if err := utils.LoadSQLFile(db, "fixtures/initial_data.sql"); err != nil {
		t.Fatalf("Failed to load initial data: %v", err)
	}

	processID, err := utils.CreateBackup(token, "full", "", apiURL)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	processStatus, err := utils.WaitForBackupCompletion(token, processID, apiURL, 0)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	backupID := processStatus.ResourceID
	if backupID == "" {
		t.Fatal("Backup ID not found in process status")
	}

	// Close connection and stop database (but don't clear data directory)
	db.Close()
	if err := dbService.Stop(); err != nil {
		t.Fatalf("Failed to stop database: %v", err)
	}

	// Attempt restore - should fail with 503
	payload := map[string]string{
		"id":     backupID,
		"target": "database",
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/restore", apiURL), strings.NewReader(string(payloadBytes)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Restart database for cleanup
	dbService.Start()

	if resp.StatusCode != HTTPServiceUnavailable {
		t.Fatalf("Expected 503, got %d", resp.StatusCode)
	}

	bodyStr := strings.ToLower(string(body))
	if (!strings.Contains(bodyStr, "data") && !strings.Contains(bodyStr, "dir")) || !strings.Contains(bodyStr, "empty") {
		t.Fatalf("Expected error message to contain 'data'/'dir' and 'empty', got: %s", string(body))
	}
}

// TestListBackups tests listing backups via API
func TestListBackups(t *testing.T) {
	token := getAPIToken(t)
	apiURL := getAPIBaseURL(t)
	db := getDBConnection(t)
	defer db.Close()

	// Clear database before loading fixtures
	if err := utils.ClearTestDatabase(db); err != nil {
		t.Fatalf("Failed to clear test database: %v", err)
	}

	// Create multiple backups
	if err := utils.LoadSQLFile(db, "fixtures/initial_data.sql"); err != nil {
		t.Fatalf("Failed to load initial data: %v", err)
	}

	var backupIDs []string
	for i := 0; i < 2; i++ {
		processID, err := utils.CreateBackup(token, "full", "", apiURL)
		if err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}

		processStatus, err := utils.WaitForBackupCompletion(token, processID, apiURL, 0)
		if err != nil {
			t.Fatalf("Backup failed: %v", err)
		}

		backupID := processStatus.ResourceID
		if backupID != "" {
			backupIDs = append(backupIDs, backupID)
		}
	}

	// List backups
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/backups", apiURL), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != HTTPOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)

	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	backups, ok := responseData["items"].([]interface{})
	if !ok {
		t.Fatal("Items not found in response")
	}

	// Verify our backups are in the list
	foundCount := 0
	for _, backup := range backups {
		backupMap := backup.(map[string]interface{})
		id := backupMap["id"].(string)
		for _, expectedID := range backupIDs {
			if id == expectedID {
				foundCount++
				break
			}
		}
	}

	if foundCount < MinExpectedBackups {
		t.Fatalf("Expected at least %d backups in list, found %d", MinExpectedBackups, foundCount)
	}
}

// TestGetBackupDetails tests retrieving specific backup details
func TestGetBackupDetails(t *testing.T) {
	token := getAPIToken(t)
	apiURL := getAPIBaseURL(t)
	db := getDBConnection(t)
	defer db.Close()

	// Clear database before loading fixtures
	if err := utils.ClearTestDatabase(db); err != nil {
		t.Fatalf("Failed to clear test database: %v", err)
	}

	// Create a backup
	if err := utils.LoadSQLFile(db, "fixtures/initial_data.sql"); err != nil {
		t.Fatalf("Failed to load initial data: %v", err)
	}

	processID, err := utils.CreateBackup(token, "full", "", apiURL)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	processStatus, err := utils.WaitForBackupCompletion(token, processID, apiURL, 0)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	backupID := processStatus.ResourceID
	if backupID == "" {
		t.Fatal("Backup ID not found in process status")
	}

	// Get backup details
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/backups/%s", apiURL, backupID), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to get backup details: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != HTTPOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)

	var backup map[string]interface{}
	if err := json.Unmarshal(body, &backup); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if backup["id"].(string) != backupID {
		t.Fatalf("Expected backup ID %s, got %s", backupID, backup["id"])
	}

	// Verify it's a full backup (from_backup_id should be nil or not present)
	fromBackupID, exists := backup["from_backup_id"]
	if exists && fromBackupID != nil {
		t.Fatalf("Expected from_backup_id to be nil for full backup, got %v", fromBackupID)
	}
}
