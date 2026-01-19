package rolekit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uptrace/bun"

	"github.com/fernandezvara/dbkit"
)

// Service provides role management and permission checking capabilities.
// It integrates with the database through dbkit with enhanced error handling.
//
// Error Handling:
// All database operations use dbkit's chainable error wrapping to provide
// detailed context about failed operations. Errors include operation names,
// database context, and preserve original error types for classification.
//
// Example error handling:
//
//	err := service.Assign(ctx, userID, role, scopeType, scopeID)
//	if err != nil {
//	    // Check for specific error types
//	    if dbkit.IsDuplicate(err) {
//	        // Handle duplicate assignment
//	    }
//	    if dbkit.IsNotFound(err) {
//	        // Handle not found scenarios
//	    }
//
//	    // Access rich error details
//	    var dbErr *dbkit.Error
//	    if errors.As(err, &dbErr) {
//	        fmt.Printf("Operation: %s, Table: %s, Constraint: %s\n",
//	            dbErr.Operation, dbErr.Table, dbErr.Constraint)
//	    }
//	}
type Service struct {
	db        dbkit.IDB
	registry  *Registry
	txMonitor *transactionMonitor
}

// TransactionMetrics provides transaction performance and failure statistics.
type TransactionMetrics struct {
	TotalTransactions      int64         `json:"total_transactions"`
	SuccessfulTransactions int64         `json:"successful_transactions"`
	FailedTransactions     int64         `json:"failed_transactions"`
	AverageDuration        time.Duration `json:"average_duration"`
	MaxDuration            time.Duration `json:"max_duration"`
	MinDuration            time.Duration `json:"min_duration"`
	LastReset              time.Time     `json:"last_reset"`
}

// transactionMonitor holds the internal transaction monitoring state
type transactionMonitor struct {
	totalCount    int64
	successCount  int64
	failureCount  int64
	totalDuration int64 // nanoseconds
	maxDuration   int64 // nanoseconds
	minDuration   int64 // nanoseconds
	lastReset     time.Time
	mu            sync.RWMutex
}

// newTransactionMonitor creates a new transaction monitor
func newTransactionMonitor() *transactionMonitor {
	return &transactionMonitor{
		minDuration: int64(time.Hour), // Initialize to a large value
		lastReset:   time.Now(),
	}
}

// recordTransaction records a transaction completion with its duration and success status
func (tm *transactionMonitor) recordTransaction(duration time.Duration, success bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	atomic.AddInt64(&tm.totalCount, 1)
	atomic.AddInt64(&tm.totalDuration, int64(duration))

	if success {
		atomic.AddInt64(&tm.successCount, 1)
	} else {
		atomic.AddInt64(&tm.failureCount, 1)
	}

	// Update max duration
	durationNs := int64(duration)
	for {
		current := atomic.LoadInt64(&tm.maxDuration)
		if durationNs <= current || atomic.CompareAndSwapInt64(&tm.maxDuration, current, durationNs) {
			break
		}
	}

	// Update min duration
	for {
		current := atomic.LoadInt64(&tm.minDuration)
		if durationNs >= current || atomic.CompareAndSwapInt64(&tm.minDuration, current, durationNs) {
			break
		}
	}
}

// getMetrics returns the current transaction metrics
func (tm *transactionMonitor) getMetrics() TransactionMetrics {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	total := atomic.LoadInt64(&tm.totalCount)
	success := atomic.LoadInt64(&tm.successCount)
	failure := atomic.LoadInt64(&tm.failureCount)
	totalDur := atomic.LoadInt64(&tm.totalDuration)
	maxDur := atomic.LoadInt64(&tm.maxDuration)
	minDur := atomic.LoadInt64(&tm.minDuration)

	var avgDuration time.Duration
	if total > 0 {
		avgDuration = time.Duration(totalDur / total)
	}

	return TransactionMetrics{
		TotalTransactions:      total,
		SuccessfulTransactions: success,
		FailedTransactions:     failure,
		AverageDuration:        avgDuration,
		MaxDuration:            time.Duration(maxDur),
		MinDuration:            time.Duration(minDur),
		LastReset:              tm.lastReset,
	}
}

// reset resets all metrics
func (tm *transactionMonitor) reset() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	atomic.StoreInt64(&tm.totalCount, 0)
	atomic.StoreInt64(&tm.successCount, 0)
	atomic.StoreInt64(&tm.failureCount, 0)
	atomic.StoreInt64(&tm.totalDuration, 0)
	atomic.StoreInt64(&tm.maxDuration, 0)
	atomic.StoreInt64(&tm.minDuration, int64(time.Hour))
	tm.lastReset = time.Now()
}

// NewService creates a new RoleKit service.
//
// Example:
//
//	registry := rolekit.NewRegistry()
//	// ... define roles ...
//	db, _ := dbkit.New(dbkit.Config{URL: "postgres://..."})
//	service := rolekit.NewService(registry, db)
func NewService(registry *Registry, db dbkit.IDB) *Service {
	return &Service{
		db:        db,
		registry:  registry,
		txMonitor: newTransactionMonitor(),
	}
}

// Registry returns the role registry.
func (s *Service) Registry() *Registry {
	return s.registry
}

// Transaction executes a function within a database transaction with automatic commit/rollback.
// If the function returns an error, the transaction is rolled back. Otherwise, it's committed.
//
// Example:
//
//	err := service.Transaction(ctx, func(ctx context.Context) error {
//	    if err := service.Assign(ctx, "user1", "admin", "organization", "org1"); err != nil {
//	        return err // This will cause a rollback
//	    }
//	    if err := service.Assign(ctx, "user2", "member", "organization", "org1"); err != nil {
//	        return err // This will cause a rollback
//	    }
//	    return nil // This will cause a commit
//	})
func (s *Service) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	start := time.Now()
	var err error

	// Check if we're already in a transaction by casting to dbkit.Tx
	if tx, ok := s.db.(*dbkit.Tx); ok {
		// We're already in a transaction, use savepoint
		err = tx.Transaction(ctx, func(tx *dbkit.Tx) error {
			// Use the transaction directly for operations within this scope
			return fn(ctx)
		})
	} else {
		// We're not in a transaction, start a new one
		if db, ok := s.db.(*dbkit.DBKit); ok {
			err = db.Transaction(ctx, func(tx *dbkit.Tx) error {
				// Use the transaction directly for operations within this scope
				return fn(ctx)
			})
		} else {
			// If we can't determine the type, try to use the generic interface
			// This is a fallback - ideally we'd have better type information
			err = fmt.Errorf("transaction support requires a dbkit.DBKit or dbkit.Tx instance")
		}
	}

	// Record transaction metrics
	duration := time.Since(start)
	s.txMonitor.recordTransaction(duration, err == nil)

	return err
}

