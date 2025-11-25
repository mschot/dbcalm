# DBCalm Database Command Server (Go)

Go implementation of the DBCalm database command server for executing privileged backup and restore operations.
Supports both MariaDB (via mariabackup) and MySQL (via xtrabackup).

## Features

- Unix Domain Socket server for secure IPC
- Support for both MariaDB and MySQL (via XtraBackup)
- Full and incremental backups
- Database and folder restores
- Asynchronous command execution with goroutines
- Pre-flight validation checks
- SQLite-based process tracking

## Requirements

- Go 1.21 or higher
- MariaDB or MySQL server
- mariabackup or xtrabackup binaries
- SQLite3

## Building

```bash
# Build the binary
make build

# Build static binary (for deployment)
make build-static

# Install to /usr/local/bin
make install

# Run tests
make test

# Format and vet code
make fmt
make vet
```

## Configuration

The server reads configuration from `/etc/dbcalm/config.yml`:

```yaml
db_type: mariadb  # or mysql
backup_dir: /var/backups/dbcalm
backup_credentials_file: /etc/dbcalm/credentials.cnf
data_dir: /var/lib/mysql
database_path: /var/lib/dbcalm/db.sqlite3
stream: false
compression: ""  # gzip or zstd
forward: ""
host: localhost
```

### Credentials File

Create `/etc/dbcalm/credentials.cnf`:

```ini
[client-dbcalm]
user = dbcalm_backup
password = your_password
socket = /var/run/mysqld/mysqld.sock
```

## Running

```bash
# Run as root or mysql user
sudo ./dbcalm-db-cmd
```

The server creates a Unix Domain Socket at `/var/run/dbcalm/db-cmd.sock`.

## Communication Protocol

The server accepts JSON commands via the socket:

### Full Backup

```json
{
  "cmd": "full_backup",
  "args": {
    "id": "backup-2024-11-22",
    "schedule_id": 123
  }
}
```

### Incremental Backup

```json
{
  "cmd": "incremental_backup",
  "args": {
    "id": "backup-2024-11-22-incr",
    "from_backup_id": "backup-2024-11-22",
    "schedule_id": 123
  }
}
```

### Restore Backup

```json
{
  "cmd": "restore_backup",
  "args": {
    "id_list": ["backup-2024-11-22", "backup-2024-11-22-incr"],
    "target": "database"
  }
}
```

### Response

```json
{
  "code": 202,
  "status": "Accepted",
  "id": "uuid-command-id"
}
```

## Python Client Compatibility

This Go server is fully compatible with the existing Python client (note: the Python client is named `dbcalm_mariadb_cmd_client` for historical reasons, but it works with both MariaDB and MySQL):

```python
from dbcalm_mariadb_cmd_client import Client

client = Client()
response = client.command("full_backup", {"id": "backup-2024-11-22"})
# Returns: {"code": 202, "status": "Accepted", "id": "uuid"}
```

## Architecture

- **cmd/db-cmd**: Main entry point
- **db-cmd-internal/adapter**: Database-agnostic backup/restore adapter (works with both MariaDB and MySQL)
- **db-cmd-internal/builder**: Command builders for mariabackup (MariaDB) and xtrabackup (MySQL)
- **db-cmd-internal/config**: YAML configuration loader
- **db-cmd-internal/handler**: Queue handler for processing completed operations
- **db-cmd-internal/repository**: Database repositories for backup/restore records
- **db-cmd-internal/socket**: Unix Domain Socket server
- **db-cmd-internal/validator**: Request validation and pre-flight checks
- **shared/process**: Shared process management (used by both db-cmd and cmd services)
- **shared/socket**: Shared socket infrastructure

## License

Same as DBCalm project
