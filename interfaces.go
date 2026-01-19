package rolekit

import (
	"context"

	"github.com/fernandezvara/dbkit"
)

// Database defines the database operations interface for dependency injection
type Database interface {
	dbkit.IDB
}

// TransactionManager defines the transaction management interface
type TransactionManager interface {
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
	TransactionWithOptions(ctx context.Context, opts dbkit.TxOptions, fn func(ctx context.Context) error) error
	ReadOnlyTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// MigrationManager defines the migration management interface
type MigrationManager interface {
	Migrations() []dbkit.Migration
	RunMigrations(ctx context.Context) (*MigrationStatus, error)
	GetMigrationStatus(ctx context.Context) (*MigrationStatus, error)
	VerifyMigrationChecksums(ctx context.Context) (bool, error)
	RollbackToMigration(ctx context.Context, targetMigrationID string) error
}

// HealthMonitor defines the health monitoring interface
type HealthMonitor interface {
	Health(ctx context.Context) dbkit.HealthStatus
	IsHealthy(ctx context.Context) bool
	Ping(ctx context.Context) error
	GetPoolStats() dbkit.PoolStats
}

// PoolManager defines the connection pool management interface
type PoolManager interface {
	ConfigureConnectionPool(config PoolConfig) error
	GetConnectionPoolConfig() (*PoolConfig, error)
	OptimizeConnectionPool() error
	ResetConnectionPool() error
}

// BulkOperations defines the bulk operations interface
type BulkOperations interface {
	AssignMultiple(ctx context.Context, assignments []RoleAssignment) error
	RevokeMultiple(ctx context.Context, revocations []RoleRevocation) error
}

// QueryHelper defines the query helper interface
type QueryHelper interface {
	CheckExists(ctx context.Context, userID, role, scopeType, scopeID string) bool
	CountRoles(ctx context.Context, userID, scopeType, scopeID string) (int, error)
	CountAllRoles(ctx context.Context) (int, error)
}

// AuditLogger defines the audit logging interface
type AuditLogger interface {
	logAudit(ctx context.Context, entry *AuditEntry) error
}

// TransactionMonitor defines the transaction monitoring interface
type TransactionMonitor interface {
	GetTransactionMetrics() TransactionMetrics
	ResetTransactionMetrics()
	IsTransactionHealthy() bool
}
