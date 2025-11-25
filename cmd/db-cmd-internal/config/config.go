package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	DbType                string `mapstructure:"db_type"`
	BackupDir             string `mapstructure:"backup_dir"`
	BackupCredentialsFile string `mapstructure:"backup_credentials_file"`
	BackupBin             string `mapstructure:"backup_bin"`
	DataDir               string `mapstructure:"data_dir"`
	Stream                bool   `mapstructure:"stream"`
	Compression           string `mapstructure:"compression"`
	Forward               string `mapstructure:"forward"`
	Host                  string `mapstructure:"host"`
	DatabasePath          string `mapstructure:"database_path"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Set defaults
	v.SetDefault("backup_credentials_file", "/etc/dbcalm/credentials.cnf")
	v.SetDefault("data_dir", "/var/lib/mysql")
	v.SetDefault("stream", false)
	v.SetDefault("compression", "")
	v.SetDefault("forward", "")
	v.SetDefault("host", "localhost")
	v.SetDefault("database_path", "/var/lib/dbcalm/db.sqlite3")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if cfg.DbType == "" {
		return nil, fmt.Errorf("db_type is required in config")
	}
	if cfg.DbType != "mariadb" && cfg.DbType != "mysql" {
		return nil, fmt.Errorf("db_type must be 'mariadb' or 'mysql', got: %s", cfg.DbType)
	}
	if cfg.BackupDir == "" {
		return nil, fmt.Errorf("backup_dir is required in config")
	}

	// Validate backup directory exists
	if _, err := os.Stat(cfg.BackupDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("backup directory does not exist: %s", cfg.BackupDir)
	}

	return &cfg, nil
}

func (c *Config) Value(key string) string {
	switch key {
	case "db_type":
		return c.DbType
	case "backup_dir":
		return c.BackupDir
	case "backup_credentials_file":
		return c.BackupCredentialsFile
	case "backup_bin":
		return c.BackupBin
	case "data_dir":
		return c.DataDir
	case "compression":
		return c.Compression
	case "forward":
		return c.Forward
	case "host":
		return c.Host
	case "database_path":
		return c.DatabasePath
	default:
		return ""
	}
}
