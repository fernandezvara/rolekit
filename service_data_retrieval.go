package rolekit

import (
	"context"

	"github.com/fernandezvara/dbkit"
)

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

	// Try to insert, ignore if it already exists
	result, err := s.db.NewInsert().Model(hierarchy).Exec(ctx)
	if err != nil {
		// Check if it's a duplicate key error (PostgreSQL error code 23505)
		if dbkit.IsDuplicate(err) {
			// Already exists, just continue
		} else {
			return dbkit.WithErr(result, err, "SetScopeParent").Err()
		}
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