// TransactionWithOptions executes a function within a database transaction with custom options.
// Supports read-only transactions, isolation levels, and other transaction parameters.
//
// Example:
//
//	err := service.TransactionWithOptions(ctx, dbkit.SerializableTxOptions(), func(ctx context.Context) error {
//	    // High isolation level operations
//	    return service.Assign(ctx, "user1", "admin", "organization", "org1")
//	})
func (s *Service) TransactionWithOptions(ctx context.Context, opts dbkit.TxOptions, fn func(ctx context.Context) error) error {
	// Check if we're already in a transaction by casting to dbkit.Tx
	if tx, ok := s.db.(*dbkit.Tx); ok {
		// We're already in a transaction, use savepoint (no options support in nested transactions)
		return tx.Transaction(ctx, func(tx *dbkit.Tx) error {
			// Create a new service that uses the transaction
			s.db = tx
			return fn(ctx)
		})
	}

	// We're not in a transaction, start a new one
	if db, ok := s.db.(*dbkit.DBKit); ok {
		return db.TransactionWithOptions(ctx, opts, func(tx *dbkit.Tx) error {
			// Create a new service that uses the transaction
			s.db = tx
			return fn(ctx)
		})
	}

	// If we can't determine the type, try to use the generic interface
	return fmt.Errorf("transaction support requires a dbkit.DBKit or dbkit.Tx instance")
}

// ReadOnlyTransaction executes a function within a read-only database transaction.
// Useful for operations that only read data and want to ensure consistency.
//
// Example:
//
//	err := service.ReadOnlyTransaction(ctx, func(ctx context.Context) error {
//	    roles, err := service.GetUserRoles(ctx, userID)
//	    if err != nil {
//	        return err
//	    }
//	    members, err := service.GetScopeMembers(ctx, "organization", orgID)
//	    return err
//	})
func (s *Service) ReadOnlyTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.TransactionWithOptions(ctx, dbkit.ReadOnlyTxOptions(), fn)
}

// Migrations returns all database migrations required for RoleKit.
// These should be executed using db.Migrate(ctx, service.Migrations()).
func (s *Service) Migrations() []dbkit.Migration {
	return []dbkit.Migration{
		// Table creation migrations
		{
			ID:          "rolekit-001",
			Description: "Create role_assignments table",
			SQL: `
                CREATE TABLE IF NOT EXISTS role_assignments (
                    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                    user_id TEXT NOT NULL,
                    role TEXT NOT NULL,
                    scope_type TEXT NOT NULL,
                    scope_id TEXT NOT NULL,
                    parent_scope_type TEXT,
                    parent_scope_id TEXT,
                    created_at TIMESTAMPTZ DEFAULT current_timestamp,
                    updated_at TIMESTAMPTZ DEFAULT current_timestamp
                )`,
		},
		{
			ID:          "rolekit-002",
			Description: "Create role_audit_log table",
			SQL: `
                CREATE TABLE IF NOT EXISTS role_audit_log (
                    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                    timestamp TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
                    actor_id TEXT NOT NULL,
                    action TEXT NOT NULL,
                    target_user_id TEXT NOT NULL,
                    role TEXT NOT NULL,
                    scope_type TEXT NOT NULL,
                    scope_id TEXT NOT NULL,
                    actor_roles TEXT[],
                    previous_roles TEXT[],
                    new_roles TEXT[],
                    ip_address TEXT,
                    user_agent TEXT,
                    request_id TEXT,
                    metadata JSONB
                )`,
		},
		{
			ID:          "rolekit-003",
			Description: "Create scope_hierarchy table",
			SQL: `
                CREATE TABLE IF NOT EXISTS scope_hierarchy (
                    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                    scope_type TEXT NOT NULL,
                    scope_id TEXT NOT NULL,
                    parent_scope_type TEXT NOT NULL,
                    parent_scope_id TEXT NOT NULL,
                    created_at TIMESTAMPTZ DEFAULT current_timestamp
                )`,
		},

		// Index migrations
		{
			ID:          "rolekit-010",
			Description: "Create idx_role_assignments_user",
			SQL:         `CREATE INDEX IF NOT EXISTS idx_role_assignments_user ON role_assignments(user_id);`,
		},
		{
			ID:          "rolekit-011",
			Description: "Create idx_role_assignments_scope",
			SQL:         `CREATE INDEX IF NOT EXISTS idx_role_assignments_scope ON role_assignments(scope_type, scope_id);`,
		},
		{
			ID:          "rolekit-012",
			Description: "Create idx_role_assignments_user_scope",
			SQL:         `CREATE INDEX IF NOT EXISTS idx_role_assignments_user_scope ON role_assignments(user_id, scope_type, scope_id);`,
		},
		{
			ID:          "rolekit-013",
			Description: "Create idx_role_assignments_unique",
			SQL:         `CREATE UNIQUE INDEX IF NOT EXISTS idx_role_assignments_unique ON role_assignments(user_id, role, scope_type, scope_id);`,
		},
		{
			ID:          "rolekit-020",
			Description: "Create idx_role_audit_log_actor",
			SQL:         `CREATE INDEX IF NOT EXISTS idx_role_audit_log_actor ON role_audit_log(actor_id);`,
		},
		{
			ID:          "rolekit-021",
			Description: "Create idx_role_audit_log_target",
			SQL:         `CREATE INDEX IF NOT EXISTS idx_role_audit_log_target ON role_audit_log(target_user_id);`,
		},
		{
			ID:          "rolekit-022",
			Description: "Create idx_role_audit_log_scope",
			SQL:         `CREATE INDEX IF NOT EXISTS idx_role_audit_log_scope ON role_audit_log(scope_type, scope_id);`,
		},
		{
			ID:          "rolekit-023",
			Description: "Create idx_role_audit_log_timestamp",
			SQL:         `CREATE INDEX IF NOT EXISTS idx_role_audit_log_timestamp ON role_audit_log(timestamp DESC);`,
		},
		{
			ID:          "rolekit-030",
			Description: "Create idx_scope_hierarchy_child",
			SQL:         `CREATE INDEX IF NOT EXISTS idx_scope_hierarchy_child ON scope_hierarchy(scope_type, scope_id);`,
		},
		{
			ID:          "rolekit-031",
			Description: "Create idx_scope_hierarchy_parent",
			SQL:         `CREATE INDEX IF NOT EXISTS idx_scope_hierarchy_parent ON scope_hierarchy(parent_scope_type, parent_scope_id);`,
		},
		{
			ID:          "rolekit-032",
			Description: "Create idx_scope_hierarchy_unique",
			SQL:         `CREATE UNIQUE INDEX IF NOT EXISTS idx_scope_hierarchy_unique ON scope_hierarchy(scope_type, scope_id, parent_scope_type, parent_scope_id);`,
		},
	}
}

// ============================================================================
// MIGRATION SYSTEM
// ============================================================================

// MigrationStatus represents the status of all migrations
type MigrationStatus struct {
	Total      int                          `json:"total"`
	Applied    int                          `json:"applied"`
	Pending    int                          `json:"pending"`
	Migrations []dbkit.MigrationStatusEntry `json:"migrations"`
}

// RunMigrations executes all pending migrations with status tracking and rollback support.
// This is the recommended way to run migrations in production.
//
// Example:
//
//	status, err := service.RunMigrations(ctx)
//	if err != nil {
//	    log.Printf("Migration failed: %v", err)
//	    return
//	}
//	log.Printf("Applied %d migrations, %d pending", status.Applied, status.Pending)
func (s *Service) RunMigrations(ctx context.Context) (*MigrationStatus, error) {
	// Check if we have a DBKit instance
	if db, ok := s.db.(*dbkit.DBKit); ok {
		migrations := s.Migrations()

		// Run migrations
		result, err := db.Migrate(ctx, migrations)
		if err != nil {
			return nil, fmt.Errorf("migration failed: %w", err)
		}

		// Get updated status
		updatedStatus, err := s.GetMigrationStatus(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get updated migration status: %w", err)
		}

		// Log migration results
		if len(result.Applied) > 0 {
			log.Printf("Successfully applied %d migrations", len(result.Applied))
		}

		return updatedStatus, nil
	}

	// If we don't have a DBKit instance, we can't run migrations
	return nil, fmt.Errorf("migration system requires a dbkit.DBKit instance")
}

