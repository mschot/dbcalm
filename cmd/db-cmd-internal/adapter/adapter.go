package adapter

import (
	sharedProcess "github.com/martijn/dbcalm/shared/process"
)

type Adapter interface {
	FullBackup(id string, scheduleID *int) (*sharedProcess.Process, chan *sharedProcess.Process, error)
	IncrementalBackup(id, fromBackupID string, scheduleID *int) (*sharedProcess.Process, chan *sharedProcess.Process, error)
	RestoreBackup(idList []string, target string) (*sharedProcess.Process, chan *sharedProcess.Process, error)
}
