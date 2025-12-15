// Package mcp provides Model Context Protocol (MCP) server implementation.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hewenyu/Ouroboros/internal/database"
	"github.com/hewenyu/Ouroboros/internal/docker"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server represents the MCP server for DevOps operations.
type Server struct {
	mcpServer *server.MCPServer
	store     *database.Store
	docker    *docker.Manager
}

// NewServer creates a new MCP server with DevOps tools and resources.
func NewServer(version string, store *database.Store, dockerMgr *docker.Manager) *Server {
	s := &Server{
		store:  store,
		docker: dockerMgr,
	}

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"DevOps-Agent",
		version,
		server.WithResourceCapabilities(true, false),
		server.WithToolCapabilities(true),
	)

	s.mcpServer = mcpServer
	s.registerTools()
	s.registerResources()

	return s
}

// registerTools registers all MCP tools.
func (s *Server) registerTools() {
	// Tool 1: check_deployment_health
	checkHealthTool := mcp.NewTool("check_deployment_health",
		mcp.WithDescription("Check the Docker deployment status and health for a service"),
		mcp.WithString("service_name",
			mcp.Required(),
			mcp.Description("The name of the service to check (Docker Compose service name)"),
		),
	)
	s.mcpServer.AddTool(checkHealthTool, s.handleCheckDeploymentHealth)

	// Tool 2: get_recent_deployments
	getDeploymentsTool := mcp.NewTool("get_recent_deployments",
		mcp.WithDescription("Get recent deployment records from the database"),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of deployments to return (default: 5)"),
		),
	)
	s.mcpServer.AddTool(getDeploymentsTool, s.handleGetRecentDeployments)

	// Tool 3: verify_commit_status
	verifyCommitTool := mcp.NewTool("verify_commit_status",
		mcp.WithDescription("Verify if a specific commit SHA has been deployed"),
		mcp.WithString("commit_sha",
			mcp.Required(),
			mcp.Description("The commit SHA to verify"),
		),
	)
	s.mcpServer.AddTool(verifyCommitTool, s.handleVerifyCommitStatus)

	// Tool 4: list_containers
	listContainersTool := mcp.NewTool("list_containers",
		mcp.WithDescription("List all Docker containers with their status"),
	)
	s.mcpServer.AddTool(listContainersTool, s.handleListContainers)

	// Tool 5: get_container_logs
	getLogsTool := mcp.NewTool("get_container_logs",
		mcp.WithDescription("Get logs from a Docker container"),
		mcp.WithString("container_id",
			mcp.Required(),
			mcp.Description("Container ID or name"),
		),
		mcp.WithString("tail",
			mcp.Description("Number of lines to show from end of logs (default: 100)"),
		),
	)
	s.mcpServer.AddTool(getLogsTool, s.handleGetContainerLogs)

	log.Println("Registered MCP tools: check_deployment_health, get_recent_deployments, verify_commit_status, list_containers, get_container_logs")
}

// registerResources registers all MCP resources.
func (s *Server) registerResources() {
	// Resource: system logs
	s.mcpServer.AddResource(mcp.NewResource(
		"logs://system/audit",
		"System Audit Log",
		mcp.WithResourceDescription("Recent MCP tool invocation audit trail"),
		mcp.WithMIMEType("application/json"),
	), s.handleAuditLogResource)

	// Resource: deployment summary
	s.mcpServer.AddResource(mcp.NewResource(
		"stats://deployments/summary",
		"Deployment Summary",
		mcp.WithResourceDescription("Summary of recent deployments"),
		mcp.WithMIMEType("application/json"),
	), s.handleDeploymentSummaryResource)

	log.Println("Registered MCP resources: logs://system/audit, stats://deployments/summary")
}

// handleCheckDeploymentHealth handles the check_deployment_health tool.
func (s *Server) handleCheckDeploymentHealth(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceName, err := request.RequireString("service_name")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: service_name"), nil
	}

	// Log the tool invocation
	s.logToolInvocation("check_deployment_health", map[string]interface{}{"service_name": serviceName})

	if s.docker == nil {
		return mcp.NewToolResultError("Docker manager not available"), nil
	}

	containers, err := s.docker.GetContainersByService(ctx, serviceName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error checking service: %v", err)), nil
	}

	if len(containers) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No containers found for service '%s'", serviceName)), nil
	}

	// Build status report
	var result string
	for _, c := range containers {
		result += fmt.Sprintf("Container: %s\n", c.ContainerName)
		result += fmt.Sprintf("  ID: %s\n", c.ContainerID)
		result += fmt.Sprintf("  State: %s\n", c.State)
		result += fmt.Sprintf("  Health: %s\n", c.Health)
		result += fmt.Sprintf("  Image: %s\n", c.ImageName)
		result += fmt.Sprintf("  Commit SHA: %s\n", c.CommitSHA)
		result += fmt.Sprintf("  Started: %s\n", c.StartedAt.Format(time.RFC3339))
		result += "\n"
	}

	return mcp.NewToolResultText(result), nil
}

