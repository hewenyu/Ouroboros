# Ouroboros - AI-Native DevOps Agent

Ouroboros is a single-binary Go service designed as an AI-native infrastructure hub. It integrates GitHub Webhook handling, GitHub Actions workflow data parsing, Docker container deployment verification, and SQLite-based data persistence. The system implements the Model Context Protocol (MCP), enabling AI models like Claude and ChatGPT to directly interact with infrastructure.

## Features

- **Single Binary Deployment**: All dependencies are compiled into a single executable with zero external dependencies
- **GitHub Webhook Integration**: Secure webhook handling with HMAC-SHA256 signature validation
- **Docker SDK Integration**: Native container status checking, health monitoring, and log retrieval
- **SQLite with WAL Mode**: High-performance local data persistence with concurrent read/write support
- **MCP Protocol Support**: Exposes DevOps tools and resources for AI agent interaction
- **MCP Authentication**: Bearer token authentication for secure remote MCP access via SSE

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

### Installation

#### Download Pre-built Binaries

Download the latest release for your platform from the [Releases](https://github.com/hewenyu/Ouroboros/releases) page:

- **Linux**: `devops-agent_*_linux_amd64.tar.gz` or `devops-agent_*_linux_arm64.tar.gz`
- **macOS**: `devops-agent_*_darwin_amd64.tar.gz` (Intel) or `devops-agent_*_darwin_arm64.tar.gz` (Apple Silicon)
- **Windows**: `devops-agent_*_windows_amd64.zip`

Extract and run:
```bash
# Linux / macOS
tar -xzf devops-agent_*.tar.gz
sudo mv devops-agent /usr/local/bin/

# Windows
# Extract the zip file and add devops-agent.exe to your PATH
```

#### Build from Source

**Prerequisites:**
- Go 1.24.11 or later
- Docker (optional, for container verification features)
- SQLite3 (for CGO compilation)

```bash
# Build the binary
make build

# Or build with static linking (for minimal containers)
make build-static
```

### Run

```bash
# Run with default settings (MCP on SSE with authentication)
MCP_AUTH_TOKEN=your-secret-token ./devops-agent

# Run with HTTP only (no MCP)
MCP_ENABLED=false ./devops-agent

# Run with stdio transport for local MCP access
MCP_TRANSPORT=stdio ./devops-agent
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
| `MCP_TRANSPORT` | MCP transport (stdio/sse) | `sse` |
| `MCP_SSE_PORT` | SSE transport port | `8081` |
| `MCP_AUTH_TOKEN` | Bearer token for MCP SSE authentication | (empty) |

## AI Agent Integration

### Claude Code Integration

[Claude Code](https://docs.anthropic.com/en/docs/claude-code) supports MCP (Model Context Protocol) for connecting to external tools and data sources. To connect Ouroboros with Claude Code:

#### SSE Transport Configuration (Recommended)

1. Start Ouroboros with SSE transport and authentication:
   ```bash
   MCP_AUTH_TOKEN=your-secret-token ./devops-agent
   ```

2. Configure Claude Code to connect to Ouroboros by adding to your MCP settings:
   ```json
   {
     "mcpServers": {
       "ouroboros": {
         "url": "http://your-server:8081/sse",
         "transport": "sse",
         "headers": {
           "Authorization": "Bearer your-secret-token"
         }
       }
     }
   }
   ```

#### Stdio Transport Configuration (Local Development)

For local development, you can use stdio transport:

1. Configure Claude Code with stdio transport:
   ```json
   {
     "mcpServers": {
       "ouroboros": {
         "command": "/path/to/devops-agent",
         "args": [],
         "env": {
           "MCP_TRANSPORT": "stdio",
           "DATABASE_PATH": "/path/to/data.db"
         }
       }
     }
   }
   ```

### OpenAI Codex Integration

[OpenAI Codex](https://platform.openai.com/docs/guides/tools-remote-mcp) supports MCP for extending AI capabilities with external tools. To connect Ouroboros with Codex:

#### SSE Transport Configuration (Recommended)

1. Start Ouroboros with SSE transport and authentication:
   ```bash
   MCP_AUTH_TOKEN=your-secret-token ./devops-agent
   ```

2. Configure Codex to connect to Ouroboros by adding to your MCP configuration:
   ```json
   {
     "mcpServers": {
       "ouroboros": {
         "type": "sse",
         "url": "http://your-server:8081/sse",
         "headers": {
           "Authorization": "Bearer your-secret-token"
         }
       }
     }
   }
   ```

#### Stdio Transport Configuration (Local Development)

For local development with Codex CLI:

1. Configure Codex with stdio transport:
   ```json
   {
     "mcpServers": {
       "ouroboros": {
         "type": "stdio",
         "command": "/path/to/devops-agent",
         "args": [],
         "env": {
           "MCP_TRANSPORT": "stdio",
           "DATABASE_PATH": "/path/to/data.db"
         }
       }
     }
   }
   ```

### Available MCP Capabilities

Once connected, AI agents can use the following tools:

- **check_deployment_health**: Check Docker service status and health
- **get_recent_deployments**: Retrieve recent deployment records
- **verify_commit_status**: Verify if a specific commit has been deployed
- **list_containers**: List all Docker containers with their status
- **get_container_logs**: Get logs from a specific Docker container

And access these resources:

- **logs://system/audit**: System audit trail of tool invocations
- **stats://deployments/summary**: Summary statistics of deployments

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

### Creating a Release

See [.github/RELEASE.md](.github/RELEASE.md) for detailed release instructions.

Quick steps:
1. Update CHANGELOG.md
2. Create and push a tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```
3. GitHub Actions will automatically build and publish the release

Test the release build locally:
```bash
./scripts/test-release.sh
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
- MCP SSE transport supports Bearer token authentication via `MCP_AUTH_TOKEN`
  - **Important**: Always set `MCP_AUTH_TOKEN` when exposing MCP over SSE in production
  - Without authentication, anyone with network access can invoke MCP tools
- SQLite database uses WAL mode for concurrent access
- Docker socket access requires appropriate permissions
- MCP stdio transport keeps all communication local to the process

## License

See [LICENSE](LICENSE) file.
