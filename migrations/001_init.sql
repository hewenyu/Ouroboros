-- Initial database schema for DevOps Agent

-- Deployment logs table
CREATE TABLE IF NOT EXISTS deployment_logs (
    id TEXT PRIMARY KEY,
    trace_id INTEGER NOT NULL,
    commit_sha TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    branch TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    logs TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create index on commit_sha for fast lookups
CREATE INDEX IF NOT EXISTS idx_deployment_logs_commit_sha ON deployment_logs(commit_sha);

-- Create index on trace_id for workflow run lookups
CREATE INDEX IF NOT EXISTS idx_deployment_logs_trace_id ON deployment_logs(trace_id);

-- Create index on status for filtering
CREATE INDEX IF NOT EXISTS idx_deployment_logs_status ON deployment_logs(status);

-- Audit trail table for MCP tool invocations
CREATE TABLE IF NOT EXISTS audit_trails (
    id TEXT PRIMARY KEY,
    tool_name TEXT NOT NULL,
    parameters TEXT,
    result TEXT,
    caller_info TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create index on tool_name for audit filtering
CREATE INDEX IF NOT EXISTS idx_audit_trails_tool_name ON audit_trails(tool_name);

-- System configuration table
CREATE TABLE IF NOT EXISTS system_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
