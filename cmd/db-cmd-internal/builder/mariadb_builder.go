package builder

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/config"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/constants"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func (v Version) LessThan(other Version) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}
	return v.Patch < other.Patch
}

type MariadbBuilder struct {
	config  *config.Config
	version Version
}

func NewMariadbBuilder(cfg *config.Config, version Version) *MariadbBuilder {
	return &MariadbBuilder{
		config:  cfg,
		version: version,
	}
}

func (b *MariadbBuilder) executable() string {
	if b.config.BackupBin != "" {
		return b.config.BackupBin
	}
	return "/usr/bin/mariabackup"
}

func (b *MariadbBuilder) BuildFullBackupCmd(id string) []string {
	return b.buildBackupCmd(id, "")
}

func (b *MariadbBuilder) BuildIncrementalBackupCmd(id, fromBackupID string) []string {
	return b.buildBackupCmd(id, fromBackupID)
}

func (b *MariadbBuilder) buildBackupCmd(id, fromBackupID string) []string {
	cmd := []string{
		b.executable(),
		fmt.Sprintf("--defaults-file=%s", b.config.BackupCredentialsFile),
		"--defaults-group-suffix=-dbcalm",
		"--backup",
	}

	targetDir := filepath.Join(b.config.BackupDir, id)
	
	if b.config.Stream {
		cmd = append(cmd, "--stream=xbstream")
	} else {
		cmd = append(cmd, fmt.Sprintf("--target-dir=%s", targetDir))
	}

	cmd = append(cmd, fmt.Sprintf("--host=%s", b.config.Host))

	// Incremental backup
	if fromBackupID != "" {
		basedir := filepath.Join(b.config.BackupDir, fromBackupID)
		cmd = append(cmd, fmt.Sprintf("--incremental-basedir=%s", basedir))
	}

	// Handle stream output
	if b.config.Stream {
		outputFile := filepath.Join(b.config.BackupDir, fmt.Sprintf("backup-%s.xbstream", id))
		
		if b.config.Compression == "gzip" {
			outputFile += ".gz"
		} else if b.config.Compression == "zstd" {
			outputFile += ".zst"
		}

		// Build shell command string for stream pipeline
		cmdStr := strings.Join(cmd, " ")
		
		if b.config.Compression == "gzip" {
			cmdStr += " | gzip"
		} else if b.config.Compression == "zstd" {
			cmdStr += " | zstd - -c -T0"
		}

		if b.config.Forward != "" {
			cmdStr += " | " + b.config.Forward
		} else {
			cmdStr += " > " + outputFile
		}

		return []string{"sh", "-c", cmdStr}
	}

	return cmd
}

func (b *MariadbBuilder) BuildRestoreCmds(tmpDir string, idList []string, target string) [][]string {
	var commands [][]string
	
	fullBackupID := idList[0]
	fullBackupPath := filepath.Join(b.config.BackupDir, fullBackupID)
	tmpFullBackupPath := filepath.Join(tmpDir, fullBackupID)

	// Step 1: Copy full backup to tmp
	commands = append(commands, []string{
		"cp", "-r", fullBackupPath, tmpDir,
	})

	// Step 2: Prepare full backup
	prepareCmd := []string{
		b.executable(),
		"--prepare",
		fmt.Sprintf("--target-dir=%s", tmpFullBackupPath),
	}
	
	// Add --apply-log-only if there are incremental backups to follow
	if len(idList) > 1 && b.shouldUseApplyLogOnly() {
		prepareCmd = append(prepareCmd, "--apply-log-only")
	}
	
	commands = append(commands, prepareCmd)

	// Step 3: Apply incremental backups
	for i := 1; i < len(idList); i++ {
		incrID := idList[i]
		incrPath := filepath.Join(b.config.BackupDir, incrID)
		
		applyCmd := []string{
			b.executable(),
			"--prepare",
			fmt.Sprintf("--target-dir=%s", tmpFullBackupPath),
			fmt.Sprintf("--incremental-dir=%s", incrPath),
		}
		
		// Add --apply-log-only for all but the last incremental
		if i < len(idList)-1 && b.shouldUseApplyLogOnly() {
			applyCmd = append(applyCmd, "--apply-log-only")
		}
		
		commands = append(commands, applyCmd)
	}

	// Step 4: Copy back to database (only for database target)
	if target == string(RestoreTargetDatabase) {
		copyBackCmd := []string{
			b.executable(),
			"--copy-back",
			fmt.Sprintf("--target-dir=%s", tmpFullBackupPath),
		}
		commands = append(commands, copyBackCmd)
	}

	return commands
}

func (b *MariadbBuilder) shouldUseApplyLogOnly() bool {
	// MariaDB >= 10.2 doesn't use --apply-log-only
	return b.version.LessThan(Version{Major: 10, Minor: 2, Patch: 0})
}

func DetectMariaDBVersion(credentialsFile string) (Version, error) {
	cmd := exec.Command(constants.MariaDBAdminBin,
		fmt.Sprintf("--defaults-file=%s", credentialsFile),
		"--defaults-group-suffix=-dbcalm",
		"--version")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return Version{}, fmt.Errorf("failed to detect MariaDB version: %w", err)
	}

	return parseVersion(string(output))
}

func parseVersion(versionStr string) (Version, error) {
	// Example: "mariadb-admin  Ver 10.5.23-MariaDB for Linux on x86_64"
	re := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	
	if len(matches) < 4 {
		return Version{}, fmt.Errorf("could not parse version from: %s", versionStr)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return Version{Major: major, Minor: minor, Patch: patch}, nil
}
