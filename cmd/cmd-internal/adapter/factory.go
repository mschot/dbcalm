package adapter

import (
	"github.com/martijn/dbcalm-cmd/cmd-internal/builder"
	"github.com/martijn/dbcalm-cmd/cmd-internal/config"
	sharedProcess "github.com/martijn/dbcalm/shared/process"
)

func NewAdapter(cfg *config.Config, runner *sharedProcess.Runner) Adapter {
	cronBuilder := builder.NewCronFileBuilder(cfg.ProjectName)
	return NewSystemCommands(runner, cronBuilder)
}