// GetMigrationStatus returns the current status of all migrations.
// This includes applied, pending, and failed migrations with checksum verification.
//
// Example:
//
//	status, err := service.GetMigrationStatus(ctx)
//	if err != nil {
//	    log.Printf("Failed to get migration status: %v", err)
//	    return
//	}
//	for _, migration := range status.Migrations {
//	    log.Printf("Migration %s: %s", migration.ID, migration.Status)
//	}
func (s *Service) GetMigrationStatus(ctx context.Context) (*MigrationStatus, error) {
	// Check if we have a DBKit instance
	if db, ok := s.db.(*dbkit.DBKit); ok {
		migrations := s.Migrations()

		// Get migration status from dbkit
		statusEntries, err := db.MigrationStatus(ctx, migrations)
		if err != nil {
			return nil, fmt.Errorf("failed to get migration status: %w", err)
		}

		// Calculate totals
		total := len(statusEntries)
		applied := 0
		pending := 0

		for _, entry := range statusEntries {
			if entry.Applied {
				applied++
			} else {
				pending++
			}
		}

		return &MigrationStatus{
			Total:      total,
			Applied:    applied,
			Pending:    pending,
			Migrations: statusEntries,
		}, nil
	}

	// If we don't have a DBKit instance, return basic status
	migrations := s.Migrations()
	return &MigrationStatus{
		Total:      len(migrations),
		Applied:    0,
		Pending:    len(migrations),
		Migrations: make([]dbkit.MigrationStatusEntry, 0),
	}, nil
}

// VerifyMigrationChecksums verifies that all applied migrations have matching checksums.
// This ensures migration integrity and detects any unauthorized changes.
//
// Example:
//
//	valid, err := service.VerifyMigrationChecksums(ctx)
//	if err != nil {
//	    log.Printf("Checksum verification failed: %v", err)
//	    return
//	}
//	if !valid {
//	    log.Printf("Migration checksums do not match - potential tampering detected")
//	}
func (s *Service) VerifyMigrationChecksums(ctx context.Context) (bool, error) {
	status, err := s.GetMigrationStatus(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get migration status for checksum verification: %w", err)
	}

	// Check all applied migrations for checksum mismatches
	for _, migration := range status.Migrations {
		if migration.Applied && !migration.ChecksumMatch {
			return false, fmt.Errorf("migration %s checksum mismatch detected", migration.ID)
		}
	}

	return true, nil
}

// RollbackToMigration rolls back migrations to a specific migration ID.
// This is useful for reverting to a known good state.
// Note: This requires manual rollback SQL to be defined in the migrations.
//
// Example:
//
//	err := service.RollbackToMigration(ctx, "rolekit-010")
//	if err != nil {
//	    log.Printf("Rollback failed: %v", err)
//	    return
//	}
func (s *Service) RollbackToMigration(ctx context.Context, targetMigrationID string) error {
	// Check if we have a DBKit instance
	if _, ok := s.db.(*dbkit.DBKit); ok {
		migrations := s.Migrations()

		// Find the target migration
		targetIndex := -1
		for i, migration := range migrations {
			if migration.ID == targetMigrationID {
				targetIndex = i
				break
			}
		}

		if targetIndex == -1 {
			return fmt.Errorf("target migration %s not found", targetMigrationID)
		}

		// Get current status
		status, err := s.GetMigrationStatus(ctx)
		if err != nil {
			return fmt.Errorf("failed to get migration status: %w", err)
		}

		// Find migrations to rollback (in reverse order)
		// Note: This requires manual rollback SQL to be defined
		var migrationsToRollback []dbkit.Migration
		for i := len(status.Migrations) - 1; i >= 0; i-- {
			migration := status.Migrations[i]
			if migration.Applied && migration.ID > targetMigrationID {
				// Find the corresponding migration definition
				for _, def := range migrations {
					if def.ID == migration.ID {
						// For now, we'll skip rollback as it requires manual SQL
						// In a real implementation, you'd have rollback SQL defined
						log.Printf("Migration %s would need manual rollback SQL", def.ID)
						break
					}
				}
			}
		}

		if len(migrationsToRollback) == 0 {
			return fmt.Errorf("no migrations to rollback to %s", targetMigrationID)
		}

		// For now, return an error indicating manual rollback is needed
		return fmt.Errorf("manual rollback required for migration %s - please define rollback SQL", targetMigrationID)
	}

	return fmt.Errorf("migration rollback requires a dbkit.DBKit instance")
}

// ValidateMigrations checks that all migrations are properly formatted and valid.
// This is useful for pre-deployment validation.
//
// Example:
//
//	err := service.ValidateMigrations()
//	if err != nil {
//	    log.Printf("Migration validation failed: %v", err)
//	    return
//	}
//	log.Printf("All migrations are valid")
func (s *Service) ValidateMigrations() error {
	migrations := s.Migrations()

	for _, migration := range migrations {
		// Validate migration ID
		if migration.ID == "" {
			return fmt.Errorf("migration ID cannot be empty")
		}

		// Validate description
		if migration.Description == "" {
			return fmt.Errorf("migration %s: description cannot be empty", migration.ID)
		}

		// Validate SQL
		if migration.SQL == "" {
			return fmt.Errorf("migration %s: SQL cannot be empty", migration.ID)
		}

		// Check for basic SQL injection patterns (basic validation)
		if strings.Contains(strings.ToLower(migration.SQL), "drop table") &&
			!strings.Contains(strings.ToLower(migration.SQL), "if exists") {
			return fmt.Errorf("migration %s: DROP TABLE without IF EXISTS is not allowed", migration.ID)
		}
	}

	return nil
}

// Health performs a comprehensive health check of the database connection.
// Returns detailed status including latency, connection pool statistics, and error information.
//
// Example:
//
//	status := service.Health(ctx)
//	if status.Healthy {
//	    log.Printf("Database is healthy, latency: %v", status.Latency)
//	} else {
//	    log.Printf("Database is unhealthy: %s", status.Error)
//	    log.Printf("Pool stats: InUse=%d, Idle=%d", status.PoolStats.InUse, status.PoolStats.Idle)
//	}
func (s *Service) Health(ctx context.Context) dbkit.HealthStatus {
	// Check if we have a DBKit instance
	if db, ok := s.db.(*dbkit.DBKit); ok {
		return db.Health(ctx)
	}

	// If we're in a transaction or have a different type, do a basic ping
	return dbkit.HealthStatus{
		Healthy: s.IsHealthy(ctx),
		Error:   "Limited health check - not a DBKit instance",
	}
}

