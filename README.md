# Ouroboros - AI-Native DevOps Agent

Ouroboros is a single-binary Go service designed as an AI-native infrastructure hub. It integrates GitHub Webhook handling, GitHub Actions workflow data parsing, Docker container deployment verification, and SQLite-based data persistence. The system implements the Model Context Protocol (MCP), enabling AI models like Claude and ChatGPT to directly interact with infrastructure.

## Features

- **Single Binary Deployment**: All dependencies are compiled into a single executable with zero external dependencies
- **GitHub Webhook Integration**: Secure webhook handling with HMAC-SHA256 signature validation
- **Docker SDK Integration**: Native container status checking, health monitoring, and log retrieval
- **SQLite with WAL Mode**: High-performance local data persistence with concurrent read/write support
- **MCP Protocol Support**: Exposes DevOps tools and resources for AI agent interaction

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    DevOps Agent                              │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │  HTTP Server │  │ MCP Server   │  │ Event Processor  │   │
│  │  (Webhook)   │  │ (Stdio/SSE)  │  │ (Worker Pool)    │   │
│  └──────┬───────┘  └──────┬───────┘  └────────┬─────────┘   │
│         │                 │                    │             │
│         ▼                 ▼                    ▼             │
│  ┌────────────────────────────────────────────────────────┐ │
│  │                  Core Services                          │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐  │ │
│  │  │ Database │  │ Docker   │  │ Signature Validation │  │ │
│  │  │ (SQLite) │  │ Manager  │  │ (HMAC-SHA256)        │  │ │
│  │  └──────────┘  └──────────┘  └──────────────────────┘  │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21 or later
- Docker (optional, for container verification features)
- SQLite3 (for CGO compilation)

### Build

```bash
# Build the binary
make build

# Or build with static linking (for minimal containers)
make build-static
```

### Run

```bash
# Run with default settings (MCP on stdio)
./devops-agent

# Run with HTTP only (no MCP)
MCP_ENABLED=false ./devops-agent

# Run with SSE transport for remote MCP access
MCP_TRANSPORT=sse ./devops-agent
```

### Configuration

Configuration is done via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | `8080` |
| `GITHUB_SECRET` | GitHub webhook secret | (empty) |
| `DATABASE_PATH` | SQLite database file path | `data.db` |
| `DOCKER_HOST` | Docker daemon socket | `unix:///var/run/docker.sock` |
| `MCP_ENABLED` | Enable MCP server | `true` |
| `MCP_TRANSPORT` | MCP transport (stdio/sse) | `stdio` |
| `MCP_SSE_PORT` | SSE transport port | `8081` |

## API Endpoints

### HTTP Endpoints

- `POST /webhook` - GitHub webhook receiver
- `GET /health` - Health check endpoint
- `GET /version` - Version information

### MCP Tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `check_deployment_health` | Check Docker service status | `service_name` (required) |
| `get_recent_deployments` | Get recent deployment records | `limit` (default: 5) |
| `verify_commit_status` | Verify if a commit is deployed | `commit_sha` (required) |
| `list_containers` | List all Docker containers | - |
| `get_container_logs` | Get container logs | `container_id` (required), `tail` (default: 100) |

### MCP Resources

- `logs://system/audit` - System audit trail
- `stats://deployments/summary` - Deployment statistics summary

## GitHub Webhook Setup

1. Go to your repository settings → Webhooks → Add webhook
2. Set Payload URL to `http://your-server:8080/webhook`
3. Set Content type to `application/json`
4. Generate a secret and set it as `GITHUB_SECRET` environment variable
5. Select "Workflow runs" event

## Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Run with coverage
make coverage

# Lint code
make lint

# Format code
make fmt
```

## Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── internal/
│   ├── config/              # Configuration management
│   ├── database/            # SQLite database layer
│   ├── docker/              # Docker SDK integration
│   ├── mcp/                 # MCP protocol implementation
│   └── webhook/             # GitHub webhook handling
├── migrations/              # SQL migration files
├── Makefile                 # Build automation
└── README.md
```

## Security Considerations

- All webhook requests are validated using HMAC-SHA256 with constant-time comparison
- SQLite database uses WAL mode for concurrent access
- Docker socket access requires appropriate permissions
- MCP stdio transport keeps all communication local to the process

## License

See [LICENSE](LICENSE) file.
