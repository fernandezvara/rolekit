package rolekit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCheckerNewChecker tests the checker constructor
func TestCheckerNewChecker(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	service := &Service{registry: registry}
	roles := &UserRoles{}

	checker := NewChecker("user123", roles, registry, service)

	assert.Equal(t, "user123", checker.UserID())
	assert.Equal(t, roles, checker.roles)
	assert.Equal(t, registry, checker.registry)
	assert.Equal(t, service, checker.service)
}

// TestCheckerCan tests role checking
func TestCheckerCan(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	service := &Service{registry: registry}
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
	}
	roles := NewUserRoles("user123", assignments)

	checker := NewChecker("user123", roles, registry, service)

	// Test existing role
	assert.True(t, checker.Can("admin", "organization", "org1"))

	// Test non-existing role
	assert.False(t, checker.Can("member", "organization", "org1"))
	assert.False(t, checker.Can("admin", "organization", "org2"))
	assert.False(t, checker.Can("admin", "project", "proj1"))
}

// TestCheckerHasAnyRole tests checking for any of multiple roles
func TestCheckerHasAnyRole(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin").Role("member")

	service := &Service{registry: registry}
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user123", Role: "viewer", ScopeType: "project", ScopeID: "proj1"},
	}
	roles := NewUserRoles("user123", assignments)

	checker := NewChecker("user123", roles, registry, service)

	// Test with one matching role
	assert.True(t, checker.HasAnyRole([]string{"admin", "owner"}, "organization", "org1"))

	// Test with no matching roles
	assert.False(t, checker.HasAnyRole([]string{"owner", "manager"}, "organization", "org1"))

	// Test with different scope
	assert.False(t, checker.HasAnyRole([]string{"admin"}, "project", "proj1"))
	assert.True(t, checker.HasAnyRole([]string{"viewer"}, "project", "proj1"))

	// Test empty role list
	assert.False(t, checker.HasAnyRole([]string{}, "organization", "org1"))
}

// TestCheckerHasAllRoles tests checking for all required roles
func TestCheckerHasAllRoles(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin").Role("member").Role("viewer")

	service := &Service{registry: registry}
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user123", Role: "member", ScopeType: "organization", ScopeID: "org1"},
	}
	roles := NewUserRoles("user123", assignments)

	checker := NewChecker("user123", roles, registry, service)

	// Test with all roles present
	assert.True(t, checker.HasAllRoles([]string{"admin", "member"}, "organization", "org1"))

	// Test with missing role
	assert.False(t, checker.HasAllRoles([]string{"admin", "member", "owner"}, "organization", "org1"))

	// Test with no roles
	assert.False(t, checker.HasAllRoles([]string{"admin"}, "project", "proj1"))

	// Test empty role list (should return true - vacuously true)
	assert.True(t, checker.HasAllRoles([]string{}, "organization", "org1"))
}

// TestCheckerHasPermission tests permission checking
func TestCheckerHasPermission(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").
		Role("admin").Permissions("organization.*").
		Role("member").Permissions("organization.read").
		Role("viewer").Permissions("organization.read")

	service := &Service{registry: registry}
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user123", Role: "member", ScopeType: "organization", ScopeID: "org2"},
	}
	roles := NewUserRoles("user123", assignments)

	checker := NewChecker("user123", roles, registry, service)

	// Test admin permissions
	assert.True(t, checker.HasPermission("organization.read", "organization", "org1"))
	assert.True(t, checker.HasPermission("organization.write", "organization", "org1"))
	assert.True(t, checker.HasPermission("organization.delete", "organization", "org1"))

	// Test member permissions
	assert.True(t, checker.HasPermission("organization.read", "organization", "org2"))
	assert.False(t, checker.HasPermission("organization.write", "organization", "org2"))

	// Test no permissions
	assert.False(t, checker.HasPermission("organization.read", "organization", "org3"))
	assert.False(t, checker.HasPermission("project.read", "project", "proj1"))
}

// TestCheckerHasAnyPermission tests checking for any of multiple permissions
func TestCheckerHasAnyPermission(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").
		Role("admin").Permissions("organization.*").
		Role("member").Permissions("organization.read")

	service := &Service{registry: registry}
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "member", ScopeType: "organization", ScopeID: "org1"},
	}
	roles := NewUserRoles("user123", assignments)

	checker := NewChecker("user123", roles, registry, service)

	// Test with one matching permission
	assert.True(t, checker.HasAnyPermission([]string{"organization.read", "organization.write"}, "organization", "org1"))

	// Test with no matching permissions
	assert.False(t, checker.HasAnyPermission([]string{"organization.write", "organization.delete"}, "organization", "org1"))

	// Test empty permission list
	assert.False(t, checker.HasAnyPermission([]string{}, "organization", "org1"))
}

