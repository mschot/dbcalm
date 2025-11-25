# DBCalm Go Application

A Go rewrite of the DBCalm backup management system with clean architecture and proper separation of concerns.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Interfaces Layer                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐  │
│  │   CLI    │  │ REST API │  │  WebSocket (future)  │  │
│  │ (Cobra)  │  │  (Gin)   │  │                      │  │
│  └────┬─────┘  └────┬─────┘  └──────────┬───────────┘  │
│       │             │                   │               │
│       └─────────────┴───────────────────┘               │
└─────────────────────┼───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                 Core Application Layer                   │
│  ┌───────────────────────────────────────────────────┐  │
│  │              Business Services                     │  │
│  │  • AuthService (JWT, bcrypt)                      │  │
│  │  • ProcessService (background jobs, goroutines)   │  │
│  │  • BackupService (full/incremental backups)       │  │
│  │  • RestoreService (database/folder restore)       │  │
│  │  • ScheduleService (cron management)              │  │
│  │  • CleanupService (retention policy)              │  │
│  └───────────────────┬───────────────────────────────┘  │
│                      │                                   │
│  ┌───────────────────▼───────────────────────────────┐  │
│  │              Repositories                          │  │
│  │  • UserRepository                                  │  │
│  │  • ClientRepository                                │  │
│  │  • AuthCodeRepository                              │  │
│  │  • BackupRepository (with chain resolution)       │  │
│  │  • RestoreRepository                               │  │
│  │  • ScheduleRepository                              │  │
│  │  • ProcessRepository                               │  │
│  └───────────────────┬───────────────────────────────┘  │
└─────────────────────┼───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                   Adapters Layer                         │
│  ┌────────────────┐  ┌─────────────────────────────┐    │
│  │ MariaDB Adapter│  │ System Commands Adapter     │    │
│  │ • mariabackup  │  │ • cron file management      │    │
│  │ • mysqladmin   │  │ • directory deletion        │    │
│  └────────────────┘  └─────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                Infrastructure Layer                      │
│  • SQLite Database                                       │
│  • File System                                           │
│  • MariaDB/MySQL Server                                  │
└─────────────────────────────────────────────────────────┘
```

## Project Structure

```
go-app/
├── cmd/
│   ├── server/          # REST API server entry point
│   └── dbcalm/          # CLI entry point
├── internal/
│   ├── core/            # Core business logic (COMPLETED ✅)
│   │   ├── domain/      # Domain models with business methods
│   │   │   ├── user.go
│   │   │   ├── client.go
│   │   │   ├── auth_code.go
│   │   │   ├── backup.go
│   │   │   ├── restore.go
│   │   │   ├── schedule.go
│   │   │   └── process.go
│   │   ├── repository/  # Repository interfaces
│   │   │   ├── user_repository.go
│   │   │   ├── client_repository.go
│   │   │   ├── auth_code_repository.go
│   │   │   ├── backup_repository.go
│   │   │   ├── restore_repository.go
│   │   │   ├── schedule_repository.go
│   │   │   └── process_repository.go
│   │   └── service/     # Business services
│   │       ├── auth_service.go (COMPLETED ✅)
│   │       ├── process_service.go (COMPLETED ✅)
│   │       ├── backup_service.go (COMPLETED ✅)
│   │       ├── restore_service.go (TODO)
│   │       ├── schedule_service.go (TODO)
│   │       └── cleanup_service.go (TODO)
│   ├── adapter/         # External integrations (COMPLETED ✅)
│   │   ├── mariadb/     # MariaDB backup/restore adapter
│   │   │   └── adapter.go
│   │   └── system/      # System commands adapter
│   │       └── adapter.go
│   ├── infrastructure/  # Database implementation (COMPLETED ✅)
│   │   └── sqlite/      # SQLite repositories
│   │       ├── db.go
│   │       ├── user_repository.go
│   │       ├── client_repository.go
│   │       ├── auth_code_repository.go
│   │       ├── backup_repository.go
│   │       ├── restore_repository.go
│   │       ├── schedule_repository.go
│   │       └── process_repository.go
│   ├── api/             # Gin REST API layer (TODO)
│   │   ├── handler/     # HTTP handlers
│   │   ├── middleware/  # JWT auth, CORS, etc.
│   │   └── dto/         # Request/response models
│   └── cli/             # Cobra CLI commands (TODO)
├── pkg/
│   └── config/          # Shared configuration (COMPLETED ✅)
│       └── config.go
├── web/                 # Embedded frontend (future)
├── go.mod
└── go.sum
```

## Completed Features ✅

### Phase 1: Foundation & Domain
- ✅ Go module setup with proper structure
- ✅ Domain models with business logic
- ✅ Repository interface definitions
- ✅ Configuration management (Viper)

### Phase 2: Infrastructure
- ✅ SQLite database schema and initialization
- ✅ All repository implementations (7 repositories)
- ✅ JSON serialization for complex fields
- ✅ Backup chain resolution logic

### Phase 3: Adapters
- ✅ MariaDB Adapter
  - Full and incremental backups
  - Backup preparation and restore
  - Server status checks
  - Backup size calculation
- ✅ System Commands Adapter
  - Cron file management
  - Directory operations
  - Ownership changes

### Phase 4: Core Services
- ✅ AuthService
  - JWT token generation/validation
  - Bcrypt password hashing
  - Authorization code flow
  - Client credentials flow
- ✅ ProcessService
  - Background job execution with goroutines
  - Process queue monitoring
  - Orphaned process detection
  - Completion handlers
- ✅ BackupService
  - Full backup creation
  - Incremental backup creation
  - Backup chain retrieval
  - Prerequisite validation

## Remaining Work

### Phase 5: Complete Services
- [ ] RestoreService - Database and folder restore logic
- [ ] ScheduleService - Cron expression generation, schedule CRUD
- [ ] CleanupService - Retention policy evaluation

### Phase 6: CLI Interface
- [ ] Cobra CLI setup
- [ ] User management commands
- [ ] Client management commands
- [ ] Backup commands (for cron)
- [ ] Server start command

### Phase 7: REST API
- [ ] Gin server setup
- [ ] DTO definitions
- [ ] API handlers
- [ ] JWT middleware
- [ ] Error handling middleware

### Phase 8: Testing & Integration
- [ ] Unit tests for services
- [ ] Integration tests for repositories
- [ ] End-to-end tests

### Phase 9: Frontend Integration
- [ ] Embed React build
- [ ] Serve static files with SPA routing
- [ ] API and frontend on single port

### Phase 10: WebSocket Interface (Future)
- [ ] Real-time process status updates
- [ ] Live log streaming
- [ ] WebSocket authentication

## Key Design Decisions

### 1. Clean Architecture
- **Core business logic** is independent of frameworks
- **Interfaces define contracts** between layers
- **Dependencies point inward** (adapters → core ← interfaces)

### 2. Separation of Concerns
- **CLI and API both call the same services** (no more CLI→API calls!)
- **Services contain all business logic**
- **Handlers/commands are thin wrappers**

### 3. Background Processing
- **Goroutines** for async operations
- **Channels** for process completion
- **Context** for cancellation and timeouts
- **Queue monitor** for orphaned processes

### 4. Error Handling
- **Wrapped errors** with context
- **Consistent error messages**
- **Proper cleanup** on failures

### 5. Type Safety
- **Type-safe enums** for backup types, process status, etc.
- **Pointer fields** for optional values
- **Proper NULL handling** in SQL

## Dependencies

```go
require (
    github.com/spf13/cobra v1.8.0          // CLI framework
    github.com/spf13/viper v1.18.2         // Configuration
    github.com/gin-gonic/gin v1.10.0       // Web framework
    github.com/golang-jwt/jwt/v5 v5.2.0    // JWT tokens
    golang.org/x/crypto v0.19.0            // Bcrypt
    github.com/google/uuid v1.6.0          // UUID generation
    github.com/jmoiron/sqlx v1.3.5         // SQL extensions
    github.com/mattn/go-sqlite3 v1.14.22   // SQLite driver
    github.com/robfig/cron/v3 v3.0.1       // Cron expression handling
)
```

## Configuration

Configuration file: `/etc/dbcalm/config.yml`

```yaml
# Required
backup_dir: /var/backups/dbcalm
db_type: mariadb  # or "mysql"
jwt_secret_key: your-secret-key-here