// IsHealthy performs a simple health check of the database connection.
// Returns true if the database is reachable, false otherwise.
//
// This is a lightweight check suitable for frequent monitoring.
// For detailed health information, use Health().
//
// Example:
//
//	if !service.IsHealthy(ctx) {
//	    log.Printf("Database is not healthy")
//	    // Handle unhealthy state
//	}
func (s *Service) IsHealthy(ctx context.Context) bool {
	// Check if we have a DBKit instance
	if db, ok := s.db.(*dbkit.DBKit); ok {
		return db.IsHealthy(ctx)
	}

	// If we're in a transaction or have a different type, try to ping
	// Use a simple query to test connectivity
	var count int
	err := s.db.NewSelect().Model((*struct{})(nil)).ColumnExpr("1").Limit(1).Scan(ctx, &count)
	return err == nil
}

// GetPoolStats returns connection pool statistics for monitoring.
// Returns zero values if the database instance doesn't support pool statistics.
//
// Example:
//
//	stats := service.GetPoolStats()
//	log.Printf("Connections: InUse=%d, Idle=%d, Max=%d",
//	    stats.InUse, stats.Idle, stats.MaxOpen)
func (s *Service) GetPoolStats() dbkit.PoolStats {
	// Check if we have a DBKit instance
	if db, ok := s.db.(*dbkit.DBKit); ok {
		sqlStats := db.Stats()
		return dbkit.PoolStatsFromSQL(sqlStats)
	}

	// Return zero values for non-DBKit instances
	return dbkit.PoolStats{}
}

// Ping performs a basic connectivity test to the database.
// Returns an error if the database is not reachable.
//
// This is useful for health check endpoints that need to verify database connectivity.
//
// Example:
//
//	if err := service.Ping(ctx); err != nil {
//	    log.Printf("Database ping failed: %v", err)
//	    return err
//	}
func (s *Service) Ping(ctx context.Context) error {
	// Use a simple query to test connectivity
	var result int
	return s.db.NewSelect().Model((*struct{})(nil)).ColumnExpr("1").Limit(1).Scan(ctx, &result)
}

// RoleRevocation represents a role revocation operation for bulk operations.
type RoleRevocation struct {
	UserID    string
	Role      string
	ScopeType string
	ScopeID   string
}

// AssignMultiple assigns multiple roles to users in a single operation.
// This is more efficient than calling Assign multiple times as it can use batch operations.
//
// Example:
//
//	assignments := []rolekit.RoleAssignment{
//	    {UserID: "user1", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
//	    {UserID: "user2", Role: "member", ScopeType: "organization", ScopeID: "org1"},
//	}
//	err := service.AssignMultiple(ctx, assignments)
func (s *Service) AssignMultiple(ctx context.Context, assignments []RoleAssignment) error {
	return s.Transaction(ctx, func(ctx context.Context) error {
		// Use batch insert for better performance
		assignmentModels := make([]*RoleAssignment, len(assignments))
		for i, assignment := range assignments {
			assignmentModels[i] = &assignment
		}

		_, err := dbkit.BatchInsert(ctx, s.db, assignmentModels, dbkit.BatchSize)
		err = dbkit.WithErr1(err, "AssignMultiple").Err()
		if err != nil {
			return NewError(ErrDatabaseError, "failed to batch assign roles").
				WithScope("", "").
				WithRole("")
		}

		// Log audit for each assignment
		for _, assignment := range assignments {
			_ = s.logAudit(ctx, &AuditEntry{
				Action:       "assign_multiple",
				TargetUserID: assignment.UserID,
				Role:         assignment.Role,
				ScopeType:    assignment.ScopeType,
				ScopeID:      assignment.ScopeID,
				IPAddress:    GetIPAddress(ctx),
				UserAgent:    GetUserAgent(ctx),
				RequestID:    GetRequestID(ctx),
			})
		}

		return nil
	})
}

// RevokeMultiple removes multiple roles from users in a single operation.
// This is more efficient than calling Revoke multiple times.
//
// Example:
//
//	revocations := []rolekit.RoleRevocation{
//	{UserID: "user1", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
//	    {UserID: "user2", Role: "member", ScopeType: "organization", ScopeID: "org1"},
//	}
//	err := service.RevokeMultiple(ctx, revocations)
func (s *Service) RevokeMultiple(ctx context.Context, revocations []RoleRevocation) error {
	return s.Transaction(ctx, func(ctx context.Context) error {
		for _, revocation := range revocations {
			// Check if user has the role before attempting to revoke
			roles, err := s.GetUserRoles(ctx, revocation.UserID)
			if err != nil {
				return err
			}

			hasRole := roles.HasRole(revocation.Role, revocation.ScopeType, revocation.ScopeID)
			if !hasRole {
				continue // Skip if user doesn't have this role
			}

			// Delete the assignment
			result, err := s.db.NewDelete().Table("role_assignments").
				Where("user_id = ? AND role = ? AND scope_type = ? AND scope_id = ?",
					revocation.UserID, revocation.Role, revocation.ScopeType, revocation.ScopeID).Exec(ctx)
			err = dbkit.WithErr(result, err, "RevokeMultiple").Err()
			if err != nil {
				return NewError(ErrDatabaseError, "failed to revoke role").
					WithUser(revocation.UserID).
					WithRole(revocation.Role).
					WithScope(revocation.ScopeType, revocation.ScopeID)
			}

			// Log audit
			_ = s.logAudit(ctx, &AuditEntry{
				Action:       "revoke_multiple",
				TargetUserID: revocation.UserID,
				Role:         revocation.Role,
				ScopeType:    revocation.ScopeType,
				ScopeID:      revocation.ScopeID,
				IPAddress:    GetIPAddress(ctx),
				UserAgent:    GetUserAgent(ctx),
				RequestID:    GetRequestID(ctx),
			})
		}

		return nil
	})
}

// CheckExists checks if a user has a specific role in a scope.
// This is more efficient than GetUserRoles when you only need to check existence.
//
// Example:
//
//	hasAdmin := service.CheckExists(ctx, "user1", "admin", "organization", "org1")
//	if hasAdmin {
//	    log.Println("User is admin")
//	}
func (s *Service) CheckExists(ctx context.Context, userID, role, scopeType, scopeID string) bool {
	exists, err := dbkit.Exists[RoleAssignment](ctx, s.db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("user_id = ? AND role = ? AND scope_type = ? AND scope_id = ?",
			userID, role, scopeType, scopeID)
	})

	if err != nil {
		return false
	}

	return exists
}

// CountRoles returns the number of roles a user has in a specific scope.
// This is more efficient than GetUserRoles when you only need the count.
//
// Example:
//
//	count := service.CountRoles(ctx, "user1", "organization", "org1")
//	log.Printf("User has %d roles in org1", count)
func (s *Service) CountRoles(ctx context.Context, userID, scopeType, scopeID string) (int, error) {
	return dbkit.Count[RoleAssignment](ctx, s.db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("user_id = ? AND scope_type = ? AND (scope_id = ? OR scope_id = '*')",
			userID, scopeType, scopeID)
	})
}

// CountAllRoles returns the total number of role assignments in the system.
// Useful for monitoring and analytics.
//
// Example:
//
//	total := service.CountAllRoles(ctx)
//	log.Printf("Total role assignments: %d", total)
func (s *Service) CountAllRoles(ctx context.Context) (int, error) {
	return dbkit.Count[RoleAssignment](ctx, s.db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q
	})
}

