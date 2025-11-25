package adapter

import (
	"fmt"

	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/builder"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/config"
	sharedProcess "github.com/martijn/dbcalm/shared/process"
)

func NewAdapter(cfg *config.Config, runner *sharedProcess.Runner) (Adapter, error) {
	// Create builder
	bldr, err := builder.NewBuilder(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create builder: %w", err)
	}

	// Both MariaDB and MySQL use the same adapter implementation
	// The difference is in the builder (MariabackupBuilder vs XtrabackupBuilder)
	return NewDatabaseAdapter(cfg, bldr, runner), nil
}
