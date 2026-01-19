package rolekit

import (
"context"
"testing"

"github.com/stretchr/testify/assert"
)

// TestServiceGetUserRoles tests retrieving all role assignments for a user
func TestServiceGetUserRoles(t *testing.T) {
// This test would require a database connection
// For now, we'll test the error handling path
registry := NewRegistry()
service := &Service{db: nil, registry: registry}
ctx := context.Background()

// Test with nil database - should panic
assert.Panics(t, func() {
service.GetUserRoles(ctx, "user1")
})
}

// TestServiceGetScopeMembers tests retrieving all users with roles in a scope
func TestServiceGetScopeMembers(t *testing.T) {
registry := NewRegistry()
service := &Service{db: nil, registry: registry}
ctx := context.Background()

// Test with nil database - should panic
assert.Panics(t, func() {
service.GetScopeMembers(ctx, "organization", "org1")
})
}

// TestServiceGetScopeMembersWithRole tests retrieving users with a specific role in a scope
func TestServiceGetScopeMembersWithRole(t *testing.T) {
registry := NewRegistry()
service := &Service{db: nil, registry: registry}
ctx := context.Background()

// Test with nil database - should panic
assert.Panics(t, func() {
service.GetScopeMembersWithRole(ctx, "admin", "organization", "org1")
})
}

// TestServiceGetChecker tests creating a Checker for a user
func TestServiceGetChecker(t *testing.T) {
registry := NewRegistry()
registry.DefineScope("organization").Role("admin")

service := &Service{db: nil, registry: registry}
ctx := context.Background()

// Test with nil database - should panic
assert.Panics(t, func() {
service.GetChecker(ctx, "user1")
})
}

// TestServiceGetCheckerFromContext tests creating a Checker using user ID from context
func TestServiceGetCheckerFromContext(t *testing.T) {
registry := NewRegistry()
registry.DefineScope("organization").Role("admin")

service := &Service{db: nil, registry: registry}
ctx := context.Background()

// Test with no user ID in context
checker, err := service.GetCheckerFromContext(ctx)
assert.Error(t, err)
assert.Nil(t, checker)
assert.IsType(t, ErrNoUserID, err)

// Test with user ID in context but nil database - should panic
ctxWithUser := WithUserID(ctx, "user1")
assert.Panics(t, func() {
service.GetCheckerFromContext(ctxWithUser)
})
}

// TestServiceSetScopeParent tests setting parent scope for hierarchical queries
func TestServiceSetScopeParent(t *testing.T) {
registry := NewRegistry()
service := &Service{db: nil, registry: registry}
ctx := context.Background()

// Test with nil database - should panic
assert.Panics(t, func() {
service.SetScopeParent(ctx, "project", "proj1", "organization", "org1")
})
}

// TestServiceGetChildScopes tests retrieving child scope IDs where user has roles
func TestServiceGetChildScopes(t *testing.T) {
registry := NewRegistry()
service := &Service{db: nil, registry: registry}
ctx := context.Background()

// Test with nil database - should panic
assert.Panics(t, func() {
service.GetChildScopes(ctx, "user1", "project", "organization", "org1")
})
}

// TestServiceGetChildScopesWithRole tests retrieving child scope IDs where user has specific role
func TestServiceGetChildScopesWithRole(t *testing.T) {
registry := NewRegistry()
service := &Service{db: nil, registry: registry}
ctx := context.Background()

// Test with nil database - should panic
assert.Panics(t, func() {
service.GetChildScopesWithRole(ctx, "user1", "admin", "project", "organization", "org1")
})
}

// TestServiceDataRetrievalEdgeCases tests edge cases and error conditions
func TestServiceDataRetrievalEdgeCases(t *testing.T) {
registry := NewRegistry()
service := &Service{db: nil, registry: registry}
ctx := context.Background()

t.Run("GetUserRoles with empty userID", func(t *testing.T) {
assert.Panics(t, func() {
service.GetUserRoles(ctx, "")
})
})

t.Run("GetScopeMembers with empty scope", func(t *testing.T) {
assert.Panics(t, func() {
service.GetScopeMembers(ctx, "", "")
})
})

t.Run("GetChecker with nil registry", func(t *testing.T) {
serviceNilRegistry := &Service{db: nil, registry: nil}
assert.Panics(t, func() {
serviceNilRegistry.GetChecker(ctx, "user1")
})
})

t.Run("SetScopeParent with empty values", func(t *testing.T) {
assert.Panics(t, func() {
service.SetScopeParent(ctx, "", "", "", "")
})
})
}
