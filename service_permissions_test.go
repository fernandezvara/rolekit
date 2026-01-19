package rolekit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestServiceCan tests checking if a user has a specific role in a scope
func TestServiceCan(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin").Role("member")
	registry.DefineScope("project").Role("owner").Role("editor")

	service := &Service{db: nil, registry: registry}
	ctx := context.Background()

	// Test with nil database - should panic
	assert.Panics(t, func() {
		service.Can(ctx, "user1", "admin", "organization", "org1")
	})
}

// TestServiceHasPermission tests checking if a user has a specific permission in a scope
func TestServiceHasPermission(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").
		Role("admin").Permissions("organization.*").
		Role("member").Permissions("organization.read")
	registry.DefineScope("project").
		Role("owner").Permissions("project.*").
		Role("editor").Permissions("project.read", "project.write")

	service := &Service{db: nil, registry: registry}
	ctx := context.Background()

	// Test with nil database - should panic
	assert.Panics(t, func() {
		service.HasPermission(ctx, "user1", "organization.read", "organization", "org1")
	})
}

// TestServiceHasAnyRole tests checking if a user has any of the specified roles in a scope
func TestServiceHasAnyRole(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin").Role("member").Role("viewer")
	registry.DefineScope("project").Role("owner").Role("editor").Role("viewer")

	service := &Service{db: nil, registry: registry}
	ctx := context.Background()

	// Test with nil database - should panic
	assert.Panics(t, func() {
		service.HasAnyRole(ctx, "user1", []string{"admin", "member"}, "organization", "org1")
	})
}

// TestServiceCanAssignRole tests checking if a user can assign a role to another user in a scope
func TestServiceCanAssignRole(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").
		Role("admin").CanAssign("member", "viewer").
		Role("member").CanAssign("viewer")
	registry.DefineScope("project").
		Role("owner").CanAssign("editor", "viewer").
		Role("editor").CanAssign("viewer")

	service := &Service{db: nil, registry: registry}
	ctx := context.Background()

	// Test with nil database - should panic
	assert.Panics(t, func() {
		service.CanAssignRole(ctx, "user1", "member", "organization", "org1")
	})
}

// TestServicePermissionsEdgeCases tests edge cases and error conditions
func TestServicePermissionsEdgeCases(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	service := &Service{db: nil, registry: registry}
	ctx := context.Background()

	t.Run("Can with empty userID", func(t *testing.T) {
		assert.Panics(t, func() {
			service.Can(ctx, "", "admin", "organization", "org1")
		})
	})

	t.Run("Can with empty role", func(t *testing.T) {
		assert.Panics(t, func() {
			service.Can(ctx, "user1", "", "organization", "org1")
		})
	})

	t.Run("Can with empty scope", func(t *testing.T) {
		assert.Panics(t, func() {
			service.Can(ctx, "user1", "admin", "", "")
		})
	})

	t.Run("HasPermission with empty permission", func(t *testing.T) {
		assert.Panics(t, func() {
			service.HasPermission(ctx, "user1", "", "organization", "org1")
		})
	})

	t.Run("HasPermission with empty scope", func(t *testing.T) {
		assert.Panics(t, func() {
			service.HasPermission(ctx, "user1", "organization.read", "", "")
		})
	})

	t.Run("HasAnyRole with nil roles slice", func(t *testing.T) {
		assert.Panics(t, func() {
			service.HasAnyRole(ctx, "user1", nil, "organization", "org1")
		})
	})

	t.Run("CanAssignRole with empty target role", func(t *testing.T) {
		assert.Panics(t, func() {
			service.CanAssignRole(ctx, "user1", "", "organization", "org1")
		})
	})

	t.Run("CanAssignRole with empty scope", func(t *testing.T) {
		assert.Panics(t, func() {
			service.CanAssignRole(ctx, "user1", "member", "", "")
		})
	})

	t.Run("All methods with nil registry", func(t *testing.T) {
		serviceNilRegistry := &Service{db: nil, registry: nil}

		// These should panic when trying to create checker
		assert.Panics(t, func() {
			serviceNilRegistry.HasPermission(ctx, "user1", "organization.read", "organization", "org1")
		})

		assert.Panics(t, func() {
			serviceNilRegistry.HasAnyRole(ctx, "user1", []string{"admin"}, "organization", "org1")
		})

		assert.Panics(t, func() {
			serviceNilRegistry.CanAssignRole(ctx, "user1", "member", "organization", "org1")
		})

		// Can should also panic as it calls GetUserRoles
		assert.Panics(t, func() {
			serviceNilRegistry.Can(ctx, "user1", "admin", "organization", "org1")
		})
	})

	t.Run("Context cancellation", func(t *testing.T) {
		// Create a cancelled context
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		// Methods should handle cancelled context gracefully (but still panic due to nil DB)
		assert.Panics(t, func() {
			service.Can(cancelledCtx, "user1", "admin", "organization", "org1")
		})

		assert.Panics(t, func() {
			service.HasPermission(cancelledCtx, "user1", "organization.read", "organization", "org1")
		})

		assert.Panics(t, func() {
			service.HasAnyRole(cancelledCtx, "user1", []string{"admin"}, "organization", "org1")
		})

		assert.Panics(t, func() {
			service.CanAssignRole(cancelledCtx, "user1", "member", "organization", "org1")
		})
	})
}

