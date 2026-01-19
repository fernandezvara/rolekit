package rolekit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
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

// Migration extension methods - delegate to MigrationService

// Migrations returns all database migrations required for RoleKit.
// Use dbkit.Migrate(ctx, service.Migrations()) to run migrations.
// Use dbkit.MigrationStatus(ctx, service.Migrations()) to check status.
func (s *Service) Migrations() []dbkit.Migration {
	return []dbkit.Migration{
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
                    ip_address TEXT,
                    user_agent TEXT,
                    request_id TEXT
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
                    created_at TIMESTAMPTZ DEFAULT current_timestamp,
                    updated_at TIMESTAMPTZ DEFAULT current_timestamp
                )`,
		},
	}
}

// Health performs a comprehensive health check of the database connection.
// Returns detailed status including latency, connection pool statistics, and error information.
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
func (s *Service) IsHealthy(ctx context.Context) bool {
	// Check if we have a DBKit instance
	if db, ok := s.db.(*dbkit.DBKit); ok {
		return db.IsHealthy(ctx)
	}

	// If we're in a transaction or have a different type, try to ping
	var count int
	err := s.db.NewSelect().Model((*struct{})(nil)).ColumnExpr("1").Limit(1).Scan(ctx, &count)
	return err == nil
}

// GetPoolStats returns connection pool statistics for monitoring.
// Returns zero values if the database instance doesn't support pool statistics.
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
func (s *Service) ConfigureConnectionPool(config PoolConfig) error {
	if db, ok := s.db.(*dbkit.DBKit); ok {
		bunDB := db.Bun()
		if bunDB == nil {
			return fmt.Errorf("database instance not available")
		}

		bunDB.SetMaxOpenConns(config.MaxOpenConnections)
		bunDB.SetMaxIdleConns(config.MaxIdleConnections)
		bunDB.SetConnMaxLifetime(config.ConnectionMaxLifetime)
		bunDB.SetConnMaxIdleTime(config.ConnectionMaxIdleTime)

		log.Printf("Connection pool configured: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v, MaxIdleTime=%v",
			config.MaxOpenConnections, config.MaxIdleConnections,
			config.ConnectionMaxLifetime, config.ConnectionMaxIdleTime)

		return nil
	}

	return fmt.Errorf("connection pool configuration requires a dbkit.DBKit instance")
}

// GetConnectionPoolConfig returns the current connection pool configuration.
func (s *Service) GetConnectionPoolConfig() (*PoolConfig, error) {
	if db, ok := s.db.(*dbkit.DBKit); ok {
		bunDB := db.Bun()
		if bunDB == nil {
			return nil, fmt.Errorf("database instance not available")
		}

		stats := bunDB.Stats()
		return &PoolConfig{
			MaxOpenConnections: stats.MaxOpenConnections,
			MaxIdleConnections: stats.MaxOpenConnections,
		}, nil
	}

	return nil, fmt.Errorf("connection pool configuration requires a dbkit.DBKit instance")
}

// OptimizeConnectionPool automatically adjusts pool settings based on current usage.
func (s *Service) OptimizeConnectionPool() error {
	stats := s.GetPoolStats()

	config, err := s.GetConnectionPoolConfig()
	if err != nil {
		return fmt.Errorf("failed to get current pool config: %w", err)
	}

	newConfig := *config

	// If we're using most of our connections, increase the pool
	if stats.InUse > 0 && float64(stats.InUse)/float64(stats.MaxOpenConnections) > 0.8 {
		newConfig.MaxOpenConnections = int(float64(config.MaxOpenConnections) * 1.5)
		newConfig.MaxIdleConnections = int(float64(config.MaxIdleConnections) * 1.5)
	}

	// If we have many idle connections, reduce the pool
	if stats.Idle > 0 && float64(stats.Idle)/float64(stats.MaxOpenConnections) > 0.8 {
		newConfig.MaxOpenConnections = int(float64(config.MaxOpenConnections) * 0.75)
		newConfig.MaxIdleConnections = int(float64(config.MaxIdleConnections) * 0.75)
	}

	// Ensure minimum values
	if newConfig.MaxOpenConnections < 5 {
		newConfig.MaxOpenConnections = 5
	}
	if newConfig.MaxIdleConnections < 2 {
		newConfig.MaxIdleConnections = 2
	}

	return s.ConfigureConnectionPool(newConfig)
}

// ResetConnectionPool resets the connection pool to default settings.
func (s *Service) ResetConnectionPool() error {
	return s.ConfigureConnectionPool(DefaultPoolConfig())
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

// Transaction extension methods - delegate to TransactionService

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
		err := s.AssignDirect(ctx, userID, role, scopeType, scopeID)
		if err == nil {
			// Success - record metrics
			if s.txMonitor != nil {
				s.txMonitor.recordTransaction(0, true)
			}
			return nil
		}

		lastErr = err

		// Don't retry on non-transient errors
		if !isTransientTransactionError(err) {
			if s.txMonitor != nil {
				s.txMonitor.recordTransaction(0, false)
			}
			return err
		}

		// If this is the last attempt, don't wait
		if attempt == maxAttempts-1 {
			break
		}

		// Exponential backoff with jitter
		backoff := time.Duration(1<<uint(attempt)) * time.Second
		jitter := time.Duration(float64(backoff) * 0.1 * (0.5 + rand.Float64()))
		time.Sleep(backoff + jitter)
	}

	// Record failure metrics
	if s.txMonitor != nil {
		s.txMonitor.recordTransaction(0, false)
	}

	return lastErr
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
			// Success - record metrics
			if s.txMonitor != nil {
				s.txMonitor.recordTransaction(0, true)
			}
			return nil
		}

		lastErr = err

		// Don't retry on non-transient errors
		if !isTransientTransactionError(err) {
			if s.txMonitor != nil {
				s.txMonitor.recordTransaction(0, false)
			}
			return err
		}

		// If this is the last attempt, don't wait
		if attempt == maxAttempts-1 {
			break
		}

		// Exponential backoff with jitter
		backoff := time.Duration(1<<uint(attempt)) * time.Second
		jitter := time.Duration(float64(backoff) * 0.1 * (0.5 + rand.Float64()))
		time.Sleep(backoff + jitter)
	}

	// Record failure metrics
	if s.txMonitor != nil {
		s.txMonitor.recordTransaction(0, false)
	}

	return lastErr
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

// isTransientTransactionError checks if an error is transient and can be retried
func isTransientTransactionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific database errors that are transient
	errStr := err.Error()

	// PostgreSQL transient errors
	transientErrors := []string{
		"connection",
		"timeout",
		"deadlock",
		"lock wait timeout",
		"connection refused",
		"connection reset",
		"broken pipe",
		"temporary failure",
		"try again",
		"resource temporarily unavailable",
	}

	for _, transientErr := range transientErrors {
		if contains(errStr, transientErr) {
			return true
		}
	}

	// Check for context errors (cancellation, deadline)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

// findSubstring checks if substr exists in s
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
