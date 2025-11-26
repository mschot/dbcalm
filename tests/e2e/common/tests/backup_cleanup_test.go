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

	_ "github.com/mattn/go-sqlite3"
	"github.com/martijn/dbcalm/tests/e2e/common/utils"
)

const (
	ExpectedBackupCountInChain = 2
	BackupDir                  = "/var/backups/dbcalm"
	DBCalmDBPath               = "/var/lib/dbcalm/db.sqlite3"
)

// createScheduleViaAPI creates a schedule via API
func createScheduleViaAPI(t *testing.T, token, apiURL, backupType, frequency string, hour, minute, retentionValue int, retentionUnit string) int {
	payload := map[string]interface{}{
		"backup_type":     backupType,
		"frequency":       frequency,
		"hour":            hour,
		"minute":          minute,
		"retention_value": retentionValue,
		"retention_unit":  retentionUnit,
		"enabled":         true,
	}

	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/schedules", apiURL), strings.NewReader(string(payloadBytes)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to create schedule: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != HTTPOK && resp.StatusCode != 201 {
		t.Fatalf("Failed to create schedule: %s", string(body))
	}

	var scheduleData map[string]interface{}
	if err := json.Unmarshal(body, &scheduleData); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	scheduleID := int(scheduleData["id"].(float64))
	return scheduleID
}

// deleteScheduleViaAPI deletes a schedule via API
func deleteScheduleViaAPI(t *testing.T, token, apiURL string, scheduleID int) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/schedules/%d", apiURL, scheduleID), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to delete schedule: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != HTTPOK && resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to delete schedule: %s", string(body))
	}
}

// createBackupViaAPI creates a backup via API and waits for completion
func createBackupViaAPI(t *testing.T, token, apiURL, backupType, fromBackupID string) string {
	processID, err := utils.CreateBackup(token, backupType, fromBackupID, apiURL)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	processStatus, err := utils.WaitForBackupCompletion(token, processID, apiURL, 0)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	backupID, ok := processStatus.Metadata["resource_id"].(string)
	if !ok || backupID == "" {
		t.Fatal("Backup ID not found in process status")
	}

	return backupID
}

// updateBackupTimestamp updates backup timestamps directly in database
func updateBackupTimestamp(t *testing.T, db *sql.DB, backupID string, startTime, endTime time.Time) {
	_, err := db.Exec(`
		UPDATE backup
		SET start_time = ?, end_time = ?
		WHERE id = ?
	`, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), backupID)

	if err != nil {
		t.Fatalf("Failed to update backup timestamp: %v", err)
	}
}