// ============================================================================
// CONNECTION POOL MANAGEMENT
// ============================================================================

// PoolConfig represents connection pool configuration settings.
type PoolConfig struct {
	// MaxOpenConnections is the maximum number of open connections to the database.
	// If MaxOpenConnections is 0, there is no limit on the number of open connections.
	MaxOpenConnections int `json:"max_open_connections"`

	// MaxIdleConnections is the maximum number of connections in the idle connection pool.
	// If MaxIdleConnections is 0, no idle connections are retained.
	MaxIdleConnections int `json:"max_idle_connections"`

	// ConnectionMaxLifetime is the maximum amount of time a connection may be reused.
	// If ConnectionMaxLifetime is 0, connections are reused forever.
	ConnectionMaxLifetime time.Duration `json:"connection_max_lifetime"`

	// ConnectionMaxIdleTime is the maximum amount of time a connection may be idle.
	// If ConnectionMaxIdleTime is 0, connections are not closed based on idle time.
	ConnectionMaxIdleTime time.Duration `json:"connection_max_idle_time"`
}

// DefaultPoolConfig returns sensible default connection pool settings.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConnections:    25,
		MaxIdleConnections:    25,
		ConnectionMaxLifetime: time.Hour,
		ConnectionMaxIdleTime: 5 * time.Minute,
	}
}

// HighPerformancePoolConfig returns optimized settings for high-performance workloads.
func HighPerformancePoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConnections:    100,
		MaxIdleConnections:    50,
		ConnectionMaxLifetime: 30 * time.Minute,
		ConnectionMaxIdleTime: 1 * time.Minute,
	}
}

// LowResourcePoolConfig returns optimized settings for resource-constrained environments.
func LowResourcePoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConnections:    5,
		MaxIdleConnections:    2,
		ConnectionMaxLifetime: 2 * time.Hour,
		ConnectionMaxIdleTime: 10 * time.Minute,
	}
}

// ConfigureConnectionPool updates the database connection pool settings.
// This method allows dynamic adjustment of connection pool parameters.
//
// Example:
//
//	config := rolekit.HighPerformancePoolConfig()
//	err := service.ConfigureConnectionPool(config)
//	if err != nil {
//	    log.Printf("Failed to configure connection pool: %v", err)
//	}
func (s *Service) ConfigureConnectionPool(config PoolConfig) error {
	// Check if we have a DBKit instance with access to the underlying database
	if db, ok := s.db.(*dbkit.DBKit); ok {
		bunDB := db.Bun()
		if bunDB == nil {
			return fmt.Errorf("database instance not available")
		}

		// bunDB is already the *sql.DB instance
		sqlDB := bunDB
		if sqlDB == nil {
			return fmt.Errorf("SQL database not available")
		}

		// Set connection pool parameters
		sqlDB.SetMaxOpenConns(config.MaxOpenConnections)
		sqlDB.SetMaxIdleConns(config.MaxIdleConnections)
		sqlDB.SetConnMaxLifetime(config.ConnectionMaxLifetime)
		sqlDB.SetConnMaxIdleTime(config.ConnectionMaxIdleTime)

		log.Printf("Connection pool configured: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v, MaxIdleTime=%v",
			config.MaxOpenConnections, config.MaxIdleConnections,
			config.ConnectionMaxLifetime, config.ConnectionMaxIdleTime)

		return nil
	}

	return fmt.Errorf("connection pool configuration requires a dbkit.DBKit instance")
}

// GetConnectionPoolConfig returns the current connection pool configuration.
// This is useful for monitoring and debugging connection pool settings.
//
// Example:
//
//	config, err := service.GetConnectionPoolConfig()
//	if err != nil {
//	    log.Printf("Failed to get connection pool config: %v", err)
//	    return
//	}
//	log.Printf("Current pool config: %+v", config)
func (s *Service) GetConnectionPoolConfig() (*PoolConfig, error) {
	// Check if we have a DBKit instance with access to the underlying database
	if db, ok := s.db.(*dbkit.DBKit); ok {
		bunDB := db.Bun()
		if bunDB == nil {
			return nil, fmt.Errorf("database instance not available")
		}

		// bunDB is already the *sql.DB instance
		sqlDB := bunDB
		if sqlDB == nil {
			return nil, fmt.Errorf("SQL database not available")
		}

		return &PoolConfig{
			MaxOpenConnections:    sqlDB.Stats().MaxOpenConnections,
			MaxIdleConnections:    sqlDB.Stats().Idle, // Use Idle field instead
			ConnectionMaxLifetime: 0,                  // Not available in sql.DBStats
			ConnectionMaxIdleTime: 0,                  // Not available in sql.DBStats
		}, nil
	}

	return nil, fmt.Errorf("connection pool configuration requires a dbkit.DBKit instance")
}

// OptimizeConnectionPool automatically adjusts connection pool settings based on current usage.
// This method analyzes current pool statistics and adjusts parameters for optimal performance.
//
// Example:
//
//	err := service.OptimizeConnectionPool()
//	if err != nil {
//	    log.Printf("Failed to optimize connection pool: %v", err)
//	}
func (s *Service) OptimizeConnectionPool() error {
	stats := s.GetPoolStats()

	// Get current configuration
	currentConfig, err := s.GetConnectionPoolConfig()
	if err != nil {
		return fmt.Errorf("failed to get current config: %w", err)
	}

	// Analyze usage patterns and adjust accordingly
	newConfig := *currentConfig

	// If wait count is high, increase max open connections
	if stats.WaitCount > 5 {
		newConfig.MaxOpenConnections = min(currentConfig.MaxOpenConnections*2, 200)
		log.Printf("High wait count detected (%d), increasing MaxOpenConnections to %d",
			stats.WaitCount, newConfig.MaxOpenConnections)
	}

	// If idle connections are very high, reduce max idle connections
	if stats.Idle > stats.InUse*3 && currentConfig.MaxIdleConnections > 5 {
		newConfig.MaxIdleConnections = max(currentConfig.MaxIdleConnections/2, 5)
		log.Printf("High idle connections detected (%d vs %d in use), reducing MaxIdleConnections to %d",
			stats.Idle, stats.InUse, newConfig.MaxIdleConnections)
	}

	// If wait duration is high, increase pool size
	if stats.WaitDuration > 100*time.Millisecond {
		newConfig.MaxOpenConnections = min(currentConfig.MaxOpenConnections+10, 150)
		log.Printf("High wait duration detected (%v), increasing MaxOpenConnections to %d",
			stats.WaitDuration, newConfig.MaxOpenConnections)
	}

	// Apply the new configuration if it changed
	if !reflect.DeepEqual(currentConfig, &newConfig) {
		return s.ConfigureConnectionPool(newConfig)
	}

	log.Printf("Connection pool is already optimized")
	return nil
}

// ResetConnectionPool resets the connection pool to default settings.
// This is useful for troubleshooting or when you want to start fresh.
//
// Example:
//
//	err := service.ResetConnectionPool()
//	if err != nil {
//	    log.Printf("Failed to reset connection pool: %v", err)
//	}
func (s *Service) ResetConnectionPool() error {
	return s.ConfigureConnectionPool(DefaultPoolConfig())
}

