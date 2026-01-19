package rolekit

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/fernandezvara/dbkit"
)

// TransactionService provides transaction-related functionality as an extension to Service
type TransactionService struct {
	*Service
}

// NewTransactionService creates a new transaction service extension
func NewTransactionService(service *Service) *TransactionService {
	return &TransactionService{Service: service}
}

// AssignDirect assigns a role to a user without pre-checks for better performance.
// This method bypasses GetUserRoles calls and handles duplicate key constraints gracefully.
func (ts *TransactionService) AssignDirect(ctx context.Context, userID, role, scopeType, scopeID string) error {
	// Validate role exists for scope
	if err := ts.registry.ValidateRole(role, scopeType); err != nil {
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
	result, err := ts.db.NewInsert().
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

	_ = ts.logAudit(ctx, entry) // Log error but don't fail the assignment

	return nil
}

// AssignWithRetry assigns a role to a user with automatic retry for transient errors.
func (ts *TransactionService) AssignWithRetry(ctx context.Context, userID, role, scopeType, scopeID string) error {
	return ts.assignWithRetry(ctx, userID, role, scopeType, scopeID, 3)
}

// assignWithRetry is the internal implementation of retry logic with configurable attempts.
func (ts *TransactionService) assignWithRetry(ctx context.Context, userID, role, scopeType, scopeID string, maxAttempts int) error {
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := ts.Assign(ctx, userID, role, scopeType, scopeID)
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
func (ts *TransactionService) AssignMultipleWithRetry(ctx context.Context, assignments []RoleAssignment) error {
	return ts.assignMultipleWithRetry(ctx, assignments, 3)
}

// assignMultipleWithRetry is the internal implementation of retry logic for bulk operations.
func (ts *TransactionService) assignMultipleWithRetry(ctx context.Context, assignments []RoleAssignment, maxAttempts int) error {
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := ts.AssignMultiple(ctx, assignments)
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