# Optional
api_host: 0.0.0.0
api_port: 8335
log_file: /var/log/dbcalm/dbcalm.log
log_level: info
jwt_algorithm: HS256
cors_origins:
  - http://localhost:3000

# Optional SSL
ssl_cert: /path/to/cert.pem
ssl_key: /path/to/key.pem
```

## Usage (When Complete)

### CLI Commands

```bash
# Start API server
dbcalm server

# User management
dbcalm users add <username>
dbcalm users delete <username>
dbcalm users list

# Client management
dbcalm clients add <label>
dbcalm clients delete <client-id>
dbcalm clients list

# Backups (for cron)
dbcalm backup full
dbcalm backup incremental
dbcalm backup full --schedule-id 1

# Cleanup
dbcalm cleanup
dbcalm cleanup --schedule-id 1
```

### API Endpoints

```
POST   /auth/authorize      - Get auth code
POST   /auth/token          - Exchange code/credentials for JWT
GET    /backups             - List backups
POST   /backups             - Create backup
GET    /backups/{id}        - Get backup
POST   /restore             - Restore backup
GET    /restores            - List restores
GET    /schedules           - List schedules
POST   /schedules           - Create schedule
GET    /schedules/{id}      - Get schedule
PUT    /schedules/{id}      - Update schedule
DELETE /schedules/{id}      - Delete schedule
POST   /cleanup             - Trigger cleanup
GET    /processes           - List processes
GET    /status/{command_id} - Get process status
GET    /clients             - List clients
POST   /clients             - Create client
DELETE /clients/{id}        - Delete client
```

## Development

### Build

```bash
go build -o bin/dbcalm ./cmd/dbcalm
go build -o bin/server ./cmd/server
```

### Run

```bash
# CLI
./bin/dbcalm server

# Or directly
go run ./cmd/dbcalm server
```

### Test

```bash
go test ./...
```

## Migration from Python

This Go implementation is designed as a **drop-in replacement** for the Python version:

- ✅ Same SQLite database schema
- ✅ Same configuration file format
- ✅ Same CLI commands
- ✅ Same API endpoints
- ✅ Compatible with existing backups
- ✅ Compatible with existing cron schedules

## Advantages Over Python Version

1. **Better Performance** - Compiled binary, concurrent operations
2. **Lower Memory** - No interpreter overhead
3. **Type Safety** - Compile-time error checking
4. **Clean Architecture** - Proper separation of concerns
5. **No More CLI→API Calls** - CLI and API share business logic
6. **Better Concurrency** - Goroutines vs threads
7. **Single Binary** - Easy deployment
8. **Better Tooling** - Static analysis, refactoring, etc.

## License

[Your License Here]
