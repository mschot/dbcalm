package adapter

import (
	"github.com/martijn/dbcalm-cmd/cmd-internal/model"
	sharedProcess "github.com/martijn/dbcalm/shared/process"
)

type Adapter interface {
	UpdateCronSchedules(schedules []model.Schedule) (*sharedProcess.Process, chan *sharedProcess.Process, error)
	DeleteDirectory(path string) (*sharedProcess.Process, chan *sharedProcess.Process, error)
	CleanupBackups(backupIDs []string, folders []string) (*sharedProcess.Process, chan *sharedProcess.Process, error)
}
