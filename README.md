# Skyscape Workbench

A personal development environment that provides developers with a persistent, cloud-based workspace featuring integrated VS Code, repository management, and system monitoring.

![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D%201.22-blue)
![Status](https://img.shields.io/badge/status-production-green)

## Features

### 🔐 Single-User Authentication
- Secure single-admin system
- JWT-based authentication with httpOnly cookies
- Inline auth pages (no separate routes)

### 📦 Repository Management
- Clone repositories from any Git source (GitHub, GitLab, Bitbucket, etc.)
- Pull latest changes with one click
- Manage multiple repositories
- Automatic SSH key generation

### 💻 Integrated VS Code
- Full VS Code experience via code-server
- Persistent workspace across sessions
- Access from any browser
- All your extensions and settings

### 📊 System Monitoring
- Real-time CPU and memory usage
- **Data Storage monitoring** (persistent data only, not system disk)
- Auto-refreshing stats every 10 seconds
- Clean visualization with progress bars

### 📝 Activity Tracking
- Track all repository operations
- Authentication events logging
- Chronological activity feed
- User action attribution

## Quick Start

### Prerequisites
- Go 1.22 or higher
- Linux environment (for deployment)
- DigitalOcean account (for cloud deployment)

### Development Setup

1. Clone the repository:
```bash
git clone https://github.com/The-Skyscape/workbench.git
cd workbench
```

2. Set environment variables:
```bash
export AUTH_SECRET="your-super-secret-jwt-key"
```

3. Run the application:
```bash
go run .
```

4. Open http://localhost:5000 in your browser

### Production Deployment

1. Build the application:
```bash
go build -o workbench
```

2. Deploy using launch-app:
```bash
cd /path/to/skyscape
./devtools/build/launch-app deploy \
  --name workbench-env \
  --binary workbench/workbench
```

## Architecture

Built with [TheSkyscape DevTools](https://github.com/The-Skyscape/devtools) MVC framework:

- **Controllers** - HTTP handlers with template access
- **Models** - Database entities with optimized queries
- **Internal** - Business logic and operations
- **Services** - External service integrations (Coder)
- **Views** - HTMX-powered templates with DaisyUI

## Technology Stack

- **Backend**: Go with DevTools MVC framework
- **Frontend**: HTMX + DaisyUI (Tailwind CSS)
- **Database**: SQLite with automatic migrations
- **IDE**: Code-server (VS Code in browser)
- **Deployment**: Docker containers on DigitalOcean

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AUTH_SECRET` | Yes | - | JWT signing secret |
| `PORT` | No | 5000 | Server port |
| `SSL_FULLCHAIN` | No | - | SSL certificate path |
| `SSL_PRIVKEY` | No | - | SSL private key path |

## API Routes

### Repository Management
- `POST /repos/clone` - Clone a new repository
- `POST /repos/pull/{name}` - Pull latest changes
- `POST /repos/delete/{name}` - Delete repository

### Authentication
- `POST /_auth/signup` - Create admin account (first time only)
- `POST /_auth/signin` - Sign in
- `POST /_auth/signout` - Sign out

### Monitoring
- `GET /partials/stats` - Get system stats (HTMX partial)
- `GET /partials/coder-status` - Get VS Code status

## Testing

Run all tests:
```bash
go test ./...
```

Run specific package tests:
```bash
go test ./internal/...
```

## License

This project is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).

**Why AGPL?** This project uses [Coder](https://github.com/coder/code-server) which is AGPL-licensed. The AGPL ensures that any modifications to the software, even when deployed as a network service, must be made available as open source.

See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## Deployment

Currently deployed at: https://test-bench.theskyscape.com

### System Requirements

- **Minimum**: 1 vCPU, 2GB RAM, 20GB Storage
- **Recommended**: 2 vCPU, 4GB RAM, 80GB Storage
- **OS**: Ubuntu 22.04 LTS or similar

## Security Considerations

- Single-user system (not designed for multi-tenancy)
- SSH keys are auto-generated and stored in container
- All passwords are bcrypt hashed
- JWT tokens stored in httpOnly cookies
- CSRF protection via HTMX same-origin policy

## Monitoring Details

The "Data Storage" metric specifically monitors the persistent data directory (`~/.skyscape/` or similar), NOT the entire system disk. This shows:
- Database files
- Repository data
- Settings and configuration
- SSH keys

This is the data that persists between deployments and server migrations.

## Support

For issues, questions, or suggestions:
- Open an issue on [GitHub](https://github.com/The-Skyscape/workbench/issues)
- Check the [CLAUDE.md](CLAUDE.md) file for development guidance

## Acknowledgments

- Built with [TheSkyscape DevTools](https://github.com/The-Skyscape/devtools)
- VS Code integration via [Coder](https://github.com/coder/code-server)
- UI components from [DaisyUI](https://daisyui.com)
- Dynamic updates with [HTMX](https://htmx.org)

---

**Part of the Skyscape Ecosystem** - Professional development tools for modern developers.