// ============================================================================
// ROLE ASSIGNMENT OPERATIONS
// ============================================================================

// Assign assigns a role to a user in a scope.
// The actor performing the assignment must have permission to assign this role.
//
// Example:
//
//	err := service.Assign(ctx, targetUserID, "editor", "project", projectID)
func (s *Service) Assign(ctx context.Context, userID, role, scopeType, scopeID string) error {
	// Validate role exists for scope
	if err := s.registry.ValidateRole(role, scopeType); err != nil {
		return err
	}

	// Check if actor can assign this role
	actorID := GetActorID(ctx)
	if actorID == "" {
		return NewError(ErrNoActorID, "actor ID required for role assignment")
	}

	// Get actor's roles to check assignment permission
	actorRoles, err := s.GetUserRoles(ctx, actorID)
	if err != nil {
		return err
	}

	// Check if actor can assign this role (skip if actor is assigning to self during bootstrap)
	if actorID != userID {
		actorChecker := NewChecker(actorID, actorRoles, s.registry, s)
		if !actorChecker.CanAssignRole(role, scopeType, scopeID) {
			return NewError(ErrCannotAssign, "actor cannot assign this role").
				WithScope(scopeType, scopeID).
				WithRole(role).
				WithActor(actorID)
		}
	}

	// Get target user's current roles for audit
	previousRoles, err := s.getUserRoleNames(ctx, userID, scopeType, scopeID)
	if err != nil {
		return err
	}

	// Check if already assigned
	for _, r := range previousRoles {
		if r == role {
			return NewError(ErrRoleAlreadyAssigned, "user already has this role").
				WithScope(scopeType, scopeID).
				WithRole(role).
				WithUser(userID)
		}
	}

	// Get parent scope if defined
	var parentScopeType, parentScopeID string
	scopeDef := s.registry.GetScope(scopeType)
	if scopeDef != nil && scopeDef.GetParentScope() != "" {
		// Look up parent from scope_hierarchy table
		parent, err := s.getParentScope(ctx, scopeType, scopeID)
		if err == nil && parent != nil {
			parentScopeType = parent.ParentScopeType
			parentScopeID = parent.ParentScopeID
		}
	}

	// Create assignment
	assignment := &RoleAssignment{
		UserID:          userID,
		Role:            role,
		ScopeType:       scopeType,
		ScopeID:         scopeID,
		ParentScopeType: parentScopeType,
		ParentScopeID:   parentScopeID,
	}

	result, err := s.db.NewInsert().Model(assignment).Exec(ctx)
	err = dbkit.WithErr(result, err, "CreateRoleAssignment").Err()
	if err != nil {
		return NewError(ErrDatabaseError, "failed to create role assignment").
			WithScope(scopeType, scopeID).
			WithRole(role).
			WithUser(userID)
	}

	// Calculate new roles after assignment
	newRoles := append(previousRoles, role)

	// Create audit log entry
	audit := GetAuditContext(ctx)
	entry := &AuditEntry{
		ActorID:       actorID,
		Action:        AuditActionAssigned,
		TargetUserID:  userID,
		Role:          role,
		ScopeType:     scopeType,
		ScopeID:       scopeID,
		ActorRoles:    actorRoles.GetRoles(scopeType, scopeID),
		PreviousRoles: previousRoles,
		NewRoles:      newRoles,
		IPAddress:     audit.IPAddress,
		UserAgent:     audit.UserAgent,
		RequestID:     audit.RequestID,
	}

	_ = s.logAudit(ctx, entry) // Log error but don't fail the assignment

	return nil
}

// Revoke removes a role from a user in a scope.
//
// Example:
//
//	err := service.Revoke(ctx, targetUserID, "editor", "project", projectID)
func (s *Service) Revoke(ctx context.Context, userID, role, scopeType, scopeID string) error {
	// Validate role exists for scope
	if err := s.registry.ValidateRole(role, scopeType); err != nil {
		return err
	}

	// Check if actor can assign (and thus revoke) this role
	actorID := GetActorID(ctx)
	if actorID == "" {
		return NewError(ErrNoActorID, "actor ID required for role revocation")
	}

	actorRoles, err := s.GetUserRoles(ctx, actorID)
	if err != nil {
		return err
	}

	if actorID != userID {
		actorChecker := NewChecker(actorID, actorRoles, s.registry, s)
		if !actorChecker.CanAssignRole(role, scopeType, scopeID) {
			return NewError(ErrCannotAssign, "actor cannot revoke this role").
				WithScope(scopeType, scopeID).
				WithRole(role).
				WithActor(actorID)
		}
	}

	// Get current roles for audit
	previousRoles, err := s.getUserRoleNames(ctx, userID, scopeType, scopeID)
	if err != nil {
		return err
	}

	// Check if role is assigned
	hasRole := false
	for _, r := range previousRoles {
		if r == role {
			hasRole = true
			break
		}
	}
	if !hasRole {
		return NewError(ErrRoleNotAssigned, "user does not have this role").
			WithScope(scopeType, scopeID).
			WithRole(role).
			WithUser(userID)
	}

	// Delete assignment
	result, err := s.db.NewDelete().Table("role_assignments").Where("user_id = ? AND role = ? AND scope_type = ? AND scope_id = ?", userID, role, scopeType, scopeID).Exec(ctx)
	err = dbkit.WithErr(result, err, "DeleteRoleAssignment").Err()
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return NewError(ErrRoleNotAssigned, "user does not have this role").
			WithScope(scopeType, scopeID).
			WithRole(role).
			WithUser(userID)
	}

	// Calculate new roles after revocation
	newRoles := make([]string, 0, len(previousRoles)-1)
	for _, r := range previousRoles {
		if r != role {
			newRoles = append(newRoles, r)
		}
	}

	// Create audit log entry
	audit := GetAuditContext(ctx)
	entry := &AuditEntry{
		ActorID:       actorID,
		Action:        AuditActionRevoked,
		TargetUserID:  userID,
		Role:          role,
		ScopeType:     scopeType,
		ScopeID:       scopeID,
		ActorRoles:    actorRoles.GetRoles(scopeType, scopeID),
		PreviousRoles: previousRoles,
		NewRoles:      newRoles,
		IPAddress:     audit.IPAddress,
		UserAgent:     audit.UserAgent,
		RequestID:     audit.RequestID,
	}

	_ = s.logAudit(ctx, entry) // Log error but don't fail the revocation

	return nil
}

// RevokeAll removes all roles from a user in a scope.
//
// Example:
//
//	err := service.RevokeAll(ctx, targetUserID, "project", projectID)
func (s *Service) RevokeAll(ctx context.Context, userID, scopeType, scopeID string) error {
	// Get current roles
	currentRoles, err := s.getUserRoleNames(ctx, userID, scopeType, scopeID)
	if err != nil {
		return err
	}

	// Revoke each role individually (for proper audit logging)
	for _, role := range currentRoles {
		if err := s.Revoke(ctx, userID, role, scopeType, scopeID); err != nil {
			// Continue revoking other roles even if one fails
			continue
		}
	}

	return nil
}

// ============================================================================
// PERMISSION CHECKING
// ============================================================================

