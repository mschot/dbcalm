package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	ProjectName  string `mapstructure:"project_name"`
	DatabasePath string `mapstructure:"database_path"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Set defaults
	v.SetDefault("project_name", "dbcalm")
	v.SetDefault("database_path", "/var/lib/dbcalm/db.sqlite3")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Value(key string) string {
	switch key {
	case "project_name":
		return c.ProjectName
	case "database_path":
		return c.DatabasePath
	default:
		return ""
	}
}
