package rolekit

import "context"

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