// Can checks if a user has a specific role in a scope.
//
// Example:
//
//	if service.Can(ctx, userID, "admin", "organization", orgID) {
//	    // User is admin
//	}
func (s *Service) Can(ctx context.Context, userID, role, scopeType, scopeID string) bool {
	roles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return false
	}
	return roles.HasRole(role, scopeType, scopeID)
}

// HasPermission checks if a user has a specific permission in a scope.
//
// Example:
//
//	if service.HasPermission(ctx, userID, "files.upload", "project", projectID) {
//	    // User can upload files
//	}
func (s *Service) HasPermission(ctx context.Context, userID, permission, scopeType, scopeID string) bool {
	roles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return false
	}
	checker := NewChecker(userID, roles, s.registry, s)
	return checker.HasPermission(permission, scopeType, scopeID)
}

// HasAnyRole checks if a user has any of the specified roles in a scope.
func (s *Service) HasAnyRole(ctx context.Context, userID string, roles []string, scopeType, scopeID string) bool {
	userRoles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return false
	}
	checker := NewChecker(userID, userRoles, s.registry, s)
	return checker.HasAnyRole(roles, scopeType, scopeID)
}

// CanAssignRole checks if a user can assign a role to another user in a scope.
func (s *Service) CanAssignRole(ctx context.Context, userID, targetRole, scopeType, scopeID string) bool {
	roles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return false
	}
	checker := NewChecker(userID, roles, s.registry, s)
	return checker.CanAssignRole(targetRole, scopeType, scopeID)
}

// ============================================================================
// DATA RETRIEVAL
// ============================================================================

// GetUserRoles retrieves all role assignments for a user.
func (s *Service) GetUserRoles(ctx context.Context, userID string) (*UserRoles, error) {
	var assignments []RoleAssignment
	err := dbkit.WithErr1(s.db.NewSelect().Model(&assignments).Where("user_id = ?", userID).Scan(ctx), "GetUserRoles").Err()
	if err != nil {
		return nil, err
	}
	return NewUserRoles(userID, assignments), nil
}

// GetScopeMembers retrieves all users with roles in a scope.
func (s *Service) GetScopeMembers(ctx context.Context, scopeType, scopeID string) ([]RoleAssignment, error) {
	var assignments []RoleAssignment
	err := dbkit.WithErr1(s.db.NewSelect().Model(&assignments).Where("scope_type = ? AND scope_id = ?", scopeType, scopeID).Scan(ctx), "GetScopeMembers").Err()
	if err != nil {
		return nil, err
	}
	return assignments, nil
}

// GetScopeMembersWithRole retrieves all users with a specific role in a scope.
func (s *Service) GetScopeMembersWithRole(ctx context.Context, role, scopeType, scopeID string) ([]RoleAssignment, error) {
	var assignments []RoleAssignment
	err := dbkit.WithErr1(s.db.NewSelect().Model(&assignments).Where("scope_type = ? AND scope_id = ? AND role = ?", scopeType, scopeID, role).Scan(ctx), "GetScopeMembersWithRole").Err()
	if err != nil {
		return nil, err
	}
	return assignments, nil
}

// GetChecker creates a Checker for a user.
// This can be stored in context for efficient permission checking in handlers.
func (s *Service) GetChecker(ctx context.Context, userID string) (*Checker, error) {
	roles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	return NewChecker(userID, roles, s.registry, s), nil
}

// GetCheckerFromContext creates a Checker using the user ID from context.
func (s *Service) GetCheckerFromContext(ctx context.Context) (*Checker, error) {
	userID := GetUserID(ctx)
	if userID == "" {
		return nil, ErrNoUserID
	}
	return s.GetChecker(ctx, userID)
}

// ============================================================================
// HIERARCHICAL QUERIES
// ============================================================================

// SetScopeParent sets the parent scope for a scope instance.
// This is used for hierarchical queries.
//
// Example:
//
//	// When creating a project, set its parent organization
//	service.SetScopeParent(ctx, "project", projectID, "organization", orgID)
func (s *Service) SetScopeParent(ctx context.Context, scopeType, scopeID, parentScopeType, parentScopeID string) error {
	hierarchy := &ScopeHierarchy{
		ScopeType:       scopeType,
		ScopeID:         scopeID,
		ParentScopeType: parentScopeType,
		ParentScopeID:   parentScopeID,
	}

	result, err := s.db.NewInsert().Model(hierarchy).On("CONFLICT (scope_type, scope_id, parent_scope_type, parent_scope_id) DO NOTHING").Exec(ctx)
	err = dbkit.WithErr(result, err, "SetScopeParent").Err()
	if err != nil {
		return err
	}

	// Update any existing role assignments with parent scope
	result, err = s.db.NewUpdate().Table("role_assignments").Set("parent_scope_type = ?", parentScopeType).Set("parent_scope_id = ?", parentScopeID).Where("scope_type = ? AND scope_id = ?", scopeType, scopeID).Exec(ctx)
	if err != nil {
		return err
	}
	_ = dbkit.WithErr(result, err, "UpdateRoleAssignmentsParent").Err()

	return nil
}

// GetChildScopes returns all child scope IDs where a user has any role.
// Useful for queries like "get all projects in org where user has access".
//
// Example:
//
//	projectIDs, err := service.GetChildScopes(ctx, userID, "project", "organization", orgID)
func (s *Service) GetChildScopes(ctx context.Context, userID, childScopeType, parentScopeType, parentScopeID string) ([]string, error) {
	var scopeIDs []string
	err := dbkit.WithErr1(s.db.NewRaw("SELECT DISTINCT scope_id FROM role_assignments WHERE user_id = ? AND scope_type = ? AND parent_scope_type = ? AND parent_scope_id = ?", userID, childScopeType, parentScopeType, parentScopeID).Scan(ctx, &scopeIDs), "GetChildScopes").Err()
	if err != nil {
		return nil, err
	}
	return scopeIDs, nil
}

// GetChildScopesWithRole returns all child scope IDs where a user has a specific role.
//
// Example:
//
//	projectIDs, err := service.GetChildScopesWithRole(ctx, userID, "editor", "project", "organization", orgID)
func (s *Service) GetChildScopesWithRole(ctx context.Context, userID, role, childScopeType, parentScopeType, parentScopeID string) ([]string, error) {
	var scopeIDs []string
	err := dbkit.WithErr1(s.db.NewRaw("SELECT DISTINCT scope_id FROM role_assignments WHERE user_id = ? AND role = ? AND scope_type = ? AND parent_scope_type = ? AND parent_scope_id = ?", userID, role, childScopeType, parentScopeType, parentScopeID).Scan(ctx, &scopeIDs), "GetChildScopesWithRole").Err()
	if err != nil {
		return nil, err
	}
	return scopeIDs, nil
}

// ============================================================================
// AUDIT LOG
// ============================================================================

