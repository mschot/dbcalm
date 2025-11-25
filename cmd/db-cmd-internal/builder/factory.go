package builder

import (
	"fmt"

	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/config"
)

func NewBuilder(cfg *config.Config) (Builder, error) {
	switch cfg.DbType {
	case "mariadb":
		version, err := DetectMariaDBVersion(cfg.BackupCredentialsFile)
		if err != nil {
			// Default to version that doesn't use --apply-log-only
			version = Version{Major: 10, Minor: 5, Patch: 0}
		}
		return NewMariadbBuilder(cfg, version), nil
	case "mysql":
		version, err := DetectMySQLVersion(cfg.BackupCredentialsFile)
		if err != nil {
			// Default version
			version = Version{Major: 8, Minor: 0, Patch: 0}
		}
		return NewMysqlBuilder(cfg, version), nil
	default:
		return nil, fmt.Errorf("unsupported db_type: %s", cfg.DbType)
	}
}
