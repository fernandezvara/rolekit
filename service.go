package rolekit

import (
	"context"

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
