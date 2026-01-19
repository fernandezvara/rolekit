package rolekit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRegistryNewRegistryBasic validates NewRegistry basics (moved from rolekit_test.go).
func TestRegistryNewRegistryBasic(t *testing.T) {
	r := NewRegistry()
	assert.NotNil(t, r)
	assert.Empty(t, r.GetScopes())
}

// TestRegistryDefineScopeBasic validates DefineScope basics (moved from rolekit_test.go).
func TestRegistryDefineScopeBasic(t *testing.T) {
	r := NewRegistry()

	scope := r.DefineScope("organization")
	assert.NotNil(t, scope)
	assert.Equal(t, "organization", scope.Name())

	retrieved := r.GetScope("organization")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "organization", retrieved.Name())
}

// TestRegistryDefineRolesBasic validates role definitions (moved from rolekit_test.go).
func TestRegistryDefineRolesBasic(t *testing.T) {
	r := NewRegistry()

	r.DefineScope("organization").
		Role("owner").
		Permissions("*").
		CanAssign("*").
		Role("admin").
		Permissions("members.*", "settings.*").
		CanAssign("member", "viewer").
		Role("member").
		Permissions("projects.create", "projects.list").
		Role("viewer").
		Permissions("projects.list")

	scope := r.GetScope("organization")
	assert.NotNil(t, scope)

	roles := scope.GetRoles()
	assert.Len(t, roles, 4)

	owner := scope.GetRole("owner")
	assert.NotNil(t, owner)
	assert.Equal(t, []string{"*"}, owner.GetPermissions())

	admin := scope.GetRole("admin")
	assert.NotNil(t, admin)
	assert.Len(t, admin.GetPermissions(), 2)
	assert.Len(t, admin.GetCanAssign(), 2)
}

// TestRegistryValidateRoleBasic validates ValidateRole behavior (moved from rolekit_test.go).
func TestRegistryValidateRoleBasic(t *testing.T) {
	r := NewRegistry()
	r.DefineScope("organization").Role("admin").Permissions("*")

	err := r.ValidateRole("admin", "organization")
	assert.NoError(t, err)

	err = r.ValidateRole("superuser", "organization")
	assert.Error(t, err)
	assert.True(t, IsInvalidRole(err))

	err = r.ValidateRole("admin", "project")
	assert.Error(t, err)
	assert.True(t, IsInvalidScope(err))
}

// TestRegistryCanRoleAssignBasic validates CanRoleAssign behavior (moved from rolekit_test.go).
func TestRegistryCanRoleAssignBasic(t *testing.T) {
	r := NewRegistry()
	r.DefineScope("organization").
		Role("owner").Permissions("*").CanAssign("*").
		Role("admin").Permissions("members.*").CanAssign("member", "viewer").
		Role("member").Permissions("read")

	assert.True(t, r.CanRoleAssign("owner", "admin", "organization"))
	assert.True(t, r.CanRoleAssign("owner", "member", "organization"))
	assert.True(t, r.CanRoleAssign("admin", "member", "organization"))
	assert.True(t, r.CanRoleAssign("admin", "viewer", "organization"))
	assert.False(t, r.CanRoleAssign("admin", "admin", "organization"))
	assert.False(t, r.CanRoleAssign("admin", "owner", "organization"))
	assert.False(t, r.CanRoleAssign("member", "viewer", "organization"))
}

// TestRegistryParentScopeBasic validates parent scope (moved from rolekit_test.go).
func TestRegistryParentScopeBasic(t *testing.T) {
	r := NewRegistry()
	r.DefineScope("organization").Role("admin").Permissions("*")

	r.DefineScope("project").
		ParentScope("organization").
		Role("editor").Permissions("files.*")

	scope := r.GetScope("project")
	assert.Equal(t, "organization", scope.GetParentScope())
}

// TestRegistryFluentAPIBasic validates fluent API chaining (moved from rolekit_test.go).
func TestRegistryFluentAPIBasic(t *testing.T) {
	r := NewRegistry()

	r.DefineScope("organization").
		Role("owner").Permissions("*").CanAssign("*").
		Role("admin").Permissions("members.*").
		DefineScope("project").
		ParentScope("organization").
		Role("editor").Permissions("files.*")

	assert.NotNil(t, r.GetScope("organization"))
	assert.NotNil(t, r.GetScope("project"))
	assert.NotNil(t, r.GetRole("owner", "organization"))
	assert.NotNil(t, r.GetRole("editor", "project"))
}
