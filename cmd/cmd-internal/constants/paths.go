package constants

// Configuration paths
const (
	// ConfigFile is the default path to the main configuration file
	ConfigFile = "/etc/dbcalm/config.yml"
)

// Socket paths
const (
	// SocketPath is the Unix domain socket path for IPC
	SocketPath = "/var/run/dbcalm/cmd.sock"

	// SocketDir is the directory containing the socket
	SocketDir = "/var/run/dbcalm"
)

// Log paths
const (
	// LogDir is the directory for log files
	LogDir = "/var/log/dbcalm"

	// LogFile is the path to the main log file
	LogFile = "/var/log/dbcalm/cmd.log"
)
