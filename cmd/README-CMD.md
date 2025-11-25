# DBCalm Command Service (Go Implementation)

Go implementation of the `dbcalm-cmd` service, which handles privileged system operations for the DBCalm backup system.

## Overview

The `dbcalm-cmd` service is a **root-privileged** command service that executes **whitelisted system operations** via Unix Domain Socket. It provides controlled root-level access for the main DBCalm application.

### Key Differences from dbcalm-db-cmd

| Aspect | dbcalm-cmd (Generic) | dbcalm-db-cmd (Database) |
|--------|---------------------|-------------------------|
| **Purpose** | System operations | Database backup/restore |
| **User** | root | mysql |
| **Socket** | `/var/run/dbcalm/cmd.sock` | `/var/run/dbcalm/db-cmd.sock` |
| **Commands** | 3 system commands | 3 database commands |
| **Dependencies** | None (system binaries) | mariabackup/xtrabackup, mysqladmin |

## Supported Commands

### 1. update_cron_schedules

Updates `/etc/cron.d/dbcalm` with backup schedules.

**Arguments:**
- `schedules` (list of schedule objects)

**Example Request:**
```json
{
  "cmd": "update_cron_schedules",
  "args": {
    "schedules": [
      {
        "id": 1,
        "backup_type": "full",
        "frequency": "daily",
        "hour": 2,
        "minute": 0,
        "enabled": true
      }
    ]
  }
}
```

**Process:**
1. Validates schedule data (frequency, time, backup_type, etc.)
2. Converts schedules to cron expressions using `CronFileBuilder`
3. Writes to temp file with proper formatting
4. Atomically moves to `/etc/cron.d/dbcalm` with chmod 644

### 2. delete_directory

Recursively deletes a directory.

**Arguments:**
- `path` (string) - Absolute path to directory

**Example Request:**
```json
{
  "cmd": "delete_directory",
  "args": {
    "path": "/var/backups/dbcalm/old-backup"
  }
}
```

**Executes:** `rm -rf <path>`

### 3. cleanup_backups

Deletes multiple backup folders in one operation.

**Arguments:**
- `backup_ids` (list of strings)
- `folders` (list of paths)

**Example Request:**
```json
{
  "cmd": "cleanup_backups",
  "args": {
    "backup_ids": ["backup-1", "backup-2"],
    "folders": ["/var/backups/dbcalm/backup-1", "/var/backups/dbcalm/backup-2"]
  }
}
```

**Executes:** `rm -rf <folder1> <folder2> ...`

## Architecture

```
cmd/cmd/main.go              # Entry point
cmd-db-cmd-internal/
├── adapter/                 # Adapter pattern
│   ├── adapter.go          # Interface
│   ├── system_commands.go  # Implementation
│   └── factory.go          # Factory
├── builder/                # Builder pattern
│   └── cron_file_builder.go # Cron expression generator
├── config/config.go        # YAML config
├── constants/paths.go      # Constants
├── handler/queue_handler.go # Queue processing
├── model/schedule.go       # Schedule model
├── process/                # Process management
│   ├── model.go            # Process struct
│   ├── runner.go           # Command execution
│   └── writer.go           # SQLite writer
├── socket/server.go        # UDS server
└── validator/validator.go # Validation
```

## Validation Rules

The validator checks:

**Command Whitelist:**
- Only 3 allowed commands: `update_cron_schedules`, `delete_directory`, `cleanup_backups`

**Required Arguments:**
- Each command must have its required args

**Schedule Validation (for update_cron_schedules):**
- **Required fields:** id, backup_type, frequency, enabled
- **Valid backup_types:** full, incremental
- **Valid frequencies:** daily, weekly, monthly, hourly, interval
- **Time validation:**
  - hour: 0-23
  - minute: 0-59
- **Day validation:**
  - day_of_week: 0-6 (for weekly)
  - day_of_month: 1-28 (for monthly)
- **Interval validation:**
  - interval_value: ≥1
  - interval_unit: minutes, hours

## Building

### Option 1: Using the build script

```bash
cd go-backend
./build-cmd.sh
```

