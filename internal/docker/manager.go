// Package docker provides Docker SDK integration for container verification.
package docker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

const (
	// LabelCommitSHA is the standard OCI label for image revision.
	LabelCommitSHA = "org.opencontainers.image.revision"
	// LabelComposeService is the Docker Compose service label.
	LabelComposeService = "com.docker.compose.service"
)

// truncate safely truncates a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// Manager handles Docker operations.
type Manager struct {
	cli *client.Client
}

// ContainerStatus represents the status of a container.
type ContainerStatus struct {
	ContainerID   string
	ContainerName string
	ServiceName   string
	State         string
	Health        string
	ImageID       string
	ImageName     string
	CommitSHA     string
	StartedAt     time.Time
}

// NewManager creates a new Docker manager.
func NewManager() (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	return &Manager{cli: cli}, nil
}

// NewManagerWithHost creates a new Docker manager with a specific host.
func NewManagerWithHost(host string) (*Manager, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(host),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Manager{cli: cli}, nil
}

// GetContainersByService finds containers by service name (Docker Compose label).
func (m *Manager) GetContainersByService(ctx context.Context, serviceName string) ([]ContainerStatus, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("%s=%s", LabelComposeService, serviceName))

	containers, err := m.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var statuses []ContainerStatus
	for _, c := range containers {
		status, err := m.getContainerStatus(ctx, c.ID)
		if err != nil {
			continue
		}
		statuses = append(statuses, *status)
	}

	return statuses, nil
}

// GetAllContainers returns all containers with their status.
func (m *Manager) GetAllContainers(ctx context.Context) ([]ContainerStatus, error) {
	containers, err := m.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var statuses []ContainerStatus
	for _, c := range containers {
		status, err := m.getContainerStatus(ctx, c.ID)
		if err != nil {
			continue
		}
		statuses = append(statuses, *status)
	}

	return statuses, nil
}

// getContainerStatus gets detailed status for a container.
func (m *Manager) getContainerStatus(ctx context.Context, containerID string) (*ContainerStatus, error) {
	inspect, err := m.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	status := &ContainerStatus{
		ContainerID:   containerID[:12],
		ContainerName: strings.TrimPrefix(inspect.Name, "/"),
		ServiceName:   inspect.Config.Labels[LabelComposeService],
		State:         inspect.State.Status,
		ImageID:       inspect.Image[:12],
		ImageName:     inspect.Config.Image,
	}

	// Parse started time
	if inspect.State.StartedAt != "" {
		status.StartedAt, _ = time.Parse(time.RFC3339Nano, inspect.State.StartedAt)
	}

	// Get health status
	if inspect.State.Health != nil {
		status.Health = inspect.State.Health.Status
	} else {
		status.Health = "no healthcheck"
	}

	// Get commit SHA from image labels
	imageInspect, _, err := m.cli.ImageInspectWithRaw(ctx, inspect.Image)
	if err == nil && imageInspect.Config != nil {
		status.CommitSHA = imageInspect.Config.Labels[LabelCommitSHA]
	}

	return status, nil
}

// VerifyDeployment verifies that a container is running with the expected commit SHA.
func (m *Manager) VerifyDeployment(commitSHA string, serviceName string) (bool, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	containers, err := m.GetContainersByService(ctx, serviceName)
	if err != nil {
		return false, "", err
	}

	if len(containers) == 0 {
		return false, fmt.Sprintf("No containers found for service '%s'", serviceName), nil
	}

	for _, c := range containers {
		if c.State != "running" {
			return false, fmt.Sprintf("Container %s is not running (state: %s)", c.ContainerName, c.State), nil
		}

		if c.Health == "unhealthy" {
			return false, fmt.Sprintf("Container %s is unhealthy", c.ContainerName), nil
		}

		if c.CommitSHA != "" && c.CommitSHA != commitSHA {
			return false, fmt.Sprintf("Container %s has different commit SHA: expected %s, got %s",
				c.ContainerName, truncate(commitSHA, 8), truncate(c.CommitSHA, 8)), nil
		}

		if c.CommitSHA == commitSHA {
			return true, fmt.Sprintf("Container %s is running with correct commit SHA: %s",
				c.ContainerName, truncate(commitSHA, 8)), nil
		}
	}

	return false, "Could not verify commit SHA (no label found on image)", nil
}

// WaitForHealth waits for a container to become healthy.
func (m *Manager) WaitForHealth(ctx context.Context, containerID string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			inspect, err := m.cli.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}

			if inspect.State.Health == nil {
				return errors.New("container has no healthcheck configured")
			}

			switch inspect.State.Health.Status {
			case "healthy":
				return nil
			case "unhealthy":
				return errors.New("container is unhealthy")
			}
			// Continue waiting if "starting"
		}
	}
}

// GetContainerLogs retrieves logs from a container.
func (m *Manager) GetContainerLogs(ctx context.Context, containerID string, tail string) (string, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	}

	reader, err := m.cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	// Docker logs use a multiplexed stream format:
	// - First 4 bytes: stream type (1=stdout, 2=stderr)
	// - Next 4 bytes: payload size (big-endian uint32)
	// - Remaining bytes: actual log content
	var logs strings.Builder
	header := make([]byte, 8)
	for {
		// Read the 8-byte header
		_, err := reader.Read(header)
		if err != nil {
			break
		}

		// Get the payload size from bytes 4-7 (big-endian uint32)
		size := uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7])
		if size == 0 {
			continue
		}

		// Read the actual log content
		payload := make([]byte, size)
		n, err := reader.Read(payload)
		if n > 0 {
			logs.Write(payload[:n])
		}
		if err != nil {
			break
		}
	}

	return logs.String(), nil
}

// Close closes the Docker client.
func (m *Manager) Close() error {
	return m.cli.Close()
}
