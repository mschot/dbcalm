package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	// Required fields
	BackupDir    string `mapstructure:"backup_dir"`
	DBType       string `mapstructure:"db_type"` // "mariadb" or "mysql"
	JWTSecretKey string `mapstructure:"jwt_secret_key"`

	// Optional API settings
	APIHost string `mapstructure:"api_host"`
	APIPort int    `mapstructure:"api_port"`

	// Optional SSL settings
	SSLCert string `mapstructure:"ssl_cert"`
	SSLKey  string `mapstructure:"ssl_key"`

	// Optional CORS settings
	CORSOrigins []string `mapstructure:"cors_origins"`

	// Optional logging settings
	LogFile  string `mapstructure:"log_file"`
	LogLevel string `mapstructure:"log_level"`

	// Optional JWT settings
	JWTAlgorithm string `mapstructure:"jwt_algorithm"`

	// Static paths
	ConfigPath           string
	MariaDBCmdSocketPath string
	CmdSocketPath        string
	DBPath               string
}

const (
	DefaultConfigPath           = "/etc/dbcalm/config.yml"
	DefaultMariaDBCmdSocketPath = "/var/run/dbcalm/db-cmd.sock"
	DefaultCmdSocketPath        = "/var/run/dbcalm/cmd.sock"
	DefaultDBPath               = "/var/lib/dbcalm/db.sqlite3"
	DefaultAPIHost              = "0.0.0.0"
	DefaultAPIPort              = 8335
	DefaultLogLevel             = "info"
	DefaultJWTAlgorithm         = "HS256"
)

func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set defaults
	viper.SetDefault("api_host", DefaultAPIHost)
	viper.SetDefault("api_port", DefaultAPIPort)
	viper.SetDefault("log_level", DefaultLogLevel)
	viper.SetDefault("jwt_algorithm", DefaultJWTAlgorithm)

	// Allow environment variable overrides
	viper.AutomaticEnv()
	viper.SetEnvPrefix("DBCALM")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set static paths
	cfg.ConfigPath = configPath
	cfg.MariaDBCmdSocketPath = DefaultMariaDBCmdSocketPath
	cfg.CmdSocketPath = DefaultCmdSocketPath
	cfg.DBPath = DefaultDBPath

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.BackupDir == "" {
		return fmt.Errorf("backup_dir is required")
	}

	if c.DBType == "" {
		return fmt.Errorf("db_type is required")
	}

	if c.DBType != "mariadb" && c.DBType != "mysql" {
		return fmt.Errorf("db_type must be 'mariadb' or 'mysql'")
	}

	if c.JWTSecretKey == "" {
		return fmt.Errorf("jwt_secret_key is required")
	}

	// Validate backup directory exists
	if _, err := os.Stat(c.BackupDir); os.IsNotExist(err) {
		return fmt.Errorf("backup_dir does not exist: %s", c.BackupDir)
	}

	// Validate SSL config if provided
	if c.SSLCert != "" || c.SSLKey != "" {
		if c.SSLCert == "" || c.SSLKey == "" {
			return fmt.Errorf("both ssl_cert and ssl_key must be provided")
		}
		if _, err := os.Stat(c.SSLCert); os.IsNotExist(err) {
			return fmt.Errorf("ssl_cert file does not exist: %s", c.SSLCert)
		}
		if _, err := os.Stat(c.SSLKey); os.IsNotExist(err) {
			return fmt.Errorf("ssl_key file does not exist: %s", c.SSLKey)
		}
	}

	return nil
}

func (c *Config) IsDevMode() bool {
	return os.Getenv("DBCALM_DEV_MODE") == "1"
}
