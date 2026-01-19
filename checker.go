package rolekit

// Checker provides permission checking capabilities for a specific user.
// It is typically created by the Service and stored in context for use in handlers.
type Checker struct {
	userID   string
	roles    *UserRoles
	registry *Registry
	service  *Service
}

// NewChecker creates a new Checker for a user.
func NewChecker(userID string, roles *UserRoles, registry *Registry, service *Service) *Checker {
	return &Checker{
		userID:   userID,
		roles:    roles,
		registry: registry,
		service:  service,
	}
}

// UserID returns the user ID this checker is for.
func (c *Checker) UserID() string {
	return c.userID
}

// Can checks if the user has a specific role in a scope.
//
// Example:
//
//	if checker.Can("admin", "organization", orgID) {
//	    // User is an admin in this organization
//	}
func (c *Checker) Can(role, scopeType, scopeID string) bool {
	return c.roles.HasRole(role, scopeType, scopeID)
}

// HasAnyRole checks if the user has any of the specified roles in a scope.
//
// Example:
//
//	if checker.HasAnyRole([]string{"admin", "owner"}, "organization", orgID) {
//	    // User is either admin or owner
//	}
func (c *Checker) HasAnyRole(roles []string, scopeType, scopeID string) bool {
	for _, role := range roles {
		if c.roles.HasRole(role, scopeType, scopeID) {
			return true
		}
	}
	return false
}

// HasAllRoles checks if the user has all of the specified roles in a scope.
//
// Example:
//
//	if checker.HasAllRoles([]string{"member", "reviewer"}, "project", projectID) {
//	    // User has both member and reviewer roles
//	}
func (c *Checker) HasAllRoles(roles []string, scopeType, scopeID string) bool {
	for _, role := range roles {
		if !c.roles.HasRole(role, scopeType, scopeID) {
			return false
		}
	}
	return true
}

// HasPermission checks if the user has a specific permission in a scope.
// This resolves the user's roles to their permissions and checks for a match.
//
// Example:
//
//	if checker.HasPermission("files.upload", "project", projectID) {
//	    // User can upload files to this project
//	}
func (c *Checker) HasPermission(permission, scopeType, scopeID string) bool {
	// Get all roles for this scope
	roles := c.roles.GetRoles(scopeType, scopeID)
	if len(roles) == 0 {
		return false
	}

	// Get all permissions from all roles (UNION)
	permissions := c.GetPermissions(scopeType, scopeID)

	// Check if any permission matches
	return MatchAnyPermission(permissions, permission)
}

