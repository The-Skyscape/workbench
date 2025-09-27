# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with this workbench application.

## Project Overview

**Skyscape Workbench** - A personal development environment that provides developers with a persistent, cloud-based workspace featuring integrated VS Code, repository management, and system monitoring. Built with TheSkyscape DevTools framework.

## License

**AGPL-3.0** - This project uses Coder (code-server) which requires AGPL licensing. All modifications must be open-sourced if deployed as a service.

## Design Philosophy

### HTMX/HATEOAS Architecture
We've rejected the complexity of modern JavaScript frameworks in favor of HTMX with HATEOAS principles:
- **HTML as the engine of application state** - The server sends HTML, not JSON
- **No client-side state management** - All state lives on the server
- **Progressive enhancement** - Works without JavaScript, enhanced with HTMX
- **Simplicity over features** - No webpack, no npm, no build pipeline for the frontend

### Value Receiver Pattern for Request Isolation
Our controllers use a unique pattern for request isolation without mutexes:
```go
// Value receiver creates a copy
func (c WorkbenchController) Handle(r *http.Request) application.Handler {
    c.Request = r  // Modifies the copy
    return &c      // Returns pointer to the copy
}
```
This gives each request its own controller instance (16-32 bytes overhead) with zero shared state.

### Template Validation with check-views
Templates are validated at build time using our `check-views` tool:
- Parses Go AST to find all controller methods
- Parses templates to find all references
- Validates that every template reference has a corresponding controller method
- Turns runtime template errors into build-time errors

### No Client State Principle
By eliminating client-side state, we've removed entire categories of bugs:
- No state synchronization issues
- No cache invalidation problems
- No version mismatches between API and client
- Debugging happens in one place: the server

## Architecture & Directory Rules

This application follows the TheSkyscape DevTools MVC pattern:

### Directory Structure (CRITICAL)
```
workbench/
├── controllers/        # HTTP handlers ONLY (no business logic!)
├── internal/          # Business logic ONLY (no HTTP!)
├── services/          # Docker containers ONLY
├── models/            # Data models ONLY (no business logic!)
├── views/             # Templates (access controllers only)
└── main.go           # Entry point
```

### Directory Responsibilities (NEVER VIOLATE)

#### controllers/
- **Purpose**: HTTP request/response handling ONLY
- **Files**:
  - `auth.go` - Single-user authentication using devtools auth Collection
  - `workbench.go` - Main dashboard, repository management
  - `monitoring.go` - System monitoring endpoints
- **Do**: Parse requests → Call internal/ → Render responses
- **Never**: Business logic, Git operations, SSH key generation

#### internal/
- **Purpose**: Business logic
- **Files**:
  - `repositories.go` - Git operations (clone, pull, delete)
  - `ssh.go` - SSH key generation and management
  - `monitoring.go` - System stats with DataDir disk monitoring
  - `activity.go` - Activity logging helpers
- **Do**: Business rules, Git operations, system monitoring
- **Never**: HTTP handling, request/response

#### services/
- **Purpose**: Docker container management ONLY
- **Files**:
  - `coder.go` - VS Code server (code-server) container
- **Pattern**: Wraps containers.Service from devtools
- **Do**: Start/stop container, health checks, proxy setup
- **Never**: Business logic, Git operations

#### models/
- **Purpose**: Data structures and repositories
- **Files**:
  - `repository.go` - Git repository tracking
  - `activity.go` - User activity logging
  - `settings.go` - Key-value settings store
- **Do**: Define structs, implement Table() method
- **Never**: Business logic, Git commands

### Package Architecture