// handleGetRecentDeployments handles the get_recent_deployments tool.
func (s *Server) handleGetRecentDeployments(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := request.GetInt("limit", 5)

	// Log the tool invocation
	s.logToolInvocation("get_recent_deployments", map[string]interface{}{"limit": limit})

	deployments, err := s.store.GetRecentDeployments(limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting deployments: %v", err)), nil
	}

	if len(deployments) == 0 {
		return mcp.NewToolResultText("No deployment records found"), nil
	}

	// Build deployment report
	var result string
	for i, d := range deployments {
		result += fmt.Sprintf("%d. Deployment %s\n", i+1, d.ID[:8])
		result += fmt.Sprintf("   Repository: %s\n", d.RepoName)
		result += fmt.Sprintf("   Branch: %s\n", d.Branch)
		result += fmt.Sprintf("   Commit: %s\n", d.CommitSHA[:8])
		result += fmt.Sprintf("   Status: %s\n", d.Status)
		result += fmt.Sprintf("   Time: %s\n", d.CreatedAt.Format(time.RFC3339))
		result += "\n"
	}

	return mcp.NewToolResultText(result), nil
}

// handleVerifyCommitStatus handles the verify_commit_status tool.
func (s *Server) handleVerifyCommitStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	commitSHA, err := request.RequireString("commit_sha")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: commit_sha"), nil
	}

	// Log the tool invocation
	s.logToolInvocation("verify_commit_status", map[string]interface{}{"commit_sha": commitSHA})

	// Check database for deployment records
	deployments, err := s.store.GetDeploymentLogByCommitSHA(commitSHA)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error checking commit: %v", err)), nil
	}

	if len(deployments) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No deployment records found for commit %s", commitSHA[:8])), nil
	}

	// Get the most recent deployment for this commit
	latest := deployments[0]
	result := fmt.Sprintf("Commit: %s\n", commitSHA)
	result += fmt.Sprintf("Repository: %s\n", latest.RepoName)
	result += fmt.Sprintf("Branch: %s\n", latest.Branch)
	result += fmt.Sprintf("Deployment Status: %s\n", latest.Status)
	result += fmt.Sprintf("Last Updated: %s\n", latest.UpdatedAt.Format(time.RFC3339))

	if latest.Logs != "" {
		result += fmt.Sprintf("\nLogs:\n%s\n", latest.Logs)
	}

	return mcp.NewToolResultText(result), nil
}

// handleListContainers handles the list_containers tool.
func (s *Server) handleListContainers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Log the tool invocation
	s.logToolInvocation("list_containers", nil)

	if s.docker == nil {
		return mcp.NewToolResultError("Docker manager not available"), nil
	}

	containers, err := s.docker.GetAllContainers(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error listing containers: %v", err)), nil
	}

	if len(containers) == 0 {
		return mcp.NewToolResultText("No containers found"), nil
	}

	// Build container list
	var result string
	for _, c := range containers {
		result += fmt.Sprintf("Container: %s (%s)\n", c.ContainerName, c.ContainerID)
		result += fmt.Sprintf("  Service: %s\n", c.ServiceName)
		result += fmt.Sprintf("  State: %s\n", c.State)
		result += fmt.Sprintf("  Health: %s\n", c.Health)
		result += fmt.Sprintf("  Image: %s\n", c.ImageName)
		result += "\n"
	}

	return mcp.NewToolResultText(result), nil
}

// handleGetContainerLogs handles the get_container_logs tool.
func (s *Server) handleGetContainerLogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	containerID, err := request.RequireString("container_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: container_id"), nil
	}

	tail := request.GetString("tail", "100")

	// Log the tool invocation
	s.logToolInvocation("get_container_logs", map[string]interface{}{"container_id": containerID, "tail": tail})

	if s.docker == nil {
		return mcp.NewToolResultError("Docker manager not available"), nil
	}

	logs, err := s.docker.GetContainerLogs(ctx, containerID, tail)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting logs: %v", err)), nil
	}

	if logs == "" {
		return mcp.NewToolResultText("No logs available"), nil
	}

	return mcp.NewToolResultText(logs), nil
}

// handleAuditLogResource handles the audit log resource.
func (s *Server) handleAuditLogResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	trails, err := s.store.GetRecentAuditTrails(50)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit trails: %w", err)
	}

	data, err := json.MarshalIndent(trails, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit trails: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "logs://system/audit",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

// handleDeploymentSummaryResource handles the deployment summary resource.
func (s *Server) handleDeploymentSummaryResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	deployments, err := s.store.GetRecentDeployments(100)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}

	// Build summary
	summary := map[string]interface{}{
		"total":           len(deployments),
		"by_status":       map[string]int{},
		"recent":          []map[string]interface{}{},
		"generated_at":    time.Now().Format(time.RFC3339),
	}

	statusCount := make(map[string]int)
	for _, d := range deployments {
		statusCount[string(d.Status)]++
	}
	summary["by_status"] = statusCount

	// Add recent deployments (limit to 10)
	recent := make([]map[string]interface{}, 0)
	for i, d := range deployments {
		if i >= 10 {
			break
		}
		recent = append(recent, map[string]interface{}{
			"id":         d.ID[:8],
			"repo":       d.RepoName,
			"branch":     d.Branch,
			"commit":     d.CommitSHA[:8],
			"status":     d.Status,
			"created_at": d.CreatedAt.Format(time.RFC3339),
		})
	}
	summary["recent"] = recent

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal summary: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "stats://deployments/summary",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

// logToolInvocation logs a tool invocation to the audit trail.
func (s *Server) logToolInvocation(toolName string, params map[string]interface{}) {
	paramsJSON, _ := json.Marshal(params)
	trail := &database.AuditTrail{
		ID:         uuid.New().String(),
		ToolName:   toolName,
		Parameters: string(paramsJSON),
		CreatedAt:  time.Now(),
	}

	if err := s.store.CreateAuditTrail(trail); err != nil {
		log.Printf("Failed to log tool invocation: %v", err)
	}
}

// MCPServer returns the underlying MCP server.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}