// verifyBackupExists checks if backup exists in both database and filesystem
func verifyBackupExists(t *testing.T, db *sql.DB, backupID string) bool {
	// Check database
	var id string
	err := db.QueryRow("SELECT id FROM backup WHERE id = ?", backupID).Scan(&id)
	if err != nil {
		return false
	}

	// Check filesystem
	backupPath := filepath.Join(BackupDir, backupID)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// verifyBackupDeletedFromDB checks if backup is deleted from database
func verifyBackupDeletedFromDB(t *testing.T, db *sql.DB, backupID string) bool {
	var id string
	err := db.QueryRow("SELECT id FROM backup WHERE id = ?", backupID).Scan(&id)
	return err == sql.ErrNoRows
}

// verifyBackupDeletedFromFilesystem checks if backup is deleted from filesystem
func verifyBackupDeletedFromFilesystem(backupID string) bool {
	backupPath := filepath.Join(BackupDir, backupID)
	_, err := os.Stat(backupPath)
	return os.IsNotExist(err)
}

// verifyBackupDeleted checks if backup is deleted from both database and filesystem
func verifyBackupDeleted(t *testing.T, db *sql.DB, backupID string) bool {
	return verifyBackupDeletedFromDB(t, db, backupID) && verifyBackupDeletedFromFilesystem(backupID)
}

// getDBCalmDB creates a connection to the dbcalm SQLite database
func getDBCalmDB(t *testing.T) *sql.DB {
	// Give the system a moment to ensure tables exist
	time.Sleep(500 * time.Millisecond)

	db, err := sql.Open("sqlite3", DBCalmDBPath)
	if err != nil {
		t.Fatalf("Failed to connect to dbcalm database: %v", err)
	}

	// Verify backup table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='backup'").Scan(&tableName)
	if err != nil {
		t.Fatalf("backup table not found in %s: %v", DBCalmDBPath, err)
	}

	return db
}

// TestCleanupDeletesSingleExpiredBackups tests that cleanup deletes expired backups while preserving recent ones
func TestCleanupDeletesSingleExpiredBackups(t *testing.T) {
	token := getAPIToken(t)
	apiURL := getAPIBaseURL(t)

	// Step 1: Create schedule with 7-day retention via API
	scheduleID := createScheduleViaAPI(t, token, apiURL, "full", "daily", 3, 0, 7, "days")

	// Step 2: Create two full backups via API
	oldBackupID := createBackupViaAPI(t, token, apiURL, "full", "")
	recentBackupID := createBackupViaAPI(t, token, apiURL, "full", "")

	// Step 3: Connect to database and associate backups with schedule
	dbcalmDB := getDBCalmDB(t)
	defer dbcalmDB.Close()

	currentTime := time.Now().UTC()

	_, err := dbcalmDB.Exec("UPDATE backup SET schedule_id = ? WHERE id = ?", scheduleID, oldBackupID)
	if err != nil {
		t.Fatalf("Failed to update old backup schedule: %v", err)
	}

	_, err = dbcalmDB.Exec("UPDATE backup SET schedule_id = ? WHERE id = ?", scheduleID, recentBackupID)
	if err != nil {
		t.Fatalf("Failed to update recent backup schedule: %v", err)
	}

	// Step 4: Manually update old backup to 10 days ago via SQL
	oldTime := currentTime.Add(-10 * 24 * time.Hour)
	updateBackupTimestamp(t, dbcalmDB, oldBackupID, oldTime, oldTime)

	// Step 5: Run cleanup via API
	payload := map[string]int{"schedule_id": scheduleID}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/cleanup", apiURL), strings.NewReader(string(payloadBytes)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to start cleanup: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != HTTPAccepted {
		t.Fatalf("Failed to start cleanup: %s", string(body))
	}

	var cleanupData map[string]interface{}
	if err := json.Unmarshal(body, &cleanupData); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	processID, ok := cleanupData["pid"].(string)
	if !ok {
		t.Fatal("PID not found in cleanup response")
	}

	// Wait for cleanup to complete
	if _, err := utils.WaitForCleanupCompletion(token, processID, apiURL, 0); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Wait for queue handler to complete database operations
	time.Sleep(3 * time.Second)

	// Step 6: Verify results
	// Old backup should be deleted from filesystem
	if !verifyBackupDeletedFromFilesystem(oldBackupID) {
		t.Fatalf("Old backup %s should be deleted from filesystem", oldBackupID)
	}

	// Old backup should be deleted from database
	if !verifyBackupDeletedFromDB(t, dbcalmDB, oldBackupID) {
		t.Fatalf("Old backup %s should be deleted from database", oldBackupID)
	}

	// Recent backup should still exist
	if !verifyBackupExists(t, dbcalmDB, recentBackupID) {
		t.Fatal("Recent backup should still exist in both DB and filesystem")
	}

	// Cleanup: Delete ALL backups for this schedule
	_, err = dbcalmDB.Exec("DELETE FROM backup WHERE schedule_id = ?", scheduleID)
	if err != nil {
		t.Fatalf("Failed to cleanup backups: %v", err)
	}

	// Cleanup: delete the schedule via API
	deleteScheduleViaAPI(t, token, apiURL, scheduleID)
}

