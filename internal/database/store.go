// Package database provides SQLite database operations with GORM.
package database

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DeploymentStatus represents the status of a deployment.
type DeploymentStatus string

const (
	DeploymentStatusPending  DeploymentStatus = "PENDING"
	DeploymentStatusVerified DeploymentStatus = "VERIFIED"
	DeploymentStatusFailed   DeploymentStatus = "FAILED"
)

// DeploymentLog represents a deployment record.
type DeploymentLog struct {
	ID        string           `gorm:"primaryKey;type:text"`
	TraceID   int64            `gorm:"index;not null"`
	CommitSHA string           `gorm:"index;not null;type:text"`
	RepoName  string           `gorm:"not null;type:text"`
	Branch    string           `gorm:"not null;type:text"`
	Status    DeploymentStatus `gorm:"index;not null;type:text;default:PENDING"`
	Logs      string           `gorm:"type:text"`
	CreatedAt time.Time        `gorm:"autoCreateTime"`
	UpdatedAt time.Time        `gorm:"autoUpdateTime"`
}

// TableName returns the table name for DeploymentLog.
func (DeploymentLog) TableName() string {
	return "deployment_logs"
}

// AuditTrail records MCP tool invocations for audit purposes.
type AuditTrail struct {
	ID         string    `gorm:"primaryKey;type:text"`
	ToolName   string    `gorm:"index;not null;type:text"`
	Parameters string    `gorm:"type:text"`
	Result     string    `gorm:"type:text"`
	CallerInfo string    `gorm:"type:text"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

// TableName returns the table name for AuditTrail.
func (AuditTrail) TableName() string {
	return "audit_trails"
}

// SystemConfig stores key-value configuration.
type SystemConfig struct {
	Key       string    `gorm:"primaryKey;type:text"`
	Value     string    `gorm:"not null;type:text"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for SystemConfig.
func (SystemConfig) TableName() string {
	return "system_config"
}

// Store wraps the GORM database connection.
type Store struct {
	db *gorm.DB
}

// NewSQLite creates a new SQLite database connection with WAL mode enabled.
func NewSQLite(dbPath string, migrationsFS embed.FS) (*Store, error) {
	// Configure SQLite connection with WAL mode and busy timeout
	dsn := fmt.Sprintf("file:%s?cache=shared&mode=rwc&_busy_timeout=5000&_journal_mode=WAL", dbPath)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable WAL mode explicitly
	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	store := &Store{db: db}

	// Run migrations
	if err := store.runMigrations(migrationsFS); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return store, nil
}

// runMigrations executes SQL migration files in order.
func (s *Store) runMigrations(migrationsFS embed.FS) error {
	// Get all SQL files from the migrations directory
	files, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		// No migrations directory embedded, use AutoMigrate instead
		log.Println("No embedded migrations found, using GORM AutoMigrate")
		return s.db.AutoMigrate(&DeploymentLog{}, &AuditTrail{}, &SystemConfig{})
	}

	// Sort files to ensure correct execution order
	var sqlFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}
	sort.Strings(sqlFiles)

	// Execute each migration file
	for _, filename := range sqlFiles {
		filePath := filepath.Join("migrations", filename)
		content, err := fs.ReadFile(migrationsFS, filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Execute the SQL
		if err := s.db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}
		log.Printf("Executed migration: %s", filename)
	}

	return nil
}

// DB returns the underlying GORM database connection.
func (s *Store) DB() *gorm.DB {
	return s.db
}

// CreateDeploymentLog creates a new deployment log record.
func (s *Store) CreateDeploymentLog(log *DeploymentLog) error {
	return s.db.Create(log).Error
}

// UpdateDeploymentLog updates an existing deployment log.
func (s *Store) UpdateDeploymentLog(log *DeploymentLog) error {
	return s.db.Save(log).Error
}

// GetDeploymentLogByID retrieves a deployment log by ID.
func (s *Store) GetDeploymentLogByID(id string) (*DeploymentLog, error) {
	var log DeploymentLog
	if err := s.db.First(&log, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// GetDeploymentLogByCommitSHA retrieves deployment logs by commit SHA.
func (s *Store) GetDeploymentLogByCommitSHA(sha string) ([]DeploymentLog, error) {
	var logs []DeploymentLog
	if err := s.db.Where("commit_sha = ?", sha).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// GetRecentDeployments retrieves the most recent deployment logs.
func (s *Store) GetRecentDeployments(limit int) ([]DeploymentLog, error) {
	var logs []DeploymentLog
	if err := s.db.Order("created_at DESC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// CreateAuditTrail creates a new audit trail record.
func (s *Store) CreateAuditTrail(trail *AuditTrail) error {
	return s.db.Create(trail).Error
}

// GetRecentAuditTrails retrieves the most recent audit trails.
func (s *Store) GetRecentAuditTrails(limit int) ([]AuditTrail, error) {
	var trails []AuditTrail
	if err := s.db.Order("created_at DESC").Limit(limit).Find(&trails).Error; err != nil {
		return nil, err
	}
	return trails, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