// TestCheckerHasAllPermissions tests checking for all required permissions
func TestCheckerHasAllPermissions(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").
		Role("admin").Permissions("organization.read", "organization.write")

	service := &Service{registry: registry}
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
	}
	roles := NewUserRoles("user123", assignments)

	checker := NewChecker("user123", roles, registry, service)

	// Test with all permissions present
	assert.True(t, checker.HasAllPermissions([]string{"organization.read", "organization.write"}, "organization", "org1"))

	// Test with missing permission
	assert.False(t, checker.HasAllPermissions([]string{"organization.read", "organization.write", "organization.delete"}, "organization", "org1"))

	// Test empty permission list (should return true - vacuously true)
	assert.True(t, checker.HasAllPermissions([]string{}, "organization", "org1"))
}

// TestCheckerGetRoles tests getting user roles in a scope
func TestCheckerGetRoles(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin").Role("member")
	registry.DefineScope("project").Role("editor").Role("viewer")

	service := &Service{registry: registry}
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user123", Role: "member", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user123", Role: "editor", ScopeType: "project", ScopeID: "proj1"},
	}
	roles := NewUserRoles("user123", assignments)

	checker := NewChecker("user123", roles, registry, service)

	// Test getting roles in organization
	orgRoles := checker.GetRoles("organization", "org1")
	assert.ElementsMatch(t, []string{"admin", "member"}, orgRoles)

	// Test getting roles in project
	projRoles := checker.GetRoles("project", "proj1")
	assert.Equal(t, []string{"editor"}, projRoles)

	// Test no roles
	assert.Empty(t, checker.GetRoles("organization", "org2"))
	assert.Empty(t, checker.GetRoles("project", "proj2"))
}

// TestCheckerGetPermissions tests getting user permissions in a scope
func TestCheckerGetPermissions(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").
		Role("admin").Permissions("organization.read", "organization.write").
		Role("member").Permissions("organization.read")

	service := &Service{registry: registry}
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user123", Role: "member", ScopeType: "organization", ScopeID: "org1"},
	}
	roles := NewUserRoles("user123", assignments)

	checker := NewChecker("user123", roles, registry, service)

	// Test getting permissions (union of all roles)
	perms := checker.GetPermissions("organization", "org1")
	assert.ElementsMatch(t, []string{"organization.read", "organization.write"}, perms)

	// Test no permissions
	assert.Nil(t, checker.GetPermissions("organization", "org2"))
}

// TestCheckerCanAssignRole tests role assignment checking
func TestCheckerCanAssignRole(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").
		Role("admin").CanAssign("member", "viewer").
		Role("member").CanAssign("viewer")

	service := &Service{registry: registry}
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user123", Role: "member", ScopeType: "organization", ScopeID: "org2"},
	}
	roles := NewUserRoles("user123", assignments)

	checker := NewChecker("user123", roles, registry, service)

	// Test admin can assign member and viewer
	assert.True(t, checker.CanAssignRole("member", "organization", "org1"))
	assert.True(t, checker.CanAssignRole("viewer", "organization", "org1"))

	// Test member can assign viewer
	assert.True(t, checker.CanAssignRole("viewer", "organization", "org2"))

	// Test member cannot assign admin
	assert.False(t, checker.CanAssignRole("admin", "organization", "org2"))

	// Test no roles in scope
	assert.False(t, checker.CanAssignRole("member", "organization", "org3"))
}

// TestCheckerGetAssignableRoles tests getting assignable roles
func TestCheckerGetAssignableRoles(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").
		Role("admin").CanAssign("member", "viewer").
		Role("member").CanAssign("viewer").
		Role("viewer")

	service := &Service{registry: registry}
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
	}
	roles := NewUserRoles("user123", assignments)

	checker := NewChecker("user123", roles, registry, service)

	// Test getting assignable roles for admin
	assignable := checker.GetAssignableRoles("organization", "org1")
	assert.ElementsMatch(t, []string{"member", "viewer"}, assignable)

	// Test no assignable roles
	assert.Nil(t, checker.GetAssignableRoles("organization", "org2"))

	// Test wildcard assignment
	registry.DefineScope("project").
		Role("owner").CanAssign("*").
		Role("editor").Role("viewer")

	assignments2 := []RoleAssignment{
		{UserID: "user123", Role: "owner", ScopeType: "project", ScopeID: "proj1"},
	}
	roles2 := NewUserRoles("user123", assignments2)

	checker2 := NewChecker("user123", roles2, registry, service)
	assignable2 := checker2.GetAssignableRoles("project", "proj1")
	assert.ElementsMatch(t, []string{"owner", "editor", "viewer"}, assignable2)
}