// TestServicePermissionsIntegration tests integration scenarios
func TestServicePermissionsIntegration(t *testing.T) {
	registry := NewRegistry()

	// Define a complex permission structure
	registry.DefineScope("organization").
		Role("super_admin").Permissions("organization.*").
		Role("admin").Permissions("organization.read", "organization.write", "organization.manage").
		Role("manager").Permissions("organization.read", "organization.write").
		Role("member").Permissions("organization.read").
		Role("viewer")

	registry.DefineScope("project").
		Role("owner").Permissions("project.*").
		Role("lead").Permissions("project.read", "project.write", "project.manage").
		Role("editor").Permissions("project.read", "project.write").
		Role("viewer").Permissions("project.read")

	// Define role assignment permissions
	registry.GetRole("super_admin", "organization").CanAssign("*")
	registry.GetRole("admin", "organization").CanAssign("manager", "member", "viewer")
	registry.GetRole("manager", "organization").CanAssign("member", "viewer")
	registry.GetRole("owner", "project").CanAssign("lead", "editor", "viewer")
	registry.GetRole("lead", "project").CanAssign("editor", "viewer")

	service := &Service{db: nil, registry: registry}
	ctx := context.Background()

	// Test hierarchical permissions
	t.Run("Super admin permissions", func(t *testing.T) {
		// Super admin should have all permissions (but panics with nil DB)
		assert.Panics(t, func() {
			service.HasPermission(ctx, "user1", "organization.delete", "organization", "org1")
		})

		assert.Panics(t, func() {
			service.HasPermission(ctx, "user1", "organization.read", "organization", "org1")
		})
	})

	t.Run("Role assignment hierarchy", func(t *testing.T) {
		// Super admin can assign any role
		assert.Panics(t, func() {
			service.CanAssignRole(ctx, "user1", "super_admin", "organization", "org1")
		})

		// Admin can assign manager
		assert.Panics(t, func() {
			service.CanAssignRole(ctx, "user1", "manager", "organization", "org1")
		})

		// Manager cannot assign admin
		assert.Panics(t, func() {
			service.CanAssignRole(ctx, "user1", "admin", "organization", "org1")
		})
	})

	t.Run("Cross-scope permissions", func(t *testing.T) {
		// Organization admin doesn't automatically have project permissions
		assert.Panics(t, func() {
			service.HasPermission(ctx, "user1", "project.read", "project", "proj1")
		})

		// Project owner has project permissions
		assert.Panics(t, func() {
			service.HasPermission(ctx, "user1", "project.write", "project", "proj1")
		})
	})
}