**Directory Purposes:**
- **internal/** - Business logic and shared utilities (Git operations, SSH keys, monitoring)
- **controllers/** - HTTP request/response handling and routing
- **services/** - Docker container management (wraps devtools containers.Service)
- **models/** - Data structures with business methods
- **views/** - HTML templates that access controller methods

### Key Features

1. **Single-User System** - No multi-user support, one admin account
2. **Inline Authentication** - Auth pages render inline, no separate routes
3. **Repository Management** - Clone, sync, and manage Git repositories
4. **VS Code Integration** - Full IDE through code-server
5. **System Monitoring** - CPU, Memory, and DataDir disk usage
6. **Activity Tracking** - Logs all user and system actions
7. **SSH Key Management** - Auto-generates and manages SSH keys

## Development Commands

### Running the Application
```bash
# Set required environment variable
export AUTH_SECRET="your-super-secret-jwt-key"

# Run in development
go run .

# Build for production
go build -o workbench
```

### Testing
```bash
# Run all tests
go test ./...

# Run internal package tests
go test ./internal/...
```

### Deployment
```bash
# Build
go build -o workbench

# Deploy using launch-app
cd /home/coder/skyscape
./devtools/build/launch-app deploy \
  --name workbench-test-env \
  --binary workbench/workbench
```

## Environment Variables

### Required
- `AUTH_SECRET` - JWT signing secret (required for authentication)

### Optional
- `PORT` - Server port (default: 5000)
- `SSL_FULLCHAIN` - SSL certificate path
- `SSL_PRIVKEY` - SSL private key path

## Database Patterns

### Optimized Query Patterns
```go
// Use Find() for single records
setting, err := models.Settings.Find("WHERE Key = ?", key)

// Use Count() for existence checks
count := models.Repositories.Count("")
if count > 0 { ... }

// Use Search() for multiple records
activities, err := models.Activities.Search("ORDER BY CreatedAt DESC LIMIT 20")
```

### Model Methods
Models are kept simple with just fields and Table() method. Business logic lives in controllers or internal packages.

## Template Patterns

### Controller Access
```html
<!-- Access controller methods -->
{{workbench.GetRepositories}}
{{monitoring.GetDataDirStats}}
{{auth.CurrentUser}}
```

### HTMX Patterns
```html
<!-- Forms use HTMX for dynamic updates -->
<form hx-post="/repos/clone" hx-swap="none">

<!-- Auto-refresh monitoring -->
<div hx-get="/partials/stats" hx-trigger="every 10s" hx-swap="innerHTML">
```

## Security

- **Authentication**: JWT-based with secure cookies
- **Single-User**: Simplified security model, one admin
- **SSH Keys**: Stored in ~/.ssh within container
- **Passwords**: Automatically hashed with bcrypt
- **CSRF**: Protected via HTMX same-origin

## Monitoring

The application monitors:
- **CPU Usage** - System-wide CPU percentage
- **Memory** - System-wide RAM usage  
- **Data Storage** - Persistent DataDir disk usage (NOT system disk)
  - This shows only what persists between deployments
  - Located at `~/.skyscape/` or similar
  - Includes repositories, database, settings

## Testing Infrastructure

Tests use the devtools testutils package:
```go
// Test files exist for internal package
internal/monitoring_test.go - System monitoring tests
internal/repositories_test.go - Repository operations tests

// Run tests (skip Docker-dependent tests in CI)
go test ./internal/...
```

Note: Database-dependent tests are skipped until testutils supports test databases.

## Security Features

### Rate Limiting
- Authentication attempts limited to 5 per minute per IP
- In-memory rate limiter with automatic cleanup
- Configured in `internal/ratelimit.go`

### Structured Logging
- Log levels: DEBUG, INFO, WARN, ERROR
- Set via `LOG_LEVEL` environment variable
- Structured format with timestamps
- Configured in `internal/logger.go`

## UI Enhancements

### Loading Indicators
- HTMX operations show loading spinners
- Clone repository shows "Cloning..." with spinner
- Buttons disabled during operations

### Keyboard Shortcuts
- `Ctrl/Cmd + K`: Open VS Code
- `Ctrl/Cmd + N`: Clone new repository  
- `Escape`: Close modals
- Configured in `views/public/shortcuts.js`

## API Endpoints

### Health Check
- `GET /health` - Returns `{"status":"healthy"}` for monitoring

### Repository Management
- `POST /repos/clone` - Clone a new repository
- `POST /repos/pull/{name}` - Pull latest changes
- `POST /repos/delete/{name}` - Delete repository

### Authentication
- `POST /_auth/signup` - Create admin account (first time only)
- `POST /_auth/signin` - Sign in with rate limiting
- `POST /_auth/signout` - Sign out

### Monitoring
- `GET /partials/stats` - Get system stats (HTMX partial)
- `GET /partials/coder-status` - Get VS Code status

## Common Development Tasks

### Adding New Features
1. Add model if needed in `models/`
2. Add business logic in `internal/`
3. Add controller methods in `controllers/`
4. Create/update templates in `views/`

### Debugging Issues
```bash
# Check server logs
ssh root@SERVER_IP "docker logs sky-app --tail 50"

# Check database
ssh root@SERVER_IP 'sqlite3 ~/.skyscape/workbench.db "SELECT * FROM repositories;"'

# Restart application
ssh root@SERVER_IP "docker restart sky-app"
```

## Git Repository

- **Source**: https://github.com/The-Skyscape/workbench
- **License**: AGPL-3.0 (required due to Coder dependency)
- **Deployment**: https://test-bench.theskyscape.com

## Important Notes

1. **License Compliance**: Must remain AGPL-3.0 due to Coder usage
2. **Single User Only**: No multi-tenancy support by design
3. **DataDir Monitoring**: Disk usage shows persistent data only
4. **Test Mode**: Services skip initialization when running tests
5. **Activity Logging**: All actions are logged for audit trail