// HasAnyPermission checks if the user has any of the specified permissions.
//
// Example:
//
//	if checker.HasAnyPermission([]string{"files.upload", "files.write"}, "project", projectID) {
//	    // User has at least one of these permissions
//	}
func (c *Checker) HasAnyPermission(permissions []string, scopeType, scopeID string) bool {
	for _, perm := range permissions {
		if c.HasPermission(perm, scopeType, scopeID) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if the user has all of the specified permissions.
//
// Example:
//
//	if checker.HasAllPermissions([]string{"files.read", "files.write"}, "project", projectID) {
//	    // User has both permissions
//	}
func (c *Checker) HasAllPermissions(permissions []string, scopeType, scopeID string) bool {
	for _, perm := range permissions {
		if !c.HasPermission(perm, scopeType, scopeID) {
			return false
		}
	}
	return true
}

// GetRoles returns all roles the user has in a scope.
//
// Example:
//
//	roles := checker.GetRoles("project", projectID)
//	// roles might be ["editor", "reviewer"]
func (c *Checker) GetRoles(scopeType, scopeID string) []string {
	return c.roles.GetRoles(scopeType, scopeID)
}

// GetPermissions returns all permissions the user has in a scope.
// This is the UNION of permissions from all roles.
//
// Example:
//
//	perms := checker.GetPermissions("project", projectID)
//	// perms might be ["files.*", "comments.read", "comments.write"]
func (c *Checker) GetPermissions(scopeType, scopeID string) []string {
	roles := c.roles.GetRoles(scopeType, scopeID)
	if len(roles) == 0 {
		return nil
	}

	// Collect all permissions from all roles
	permSet := make(map[string]bool)
	for _, role := range roles {
		perms := c.registry.GetPermissions(role, scopeType)
		for _, p := range perms {
			permSet[p] = true
		}
	}

	result := make([]string, 0, len(permSet))
	for p := range permSet {
		result = append(result, p)
	}
	return result
}

// CanAssignRole checks if the user can assign a role to another user in a scope.
// This checks the "CanAssign" configuration of the user's roles.
//
// Example:
//
//	if checker.CanAssignRole("member", "organization", orgID) {
//	    // User can assign the "member" role in this organization
//	}
func (c *Checker) CanAssignRole(targetRole, scopeType, scopeID string) bool {
	// Get user's roles in this scope
	roles := c.roles.GetRoles(scopeType, scopeID)
	if len(roles) == 0 {
		return false
	}

	// Check if any of the user's roles can assign the target role
	for _, userRole := range roles {
		if c.registry.CanRoleAssign(userRole, targetRole, scopeType) {
			return true
		}
	}
	return false
}

// GetAssignableRoles returns all roles the user can assign in a scope.
//
// Example:
//
//	roles := checker.GetAssignableRoles("organization", orgID)
//	// roles might be ["member", "viewer"]
func (c *Checker) GetAssignableRoles(scopeType, scopeID string) []string {
	// Get user's roles in this scope
	userRoles := c.roles.GetRoles(scopeType, scopeID)
	if len(userRoles) == 0 {
		return nil
	}

	// Get scope definition
	scope := c.registry.GetScope(scopeType)
	if scope == nil {
		return nil
	}

	// Collect all assignable roles
	assignable := make(map[string]bool)
	for _, userRole := range userRoles {
		roleDef := scope.GetRole(userRole)
		if roleDef == nil {
			continue
		}

		for _, canAssign := range roleDef.GetCanAssign() {
			if canAssign == "*" {
				// Can assign any role in this scope
				for _, r := range scope.GetRoles() {
					assignable[r] = true
				}
			} else {
				assignable[canAssign] = true
			}
		}
	}

	result := make([]string, 0, len(assignable))
	for r := range assignable {
		result = append(result, r)
	}
	return result
}

// HasRoleInAnyScope checks if the user has a role in any entity of a scope type.
// Useful for checking if user has access to "any project" without specifying which one.
//
// Example:
//
//	if checker.HasRoleInAnyScope("editor", "project") {
//	    // User is editor in at least one project
//	}
func (c *Checker) HasRoleInAnyScope(role, scopeType string) bool {
	for _, assignment := range c.roles.Assignments {
		if assignment.ScopeType == scopeType && assignment.Role == role {
			return true
		}
	}
	return false
}

// GetScopesWithRole returns all scope IDs where the user has a specific role.
//
// Example:
//
//	projectIDs := checker.GetScopesWithRole("editor", "project")
//	// projectIDs might be ["proj_123", "proj_456"]
func (c *Checker) GetScopesWithRole(role, scopeType string) []string {
	var scopeIDs []string
	for _, assignment := range c.roles.Assignments {
		if assignment.ScopeType == scopeType && assignment.Role == role {
			scopeIDs = append(scopeIDs, assignment.ScopeID)
		}
	}
	return scopeIDs
}

// GetScopesWithAnyRole returns all scope IDs where the user has any role.
//
// Example:
//
//	projectIDs := checker.GetScopesWithAnyRole("project")
//	// projectIDs might be ["proj_123", "proj_456", "proj_789"]
func (c *Checker) GetScopesWithAnyRole(scopeType string) []string {
	scopeIDSet := make(map[string]bool)
	for _, assignment := range c.roles.Assignments {
		if assignment.ScopeType == scopeType {
			scopeIDSet[assignment.ScopeID] = true
		}
	}

	result := make([]string, 0, len(scopeIDSet))
	for id := range scopeIDSet {
		result = append(result, id)
	}
	return result
}

// IsEmpty returns true if the user has no role assignments.
func (c *Checker) IsEmpty() bool {
	return len(c.roles.Assignments) == 0
}