// TestCheckerHasRoleInAnyScope tests checking if user has role in any scope
func TestCheckerHasRoleInAnyScope(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")
	registry.DefineScope("project").Role("editor")

	service := &Service{registry: registry}
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org2"},
		{UserID: "user123", Role: "editor", ScopeType: "project", ScopeID: "proj1"},
	}
	roles := NewUserRoles("user123", assignments)

	checker := NewChecker("user123", roles, registry, service)

	// Test existing role in any scope
	assert.True(t, checker.HasRoleInAnyScope("admin", "organization"))
	assert.True(t, checker.HasRoleInAnyScope("editor", "project"))

	// Test non-existing role
	assert.False(t, checker.HasRoleInAnyScope("member", "organization"))
	assert.False(t, checker.HasRoleInAnyScope("admin", "project"))
}

// TestCheckerGetScopesWithRole tests getting scopes where user has specific role
func TestCheckerGetScopesWithRole(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")
	registry.DefineScope("project").Role("editor")

	service := &Service{registry: registry}
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org2"},
		{UserID: "user123", Role: "editor", ScopeType: "project", ScopeID: "proj1"},
		{UserID: "user123", Role: "editor", ScopeType: "project", ScopeID: "proj2"},
	}
	roles := NewUserRoles("user123", assignments)

	checker := NewChecker("user123", roles, registry, service)

	// Test getting scopes with admin role
	adminScopes := checker.GetScopesWithRole("admin", "organization")
	assert.ElementsMatch(t, []string{"org1", "org2"}, adminScopes)

	// Test getting scopes with editor role
	editorScopes := checker.GetScopesWithRole("editor", "project")
	assert.ElementsMatch(t, []string{"proj1", "proj2"}, editorScopes)

	// Test no scopes
	assert.Empty(t, checker.GetScopesWithRole("member", "organization"))
	assert.Empty(t, checker.GetScopesWithRole("admin", "project"))
}

// TestCheckerGetScopesWithAnyRole tests getting scopes where user has any role
func TestCheckerGetScopesWithAnyRole(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin").Role("member")
	registry.DefineScope("project").Role("editor")

	service := &Service{registry: registry}
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user123", Role: "member", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org2"},
		{UserID: "user123", Role: "editor", ScopeType: "project", ScopeID: "proj1"},
	}
	roles := NewUserRoles("user123", assignments)

	checker := NewChecker("user123", roles, registry, service)

	// Test getting organization scopes
	orgScopes := checker.GetScopesWithAnyRole("organization")
	assert.ElementsMatch(t, []string{"org1", "org2"}, orgScopes)

	// Test getting project scopes
	projScopes := checker.GetScopesWithAnyRole("project")
	assert.Equal(t, []string{"proj1"}, projScopes)

	// Test no scopes
	assert.Empty(t, checker.GetScopesWithAnyRole("team"))
}

// TestCheckerIsEmpty tests checking if user has no roles
func TestCheckerIsEmpty(t *testing.T) {
	registry := NewRegistry()
	service := &Service{registry: registry}

	// Test with no roles
	emptyRoles := NewUserRoles("user123", []RoleAssignment{})
	emptyChecker := NewChecker("user123", emptyRoles, registry, service)
	assert.True(t, emptyChecker.IsEmpty())

	// Test with roles
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
	}
	roles := NewUserRoles("user123", assignments)
	checker := NewChecker("user123", roles, registry, service)
	assert.False(t, checker.IsEmpty())
}

// TestCheckerEdgeCases tests edge cases and error conditions
func TestCheckerEdgeCases(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").
		Role("admin").Permissions("organization.*")

	service := &Service{registry: registry}

	t.Run("Empty roles", func(t *testing.T) {
		emptyRoles := NewUserRoles("user123", []RoleAssignment{})
		checker := NewChecker("user123", emptyRoles, registry, service)
		assert.False(t, checker.Can("admin", "organization", "org1"))
		assert.Empty(t, checker.GetRoles("organization", "org1"))
		assert.Nil(t, checker.GetPermissions("organization", "org1"))
		assert.True(t, checker.IsEmpty())
	})

	t.Run("Undefined scope", func(t *testing.T) {
		assignments := []RoleAssignment{
			{UserID: "user123", Role: "admin", ScopeType: "undefined", ScopeID: "test"},
		}
		roles := NewUserRoles("user123", assignments)
		checker := NewChecker("user123", roles, registry, service)
		// Should not panic, just return empty/false
		assert.Empty(t, checker.GetPermissions("undefined", "test"))
		assert.Nil(t, checker.GetAssignableRoles("undefined", "test"))
	})

	t.Run("Undefined role", func(t *testing.T) {
		assignments := []RoleAssignment{
			{UserID: "user123", Role: "undefined", ScopeType: "organization", ScopeID: "org1"},
		}
		roles := NewUserRoles("user123", assignments)
		checker := NewChecker("user123", roles, registry, service)
		// Should not panic, just return empty/false
		assert.Empty(t, checker.GetPermissions("organization", "org1"))
		assert.False(t, checker.CanAssignRole("admin", "organization", "org1"))
	})
}
