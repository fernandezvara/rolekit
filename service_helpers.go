package rolekit

import (
	"context"
	"database/sql"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/fernandezvara/dbkit"
)

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