### Option 2: Manual build

```bash
cd go-backend
go build -modfile=go-cmd.mod -o dbcalm-cmd ./cmd/cmd/main.go
```

### Option 3: Using go.mod symlink

```bash
cd go-backend
ln -sf go-cmd.mod go.mod
go build -o dbcalm-cmd ./cmd/cmd/main.go
rm go.mod  # Clean up symlink
```

## Installation

1. **Build the binary:**
   ```bash
   ./build-cmd.sh
   ```

2. **Install the binary:**
   ```bash
   sudo cp dbcalm-cmd /usr/bin/
   sudo chmod 755 /usr/bin/dbcalm-cmd
   ```

3. **Create configuration:**
   ```bash
   sudo mkdir -p /etc/dbcalm
   sudo cp cmd-config.example.yml /etc/dbcalm/cmd-config.yml
   sudo chown root:dbcalm /etc/dbcalm/cmd-config.yml
   sudo chmod 640 /etc/dbcalm/cmd-config.yml
   ```

4. **Install systemd service:**
   ```bash
   sudo cp dbcalm-cmd.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable dbcalm-cmd
   sudo systemctl start dbcalm-cmd
   ```

5. **Verify it's running:**
   ```bash
   sudo systemctl status dbcalm-cmd
   ls -la /var/run/dbcalm/cmd.sock
   ```

## Configuration

Default config location: `/etc/dbcalm/cmd-config.yml`

```yaml
# Project name (used for log paths and cron file paths)
project_name: dbcalm

# Path to SQLite database for process tracking
database_path: /var/lib/dbcalm/db.sqlite3
```

## Logs

- **Service logs:** `/var/log/dbcalm/cmd.log`
- **Systemd journal:** `journalctl -u dbcalm-cmd -f`

## Security Considerations

1. **Runs as root** - Necessary for:
   - Writing to `/etc/cron.d/`
   - Deleting system directories
   - Setting file permissions

2. **Whitelist-based** - Only 3 commands allowed
   - No arbitrary command execution
   - No shell injection vulnerabilities

3. **Path validation** - Should validate:
   - Delete operations only in allowed directories
   - No traversal attacks (../../etc/passwd)
   - Absolute paths only

4. **Socket permissions** - 0666 with parent directory control
   - API server can write as different user
   - Root reads and executes

5. **Atomic operations** - Cron updates use temp file + mv
   - Prevents partial writes
   - All-or-nothing updates

## Testing

### Test cron schedule update

```bash
# Create test request
cat > /tmp/test-cron.json <<'EOF'
{
  "cmd": "update_cron_schedules",
  "args": {
    "schedules": [
      {
        "id": 1,
        "backup_type": "full",
        "frequency": "daily",
        "hour": 2,
        "minute": 30,
        "enabled": true
      }
    ]
  }
}
EOF

# Send request via socket
cat /tmp/test-cron.json | nc -U /var/run/dbcalm/cmd.sock

# Check cron file
cat /etc/cron.d/dbcalm
```

### Test directory deletion

```bash
# Create test directory
sudo mkdir -p /tmp/test-delete-dir

# Create test request
cat > /tmp/test-delete.json <<'EOF'
{
  "cmd": "delete_directory",
  "args": {
    "path": "/tmp/test-delete-dir"
  }
}
EOF

# Send request
cat /tmp/test-delete.json | nc -U /var/run/dbcalm/cmd.sock

# Verify deletion
ls -la /tmp/test-delete-dir  # Should not exist
```

## Differences from Python Implementation

### Code Reuse
- **60% code reuse** from dbcalm-db-cmd:
  - Socket server (95% reusable)
  - Process runner (100% reusable)
  - Process model (100% reusable)
  - Process writer (100% reusable)
  - Config (80% reusable)

### New Components
- Validator (0% - completely different validation logic)
- Adapter (0% - completely different operations)
- CronFileBuilder (0% - new implementation)
- Handler (30% - simpler than db-cmd handler)

