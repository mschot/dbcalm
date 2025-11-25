package adapter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/martijn/dbcalm-cmd/cmd-internal/builder"
	"github.com/martijn/dbcalm-cmd/cmd-internal/model"
	sharedProcess "github.com/martijn/dbcalm/shared/process"
	"github.com/martijn/dbcalm-cmd/cmd-internal/process"
)

type SystemCommands struct {
	runner      *sharedProcess.Runner
	cronBuilder *builder.CronFileBuilder
}

func NewSystemCommands(runner *sharedProcess.Runner, cronBuilder *builder.CronFileBuilder) *SystemCommands {
	return &SystemCommands{
		runner:      runner,
		cronBuilder: cronBuilder,
	}
}

// UpdateCronSchedules updates /etc/cron.d/dbcalm with all schedules.
//
// Writes complete cron file atomically by:
// 1. Building complete cron file content
// 2. Writing to temp file
// 3. Setting permissions
// 4. Moving atomically to /etc/cron.d/dbcalm
func (s *SystemCommands) UpdateCronSchedules(schedules []model.Schedule) (*sharedProcess.Process, chan *sharedProcess.Process, error) {
	// Build complete cron file content
	cronContent := s.cronBuilder.BuildCronFileContent(schedules)

	// Create temp file path
	tempFile := fmt.Sprintf("/tmp/dbcalm-cron-%s.tmp", uuid.New().String())
	targetFile := "/etc/cron.d/dbcalm"

	// Escape content for shell command (replace quotes with escaped quotes)
	escapedContent := strings.ReplaceAll(cronContent, `"`, `\"`)
	escapedContent = strings.ReplaceAll(escapedContent, `$`, `\$`)
	escapedContent = strings.ReplaceAll(escapedContent, "`", "\\`")

	// Write content to temp file, set permissions, then move atomically
	// Using shell to handle multi-step operation atomically
	command := []string{
		"/bin/sh",
		"-c",
		fmt.Sprintf(`echo "%s" > %s && chmod 644 %s && mv %s %s`,
			escapedContent, tempFile, tempFile, tempFile, targetFile),
	}

	args := map[string]interface{}{
		"schedule_count": len(schedules),
	}

	proc, procChan := s.runner.Execute(command, process.TypeUpdateCron, nil, args)
	return proc, procChan, nil
}

// DeleteDirectory deletes a directory and all its contents.
func (s *SystemCommands) DeleteDirectory(path string) (*sharedProcess.Process, chan *sharedProcess.Process, error) {
	// Use rm -rf to recursively delete directory
	// Runs with elevated permissions (root or sudo) via command service
	command := []string{
		"/bin/rm",
		"-rf",
		path,
	}

	args := map[string]interface{}{
		"path": path,
	}

	proc, procChan := s.runner.Execute(command, process.TypeDeleteDir, nil, args)
	return proc, procChan, nil
}

// CleanupBackups deletes multiple backup folders.
func (s *SystemCommands) CleanupBackups(backupIDs []string, folders []string) (*sharedProcess.Process, chan *sharedProcess.Process, error) {
	// Build command to delete all folders in a single rm call
	// This is more efficient than running separate commands
	command := []string{"/bin/rm", "-rf"}
	command = append(command, folders...)

	// Marshal backup_ids for args
	backupIDsJSON, _ := json.Marshal(backupIDs)
	args := map[string]interface{}{
		"backup_ids": string(backupIDsJSON),
	}

	proc, procChan := s.runner.Execute(command, process.TypeCleanupBackups, nil, args)
	return proc, procChan, nil
}
