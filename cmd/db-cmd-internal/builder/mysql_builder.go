package builder

import (
	"fmt"
	"os/exec"

	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/config"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/constants"
)

type MysqlBuilder struct {
	*MariadbBuilder
}

func NewMysqlBuilder(cfg *config.Config, version Version) *MysqlBuilder {
	return &MysqlBuilder{
		MariadbBuilder: NewMariadbBuilder(cfg, version),
	}
}

func (b *MysqlBuilder) executable() string {
	if b.config.BackupBin != "" {
		return b.config.BackupBin
	}
	return "/usr/bin/xtrabackup"
}

func (b *MysqlBuilder) BuildFullBackupCmd(id string) []string {
	// Use parent implementation but with xtrabackup executable
	cmd := b.MariadbBuilder.buildBackupCmd(id, "")
	// Replace mariabackup with xtrabackup
	if len(cmd) > 0 && cmd[0] != b.executable() {
		cmd[0] = b.executable()
	}
	return cmd
}

func (b *MysqlBuilder) BuildIncrementalBackupCmd(id, fromBackupID string) []string {
	cmd := b.MariadbBuilder.buildBackupCmd(id, fromBackupID)
	if len(cmd) > 0 && cmd[0] != b.executable() {
		cmd[0] = b.executable()
	}
	return cmd
}

func (b *MysqlBuilder) BuildRestoreCmds(tmpDir string, idList []string, target string) [][]string {
	commands := b.MariadbBuilder.BuildRestoreCmds(tmpDir, idList, target)
	
	// Replace mariabackup with xtrabackup in all commands
	for i, cmd := range commands {
		if len(cmd) > 0 {
			if cmd[0] == b.MariadbBuilder.executable() {
				commands[i][0] = b.executable()
			}
		}
	}

	// Add --datadir for copy-back command (MySQL specific)
	if target == string(RestoreTargetDatabase) && len(commands) > 0 {
		lastCmd := commands[len(commands)-1]
		if len(lastCmd) > 1 && lastCmd[1] == "--copy-back" {
			commands[len(commands)-1] = append(lastCmd, fmt.Sprintf("--datadir=%s", b.config.DataDir))
		}
	}

	return commands
}

func DetectMySQLVersion(credentialsFile string) (Version, error) {
	cmd := exec.Command(constants.MySQLAdminBin,
		fmt.Sprintf("--defaults-file=%s", credentialsFile),
		"--defaults-group-suffix=-dbcalm",
		"--version")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return Version{}, fmt.Errorf("failed to detect MySQL version: %w", err)
	}

	return parseVersion(string(output))
}
