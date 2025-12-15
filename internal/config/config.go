// Package config provides configuration management for the DevOps Agent.
package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration.
type Config struct {
	// ServerPort is the HTTP server port
	ServerPort string

	// GitHubSecret is the secret used to validate GitHub webhook signatures
	GitHubSecret string

	// DatabasePath is the path to the SQLite database file
	DatabasePath string

	// DockerHost is the Docker daemon socket path (defaults to unix:///var/run/docker.sock)
	DockerHost string

	// MCPEnabled enables the MCP server interface
	MCPEnabled bool

	// MCPTransport is the MCP transport type (stdio or sse)
	MCPTransport string

	// MCPSSEPort is the port for SSE transport (if enabled)
	MCPSSEPort string

	// MCPAuthToken is the Bearer token for MCP SSE authentication (optional)
	MCPAuthToken string
}

// Load loads configuration from environment variables with sensible defaults.
func Load() *Config {
	cfg := &Config{
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		GitHubSecret: getEnv("GITHUB_SECRET", ""),
		DatabasePath: getEnv("DATABASE_PATH", "data.db"),
		DockerHost:   getEnv("DOCKER_HOST", "unix:///var/run/docker.sock"),
		MCPEnabled:   getEnvBool("MCP_ENABLED", true),
		MCPTransport: getEnv("MCP_TRANSPORT", "sse"),
		MCPSSEPort:   getEnv("MCP_SSE_PORT", "8081"),
		MCPAuthToken: getEnv("MCP_AUTH_TOKEN", ""),
	}
	return cfg
}

// getEnv returns an environment variable value or a default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool returns a boolean environment variable or a default.
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return b
	}
	return defaultValue
}
