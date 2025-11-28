package validator

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/config"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/constants"
)

const (
	StatusOK                 = 200
	StatusBadRequest         = 400
	StatusNotFound           = 404
	StatusConflict           = 409
	StatusServiceUnavailable = 503
)

type ValidationResult struct {
	Code    int
	Message string
}

type Validator struct {
	config *config.Config
}

func NewValidator(cfg *config.Config) *Validator {
	return &Validator{config: cfg}
}

func (v *Validator) Validate(cmd string, args map[string]interface{}) ValidationResult {
	// Check command is valid
	switch cmd {
	case "full_backup":
		return v.validateFullBackup(args)
	case "incremental_backup":
		return v.validateIncrementalBackup(args)
	case "restore_backup":
		return v.validateRestoreBackup(args)
	default:
		return ValidationResult{Code: StatusBadRequest, Message: fmt.Sprintf("Unknown command: %s", cmd)}
	}
}

func (v *Validator) validateFullBackup(args map[string]interface{}) ValidationResult {
	// Check required arguments
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return ValidationResult{Code: StatusBadRequest, Message: "Missing required argument: id"}
	}

	// Check backup ID is unique
	if v.backupExists(id) {
		return ValidationResult{Code: StatusConflict, Message: fmt.Sprintf("Backup with id '%s' already exists", id)}
	}

	// Check credentials file is valid
	if !v.credentialsFileValid() {
		return ValidationResult{Code: StatusServiceUnavailable, Message: "credentials file not found or missing [client-dbcalm] section"}
	}

	// Check server is alive
	if !v.serverAlive() {
		return ValidationResult{Code: StatusServiceUnavailable, Message: "cannot create backup, MySQL/MariaDB server is not running"}
	}

	return ValidationResult{Code: StatusOK, Message: ""}
}

func (v *Validator) validateIncrementalBackup(args map[string]interface{}) ValidationResult {
	// Check required arguments
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return ValidationResult{Code: StatusBadRequest, Message: "Missing required argument: id"}
	}

	fromBackupID, ok := args["from_backup_id"].(string)
	if !ok || fromBackupID == "" {
		return ValidationResult{Code: StatusBadRequest, Message: "Missing required argument: from_backup_id"}
	}

	// Check backup ID is unique
	if v.backupExists(id) {
		return ValidationResult{Code: StatusConflict, Message: fmt.Sprintf("Backup with id '%s' already exists", id)}
	}

	// Check base backup exists
	if !v.backupExists(fromBackupID) {
		return ValidationResult{Code: StatusNotFound, Message: fmt.Sprintf("Base backup with id '%s' not found", fromBackupID)}
	}

	// Check credentials file is valid
	if !v.credentialsFileValid() {
		return ValidationResult{Code: StatusServiceUnavailable, Message: "credentials file not found or missing [client-dbcalm] section"}
	}

	// Check server is alive
	if !v.serverAlive() {
		return ValidationResult{Code: StatusServiceUnavailable, Message: "cannot create backup, MySQL/MariaDB server is not running"}
	}

	return ValidationResult{Code: StatusOK, Message: ""}
}

func (v *Validator) validateRestoreBackup(args map[string]interface{}) ValidationResult {
	// Check required arguments
	idListRaw, ok := args["id_list"]
	if !ok {
		return ValidationResult{Code: StatusBadRequest, Message: "Missing required argument: id_list"}
	}

	// Convert id_list to []string
	var idList []string
	switch v := idListRaw.(type) {
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok {
				idList = append(idList, str)
			}
		}
	case []string:
		idList = v
	default:
		return ValidationResult{Code: StatusBadRequest, Message: "id_list must be an array of strings"}
	}

	if len(idList) == 0 {
		return ValidationResult{Code: StatusBadRequest, Message: "id_list cannot be empty"}
	}

	target, ok := args["target"].(string)
	if !ok || target == "" {
		return ValidationResult{Code: StatusBadRequest, Message: "Missing required argument: target"}
	}

	if target != "database" && target != "folder" {
		return ValidationResult{Code: StatusBadRequest, Message: "target must be 'database' or 'folder'"}
	}

	// Check all backups exist
	for _, id := range idList {
		if !v.backupExists(id) {
			return ValidationResult{Code: StatusNotFound, Message: fmt.Sprintf("Backup with id '%s' not found", id)}
		}
	}

	// For database restore, check server is stopped and data dir is empty
	if target == "database" {
		if v.serverAlive() {
			return ValidationResult{Code: StatusServiceUnavailable, Message: "cannot restore to database, MySQL/MariaDb server is not stopped"}
		}

		if !v.dataDirEmpty() {
			return ValidationResult{Code: StatusServiceUnavailable, Message: "cannot restore to database, mysql/mariadb data directory is not empty (usually /var/lib/mysql)"}
		}
	}

	return ValidationResult{Code: StatusOK, Message: ""}
}

func (v *Validator) credentialsFileValid() bool {
	file, err := os.Open(v.config.BackupCredentialsFile)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[client-dbcalm]" {
			return true
		}
	}

	return false
}

func (v *Validator) serverAlive() bool {
	var cmd *exec.Cmd

	if v.config.DbType == "mariadb" {
		cmd = exec.Command(constants.MariaDBAdminBin,
			fmt.Sprintf("--defaults-file=%s", v.config.BackupCredentialsFile),
			"--defaults-group-suffix=-dbcalm",
			"ping")
	} else {
		cmd = exec.Command(constants.MySQLAdminBin,
			fmt.Sprintf("--defaults-file=%s", v.config.BackupCredentialsFile),
			"--defaults-group-suffix=-dbcalm",
			"ping")
	}

	err := cmd.Run()
	return err == nil
}

func (v *Validator) dataDirEmpty() bool {
	entries, err := os.ReadDir(v.config.DataDir)
	if err != nil {
		return false
	}

	allowedFiles := map[string]bool{
		"ib_buffer_pool": true,
		"ibdata1":        true,
	}

	for _, entry := range entries {
		name := entry.Name()
		
		// Ignore allowed files
		if allowedFiles[name] {
			continue
		}
		
		// Ignore log files, sockets, pid files, error files
		if strings.HasPrefix(name, "ib_logfile") ||
			strings.HasSuffix(name, ".sock") ||
			strings.HasSuffix(name, ".pid") ||
			strings.HasSuffix(name, ".err") {
			continue
		}

		// Found a non-allowed file
		return false
	}

	return true
}

func (v *Validator) backupExists(id string) bool {
	backupPath := filepath.Join(v.config.BackupDir, id)
	_, err := os.Stat(backupPath)
	return err == nil
}
