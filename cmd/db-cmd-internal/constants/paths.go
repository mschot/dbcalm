package constants

// Configuration paths
const (
	// ConfigFile is the default path to the main configuration file
	ConfigFile = "/etc/dbcalm/config.yml"
)

// Socket paths
const (
	// SocketPath is the Unix domain socket path for IPC
	SocketPath = "/var/run/dbcalm/db-cmd.sock"

	// SocketDir is the directory containing the socket
	SocketDir = "/var/run/dbcalm"
)

// Temporary directory paths
const (
	// TempRestorePrefix is the prefix for temporary restore directories
	// The full path will be TempRestorePrefix + UUID
	TempRestorePrefix = "/tmp/dbcalm-restore-"
)

// Database admin tool paths
const (
	// MariaDBAdminBin is the path to the mariadb-admin binary
	MariaDBAdminBin = "/usr/bin/mariadb-admin"

	// MySQLAdminBin is the path to the mysqladmin binary
	MySQLAdminBin = "/usr/bin/mysqladmin"
)

// Log paths
const (
	// LogDir is the directory for log files
	LogDir = "/var/log/dbcalm"

	// LogFile is the path to the main log file
	LogFile = "/var/log/dbcalm/dbcalm.log"
)
