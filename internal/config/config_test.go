package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Test default values
	cfg := Load()

	if cfg.ServerPort != "8080" {
		t.Errorf("Expected default ServerPort '8080', got '%s'", cfg.ServerPort)
	}

	if cfg.DatabasePath != "data.db" {
		t.Errorf("Expected default DatabasePath 'data.db', got '%s'", cfg.DatabasePath)
	}

	if cfg.DockerHost != "unix:///var/run/docker.sock" {
		t.Errorf("Expected default DockerHost 'unix:///var/run/docker.sock', got '%s'", cfg.DockerHost)
	}

	if !cfg.MCPEnabled {
		t.Errorf("Expected MCPEnabled to be true by default")
	}

	if cfg.MCPTransport != "sse" {
		t.Errorf("Expected default MCPTransport 'sse', got '%s'", cfg.MCPTransport)
	}
}

func TestLoadWithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("DATABASE_PATH", "/tmp/test.db")
	os.Setenv("GITHUB_SECRET", "test-secret")
	os.Setenv("MCP_ENABLED", "false")
	os.Setenv("MCP_TRANSPORT", "sse")
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DATABASE_PATH")
		os.Unsetenv("GITHUB_SECRET")
		os.Unsetenv("MCP_ENABLED")
		os.Unsetenv("MCP_TRANSPORT")
	}()

	cfg := Load()

	if cfg.ServerPort != "9090" {
		t.Errorf("Expected ServerPort '9090', got '%s'", cfg.ServerPort)
	}

	if cfg.DatabasePath != "/tmp/test.db" {
		t.Errorf("Expected DatabasePath '/tmp/test.db', got '%s'", cfg.DatabasePath)
	}

	if cfg.GitHubSecret != "test-secret" {
		t.Errorf("Expected GitHubSecret 'test-secret', got '%s'", cfg.GitHubSecret)
	}

	if cfg.MCPEnabled {
		t.Errorf("Expected MCPEnabled to be false")
	}

	if cfg.MCPTransport != "sse" {
		t.Errorf("Expected MCPTransport 'sse', got '%s'", cfg.MCPTransport)
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"true string", "true", false, true},
		{"false string", "false", true, false},
		{"1 string", "1", false, true},
		{"0 string", "0", true, false},
		{"invalid string", "invalid", true, true},
		{"empty string - use default true", "", true, true},
		{"empty string - use default false", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envKey := "TEST_BOOL_VAR"
			if tt.envValue != "" {
				os.Setenv(envKey, tt.envValue)
				defer os.Unsetenv(envKey)
			} else {
				os.Unsetenv(envKey)
			}

			result := getEnvBool(envKey, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvBool(%s, %v) = %v, expected %v", tt.envValue, tt.defaultValue, result, tt.expected)
			}
		})
	}
}
