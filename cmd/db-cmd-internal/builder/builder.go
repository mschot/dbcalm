package builder

type Builder interface {
	BuildFullBackupCmd(id string) []string
	BuildIncrementalBackupCmd(id, fromBackupID string) []string
	BuildRestoreCmds(tmpDir string, idList []string, target string) [][]string
}

type RestoreTarget string

const (
	RestoreTargetDatabase RestoreTarget = "database"
	RestoreTargetFolder   RestoreTarget = "folder"
)
