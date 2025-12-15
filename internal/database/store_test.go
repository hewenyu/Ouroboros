package database

import (
	"embed"
	"os"
	"testing"
)

func TestNewSQLite(t *testing.T) {
	// Create a temporary database file
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)
	defer os.Remove(tmpPath + "-wal")
	defer os.Remove(tmpPath + "-shm")

	// Create empty embed FS to trigger AutoMigrate
	var emptyFS embed.FS

	store, err := NewSQLite(tmpPath, emptyFS)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	// Verify the database is accessible
	if store.DB() == nil {
		t.Fatal("Expected non-nil database connection")
	}
}

func TestDeploymentLogCRUD(t *testing.T) {
	// Create a temporary database file
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)
	defer os.Remove(tmpPath + "-wal")
	defer os.Remove(tmpPath + "-shm")

	var emptyFS embed.FS
	store, err := NewSQLite(tmpPath, emptyFS)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	// Test Create
	log := &DeploymentLog{
		ID:        "test-id-1",
		TraceID:   12345,
		CommitSHA: "abc123def456",
		RepoName:  "owner/repo",
		Branch:    "main",
		Status:    DeploymentStatusPending,
	}

	if err := store.CreateDeploymentLog(log); err != nil {
		t.Fatalf("Failed to create deployment log: %v", err)
	}

	// Test Read by ID
	retrieved, err := store.GetDeploymentLogByID("test-id-1")
	if err != nil {
		t.Fatalf("Failed to get deployment log: %v", err)
	}

	if retrieved.CommitSHA != "abc123def456" {
		t.Errorf("Expected commit SHA 'abc123def456', got '%s'", retrieved.CommitSHA)
	}

	if retrieved.Status != DeploymentStatusPending {
		t.Errorf("Expected status PENDING, got '%s'", retrieved.Status)
	}

	// Test Update
	retrieved.Status = DeploymentStatusVerified
	retrieved.Logs = "Deployment successful"
	if err := store.UpdateDeploymentLog(retrieved); err != nil {
		t.Fatalf("Failed to update deployment log: %v", err)
	}

	// Verify update
	updated, err := store.GetDeploymentLogByID("test-id-1")
	if err != nil {
		t.Fatalf("Failed to get updated deployment log: %v", err)
	}

	if updated.Status != DeploymentStatusVerified {
		t.Errorf("Expected status VERIFIED, got '%s'", updated.Status)
	}

	if updated.Logs != "Deployment successful" {
		t.Errorf("Expected logs 'Deployment successful', got '%s'", updated.Logs)
	}

	// Test Read by CommitSHA
	logs, err := store.GetDeploymentLogByCommitSHA("abc123def456")
	if err != nil {
		t.Fatalf("Failed to get deployment logs by SHA: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 deployment log, got %d", len(logs))
	}

	// Test GetRecentDeployments
	recent, err := store.GetRecentDeployments(10)
	if err != nil {
		t.Fatalf("Failed to get recent deployments: %v", err)
	}

	if len(recent) != 1 {
		t.Errorf("Expected 1 recent deployment, got %d", len(recent))
	}
}

func TestAuditTrailCRUD(t *testing.T) {
	// Create a temporary database file
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)
	defer os.Remove(tmpPath + "-wal")
	defer os.Remove(tmpPath + "-shm")

	var emptyFS embed.FS
	store, err := NewSQLite(tmpPath, emptyFS)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	// Test Create
	trail := &AuditTrail{
		ID:         "audit-id-1",
		ToolName:   "check_deployment_health",
		Parameters: `{"service_name": "backend"}`,
		Result:     "success",
	}

	if err := store.CreateAuditTrail(trail); err != nil {
		t.Fatalf("Failed to create audit trail: %v", err)
	}

	// Test GetRecentAuditTrails
	trails, err := store.GetRecentAuditTrails(10)
	if err != nil {
		t.Fatalf("Failed to get recent audit trails: %v", err)
	}

	if len(trails) != 1 {
		t.Errorf("Expected 1 audit trail, got %d", len(trails))
	}

	if trails[0].ToolName != "check_deployment_health" {
		t.Errorf("Expected tool name 'check_deployment_health', got '%s'", trails[0].ToolName)
	}
}