// TestCleanupRespectsBackupChains tests that cleanup respects backup chains based on retention
func TestCleanupRespectsBackupChains(t *testing.T) {
	token := getAPIToken(t)
	apiURL := getAPIBaseURL(t)

	// Step 1: Create schedule with 7-day retention via API
	scheduleID := createScheduleViaAPI(t, token, apiURL, "full", "daily", 3, 0, 7, "days")

	// Step 2: Create backup chain (full + incremental) via API
	fullBackupID := createBackupViaAPI(t, token, apiURL, "full", "")
	incrementalBackupID := createBackupViaAPI(t, token, apiURL, "incremental", fullBackupID)

	// Step 3: Connect to database and associate backups with schedule
	dbcalmDB := getDBCalmDB(t)
	defer dbcalmDB.Close()

	currentTime := time.Now().UTC()

	_, err := dbcalmDB.Exec("UPDATE backup SET schedule_id = ? WHERE id = ?", scheduleID, fullBackupID)
	if err != nil {
		t.Fatalf("Failed to update full backup schedule: %v", err)
	}

	_, err = dbcalmDB.Exec("UPDATE backup SET schedule_id = ? WHERE id = ?", scheduleID, incrementalBackupID)
	if err != nil {
		t.Fatalf("Failed to update incremental backup schedule: %v", err)
	}

	// Part A: Set only full backup to 10 days ago
	// Step 4: Update full backup timestamp via SQL
	oldTime := currentTime.Add(-10 * 24 * time.Hour)
	updateBackupTimestamp(t, dbcalmDB, fullBackupID, oldTime, oldTime)

	// Step 5: Run cleanup via API
	payload := map[string]int{"schedule_id": scheduleID}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/cleanup", apiURL), strings.NewReader(string(payloadBytes)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to start cleanup: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != HTTPAccepted {
		t.Fatalf("Failed to start cleanup: %s", string(body))
	}

	var cleanupDataA map[string]interface{}
	if err := json.Unmarshal(body, &cleanupDataA); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if processIDA, ok := cleanupDataA["pid"].(string); ok && processIDA != "" {
		if _, err := utils.WaitForCleanupCompletion(token, processIDA, apiURL, 0); err != nil {
			t.Fatalf("Cleanup failed: %v", err)
		}
	}

	// Step 6: Verify chain is preserved (incremental is still recent)
	if !verifyBackupExists(t, dbcalmDB, fullBackupID) {
		t.Fatal("Full backup should be preserved (chain has recent incremental)")
	}

	if !verifyBackupExists(t, dbcalmDB, incrementalBackupID) {
		t.Fatal("Incremental backup should be preserved")
	}

	// Part B: Set both backups to 10 days ago
	// Step 7: Update incremental backup timestamp via SQL
	updateBackupTimestamp(t, dbcalmDB, incrementalBackupID, oldTime, oldTime)

	time.Sleep(1 * time.Second)

	// Step 8: Run cleanup again via API
	req2, err := http.NewRequest("POST", fmt.Sprintf("%s/cleanup", apiURL), strings.NewReader(string(payloadBytes)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := httpClient().Do(req2)
	if err != nil {
		t.Fatalf("Failed to start cleanup: %v", err)
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)

	if resp2.StatusCode != HTTPAccepted {
		t.Fatalf("Failed to start cleanup: %s", string(body2))
	}

	var cleanupDataB map[string]interface{}
	if err := json.Unmarshal(body2, &cleanupDataB); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	processIDB, ok := cleanupDataB["pid"].(string)
	if !ok {
		t.Fatal("PID not found in cleanup response")
	}

	// Wait for cleanup to complete
	if _, err := utils.WaitForCleanupCompletion(token, processIDB, apiURL, 0); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Step 9: Verify entire chain is deleted
	// Full backup should be deleted from database
	if !verifyBackupDeletedFromDB(t, dbcalmDB, fullBackupID) {
		t.Fatalf("Full backup %s should be deleted from database", fullBackupID)
	}

	// Full backup should be deleted from filesystem
	if !verifyBackupDeletedFromFilesystem(fullBackupID) {
		t.Fatalf("Full backup %s should be deleted from filesystem", fullBackupID)
	}

	// Incremental backup should be deleted from database
	if !verifyBackupDeletedFromDB(t, dbcalmDB, incrementalBackupID) {
		t.Fatalf("Incremental backup %s should be deleted from database", incrementalBackupID)
	}

	// Incremental backup should be deleted from filesystem
	if !verifyBackupDeletedFromFilesystem(incrementalBackupID) {
		t.Fatalf("Incremental backup %s should be deleted from filesystem", incrementalBackupID)
	}

	// Cleanup: delete the schedule via API
	deleteScheduleViaAPI(t, token, apiURL, scheduleID)
}
