package rolekit

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/fernandezvara/dbkit"
)

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