### Benefits
1. **Single Binary** - Easy deployment (no Python dependencies)
2. **Lower Memory** - Native Go vs Python interpreter
3. **Faster Startup** - Milliseconds vs seconds
4. **Better Logging** - Structured logging with Go
5. **Type Safety** - Compile-time error detection

### Estimated Metrics
- **Lines of Code:** ~800-1000 lines (vs ~1500 for Python)
- **Binary Size:** ~8-10 MB (vs ~50 MB for PyInstaller)
- **Memory Usage:** ~5-10 MB (vs ~30-50 MB for Python)
- **Startup Time:** <10ms (vs ~100-200ms for Python)

## Compatibility

This Go implementation is **100% compatible** with the Python client API:
- Same socket protocol
- Same JSON request/response format
- Same validation rules
- Same command semantics

No changes required to the DBCalm API server or frontend.

## Development

### Project Structure

```
go-backend/
├── cmd/
│   ├── db-cmd/main.go      # Existing DB service
│   └── cmd/main.go         # NEW: Generic cmd service
├── cmd-db-cmd-internal/           # NEW: cmd-specific packages
│   ├── adapter/
│   ├── builder/
│   ├── config/
│   ├── constants/
│   ├── handler/
│   ├── model/
│   ├── process/
│   ├── socket/
│   └── validator/
├── db-cmd-internal/               # Existing DB service packages
│   ├── adapter/
│   ├── builder/
│   ├── config/
│   ├── constants/
│   ├── handler/
│   ├── process/
│   ├── repository/
│   ├── socket/
│   └── validator/
├── go.mod                  # DB service dependencies
├── go-cmd.mod              # CMD service dependencies
├── build-cmd.sh            # Build script
├── dbcalm-cmd.service      # Systemd service
└── cmd-config.example.yml  # Example config
```

### Adding New Commands

1. **Add to validator whitelist** ([cmd-db-cmd-internal/validator/validator.go](cmd-db-cmd-internal/validator/validator.go)):
   ```go
   commands: map[string]map[string]string{
       "new_command": {
           "arg1": "required",
       },
   }
   ```

2. **Add to process type constants** ([cmd-db-cmd-internal/process/model.go](cmd-db-cmd-internal/process/model.go)):
   ```go
   const (
       TypeNewCommand = "new_command"
   )
   ```

3. **Implement in adapter** ([cmd-db-cmd-internal/adapter/system_commands.go](cmd-db-cmd-internal/adapter/system_commands.go)):
   ```go
   func (s *SystemCommands) NewCommand(args) (*process.Process, chan *process.Process, error) {
       // Implementation
   }
   ```

4. **Add to adapter interface** ([cmd-db-cmd-internal/adapter/adapter.go](cmd-db-cmd-internal/adapter/adapter.go)):
   ```go
   type Adapter interface {
       NewCommand(args) (*process.Process, chan *process.Process, error)
   }
   ```

5. **Handle in socket server** ([cmd-db-cmd-internal/socket/server.go](cmd-db-cmd-internal/socket/server.go)):
   ```go
   case "new_command":
       proc, procChan, err = s.adapter.NewCommand(args)
   ```

## Troubleshooting

### Service won't start
```bash
# Check logs
sudo journalctl -u dbcalm-cmd -n 50

# Check permissions
ls -la /var/run/dbcalm/
ls -la /var/log/dbcalm/

# Verify config
sudo cat /etc/dbcalm/cmd-config.yml
```

### Socket permission errors
```bash
# Fix socket directory permissions
sudo chmod 2774 /var/run/dbcalm/

# Verify socket exists
ls -la /var/run/dbcalm/cmd.sock
```

### Database errors
```bash
# Check database exists
ls -la /var/lib/dbcalm/db.sqlite3

# Check permissions
sudo chmod 664 /var/lib/dbcalm/db.sqlite3
sudo chown mysql:dbcalm /var/lib/dbcalm/db.sqlite3
```

## Contributing

When making changes:
1. Follow existing code patterns
2. Add tests for new functionality
3. Update this README
4. Maintain 100% Python client compatibility
5. Run validation before committing

## License

Same as DBCalm project.