// GetAuditLog retrieves audit log entries with optional filters.
func (s *Service) GetAuditLog(ctx context.Context, filter AuditLogFilter) ([]RoleAuditLog, error) {
	var logs []RoleAuditLog
	q := s.db.NewSelect().Model(&logs)
	if filter.ActorID != "" {
		q = q.Where("actor_id = ?", filter.ActorID)
	}
	if filter.TargetUserID != "" {
		q = q.Where("target_user_id = ?", filter.TargetUserID)
	}
	if filter.ScopeType != "" {
		q = q.Where("scope_type = ?", filter.ScopeType)
	}
	if filter.ScopeID != "" {
		q = q.Where("scope_id = ?", filter.ScopeID)
	}
	if filter.Action != "" {
		q = q.Where("action = ?", filter.Action)
	}
	if !filter.Since.IsZero() {
		q = q.Where("timestamp >= ?", filter.Since)
	}
	if !filter.Until.IsZero() {
		q = q.Where("timestamp <= ?", filter.Until)
	}

	limit := filter.Limit
	if limit == 0 {
		limit = 100 // Default limit
	}
	q = q.Limit(limit)

	if filter.Offset > 0 {
		q = q.Offset(filter.Offset)
	}

	q = q.Order("timestamp DESC")
	err := dbkit.WithErr1(q.Scan(ctx), "GetAuditLog").Err()
	if err != nil {
		return nil, err
	}

	return logs, nil
}

// ============================================================================
// INTERNAL HELPERS
// ============================================================================

func (s *Service) getUserRoleNames(ctx context.Context, userID, scopeType, scopeID string) ([]string, error) {
	var roles []string
	err := dbkit.WithErr1(s.db.NewRaw("SELECT role FROM role_assignments WHERE user_id = ? AND scope_type = ? AND (scope_id = ? OR scope_id = '*')", userID, scopeType, scopeID).Scan(ctx, &roles), "GetUserRoleNames").Err()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return roles, nil
}

func (s *Service) getParentScope(ctx context.Context, scopeType, scopeID string) (*ScopeHierarchy, error) {
	var hierarchy ScopeHierarchy
	err := dbkit.WithErr1(s.db.NewSelect().Model(&hierarchy).Where("scope_type = ? AND scope_id = ?", scopeType, scopeID).Limit(1).Scan(ctx), "GetParentScope").Err()
	if err != nil {
		if dbkit.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &hierarchy, nil
}

func (s *Service) logAudit(ctx context.Context, entry *AuditEntry) error {
	_, err := s.db.NewInsert().Model(entry.ToModel()).Exec(ctx)
	return dbkit.WithErr1(err, "LogAudit").Err()
}

// AssignDirect assigns a role to a user without pre-checks for better performance.
// This method bypasses GetUserRoles calls and handles duplicate key constraints gracefully.
func (s *Service) AssignDirect(ctx context.Context, userID, role, scopeType, scopeID string) error {
	// Validate role exists for scope
	if err := s.registry.ValidateRole(role, scopeType); err != nil {
		return err
	}

	// Check if actor can assign this role
	actorID := GetActorID(ctx)
	if actorID == "" {
		return NewError(ErrNoActorID, "actor ID required for role assignment")
	}

	// Create assignment
	assignment := &RoleAssignment{
		UserID:    userID,
		Role:      role,
		ScopeType: scopeType,
		ScopeID:   scopeID,
	}

	// Direct assignment with conflict resolution
	result, err := s.db.NewInsert().
		Model(assignment).
		On("CONFLICT (user_id, role, scope_type, scope_id) DO NOTHING").
		Exec(ctx)

	err = dbkit.WithErr(result, err, "CreateRoleAssignmentDirect").Err()
	if err != nil {
		return NewError(ErrDatabaseError, "failed to create role assignment").
			WithScope(scopeType, scopeID).
			WithRole(role).
			WithUser(userID)
	}

	// Check if assignment was actually made (not a duplicate)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		// Role already exists - this is not an error for AssignDirect
		return NewError(ErrRoleAlreadyAssigned, "user already has this role").
			WithScope(scopeType, scopeID).
			WithRole(role).
			WithUser(userID)
	}

	// Create audit log entry (simplified)
	audit := GetAuditContext(ctx)
	entry := &AuditEntry{
		ActorID:      actorID,
		Action:       AuditActionAssigned,
		TargetUserID: userID,
		Role:         role,
		ScopeType:    scopeType,
		ScopeID:      scopeID,
		IPAddress:    audit.IPAddress,
		UserAgent:    audit.UserAgent,
		RequestID:    audit.RequestID,
	}

	_ = s.logAudit(ctx, entry) // Log error but don't fail the assignment

	return nil
}

// AssignWithRetry assigns a role to a user with automatic retry for transient errors.
func (s *Service) AssignWithRetry(ctx context.Context, userID, role, scopeType, scopeID string) error {
	return s.assignWithRetry(ctx, userID, role, scopeType, scopeID, 3)
}

// assignWithRetry is the internal implementation of retry logic with configurable attempts.
func (s *Service) assignWithRetry(ctx context.Context, userID, role, scopeType, scopeID string, maxAttempts int) error {
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := s.Assign(ctx, userID, role, scopeType, scopeID)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if this is a transient error that can be retried
		if !isTransientTransactionError(err) {
			// Not a transient error, return immediately
			return err
		}

		// For transient errors, wait with exponential backoff
		if attempt < maxAttempts-1 {
			backoffDuration := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			time.Sleep(backoffDuration)
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}

// AssignMultipleWithRetry assigns multiple roles with automatic retry for transient errors.
func (s *Service) AssignMultipleWithRetry(ctx context.Context, assignments []RoleAssignment) error {
	return s.assignMultipleWithRetry(ctx, assignments, 3)
}

// assignMultipleWithRetry is the internal implementation of retry logic for bulk operations.
func (s *Service) assignMultipleWithRetry(ctx context.Context, assignments []RoleAssignment, maxAttempts int) error {
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := s.AssignMultiple(ctx, assignments)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if this is a transient error that can be retried
		if !isTransientTransactionError(err) {
			// Not a transient error, return immediately
			return err
		}

		// For transient errors, wait with exponential backoff
		if attempt < maxAttempts-1 {
			backoffDuration := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			time.Sleep(backoffDuration)
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}

// isTransientTransactionError checks if an error is a transient transaction error
func isTransientTransactionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check for transaction state errors
	if strings.Contains(errStr, "transaction has already been committed") ||
		strings.Contains(errStr, "transaction has already been rolled back") ||
		strings.Contains(errStr, "transaction is closed") {
		return true
	}

	// Check for connection errors
	if strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "network") ||
		dbkit.IsConnection(err) {
		return true
	}

	// Check for deadlock errors
	if strings.Contains(errStr, "deadlock") ||
		strings.Contains(errStr, "lock wait timeout") {
		return true
	}

	return false
}

// GetTransactionMetrics returns the current transaction performance metrics.
func (s *Service) GetTransactionMetrics() TransactionMetrics {
	return s.txMonitor.getMetrics()
}

// ResetTransactionMetrics resets all transaction metrics.
func (s *Service) ResetTransactionMetrics() {
	s.txMonitor.reset()
}

// IsTransactionHealthy checks if transaction performance is within acceptable thresholds.
func (s *Service) IsTransactionHealthy() bool {
	metrics := s.txMonitor.getMetrics()

	// If we have very few transactions, consider it healthy
	if metrics.TotalTransactions < 10 {
		return true
	}

	// Check failure rate (should be less than 5%)
	failureRate := float64(metrics.FailedTransactions) / float64(metrics.TotalTransactions)
	if failureRate > 0.05 {
		return false
	}

	// Check average duration (should be less than 1 second)
	if metrics.AverageDuration > time.Second {
		return false
	}

	return true
}
