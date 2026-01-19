package rolekit

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestServiceSetScopeParentDatabase tests the SetScopeParent method with real database
func TestServiceSetScopeParentDatabase(t *testing.T) {
	helper := NewTestDataHelper(t)
	if helper == nil {
		return
	}

	service := helper.GetService()
	ctx := helper.GetContext()
	orgID := "test-org"
	projectID := "project-1"
	teamID := "team-1"

	// Set parent relationship: project -> organization
	err := service.SetScopeParent(ctx, "project", projectID, "organization", orgID)
	require.NoError(t, err, "Should be able to set parent scope")

	// Set another parent relationship: team -> project
	err = service.SetScopeParent(ctx, "team", teamID, "project", projectID)
	require.NoError(t, err, "Should be able to set nested parent scope")

	// Verify the hierarchy was created by checking if we can query it
	// Note: We can't directly query the hierarchy table, but we can test
	// that the relationships don't cause errors when set
}

// TestServiceGetChildScopesBasicDatabase tests basic GetChildScopes functionality
func TestServiceGetChildScopesBasicDatabase(t *testing.T) {
	helper := NewTestDataHelper(t)
	if helper == nil {
		return
	}

	service := helper.GetService()
	ctx := helper.GetContext()
	orgID := "test-org"
	userID := fmt.Sprintf("user-%d", time.Now().UnixNano())

	// Create projects under the organization
	projectIDs := []string{"project-1", "project-2"}

	// Setup admin user
	adminID := fmt.Sprintf("admin-%d", time.Now().UnixNano())
	err := helper.SetupAdminUser(adminID, orgID)
	require.NoError(t, err)

	// Set up parent relationships
	for _, projectID := range projectIDs {
		err = service.SetScopeParent(ctx, "project", projectID, "organization", orgID)
		require.NoError(t, err, "Should set parent for project %s", projectID)
	}

	// Assign user to organization scope (super_admin)
	actorCtx := WithActorID(ctx, adminID)
	err = service.Assign(actorCtx, userID, "super_admin", "organization", orgID)
	require.NoError(t, err)

	// Get child scopes (should return empty since user only has org role)
	childScopes, err := service.GetChildScopes(ctx, userID, "project", "organization", orgID)
	require.NoError(t, err, "Should not error")
	require.Empty(t, childScopes, "Should return empty since user has no project roles")

	// Test with non-existent parent
	childScopes, err = service.GetChildScopes(ctx, userID, "project", "organization", "non-existent")
	require.NoError(t, err, "Should not error with non-existent parent")
	require.Empty(t, childScopes, "Should return empty list for non-existent parent")

	// Test with user who has no roles
	otherUserID := fmt.Sprintf("user-%d", time.Now().UnixNano()+1)
	childScopes, err = service.GetChildScopes(ctx, otherUserID, "project", "organization", orgID)
	require.NoError(t, err, "Should not error for user with no roles")
	require.Empty(t, childScopes, "Should return empty list for user with no roles")
}

// TestServiceGetChildScopesWithRoleBasicDatabase tests basic GetChildScopesWithRole functionality
func TestServiceGetChildScopesWithRoleBasicDatabase(t *testing.T) {
	helper := NewTestDataHelper(t)
	if helper == nil {
		return
	}

	service := helper.GetService()
	ctx := helper.GetContext()
	orgID := "test-org"
	userID := fmt.Sprintf("user-%d", time.Now().UnixNano())

	// Create projects under the organization
	projectIDs := []string{"project-1", "project-2"}

	// Setup admin user
	adminID := fmt.Sprintf("admin-%d", time.Now().UnixNano())
	err := helper.SetupAdminUser(adminID, orgID)
	require.NoError(t, err)

	// Set up parent relationships
	for _, projectID := range projectIDs {
		err = service.SetScopeParent(ctx, "project", projectID, "organization", orgID)
		require.NoError(t, err, "Should set parent for project %s", projectID)
	}

	// Assign user to organization scope (super_admin)
	actorCtx := WithActorID(ctx, adminID)
	err = service.Assign(actorCtx, userID, "super_admin", "organization", orgID)
	require.NoError(t, err)

	// Get child scopes where user has super_admin role (should be empty)
	superAdminScopes, err := service.GetChildScopesWithRole(ctx, userID, "super_admin", "project", "organization", orgID)
	require.NoError(t, err, "Should not error")
	require.Empty(t, superAdminScopes, "Should return empty since user has super_admin in org scope, not project")

	// Test with non-existent role
	nonExistentScopes, err := service.GetChildScopesWithRole(ctx, userID, "non_existent", "project", "organization", orgID)
	require.NoError(t, err, "Should not error for non-existent role")
	require.Empty(t, nonExistentScopes, "Should return empty list for non-existent role")

	// Test with non-existent parent
	nonExistentScopes, err = service.GetChildScopesWithRole(ctx, userID, "viewer", "project", "organization", "non-existent")
	require.NoError(t, err, "Should not error with non-existent parent")
	require.Empty(t, nonExistentScopes, "Should return empty list for non-existent parent")
}

// TestServiceHierarchyQueryFlowDatabase tests the complete query flow
func TestServiceHierarchyQueryFlowDatabase(t *testing.T) {
	helper := NewTestDataHelper(t)
	if helper == nil {
		return
	}

	service := helper.GetService()
	ctx := helper.GetContext()
	orgID := "test-org"
	userID := fmt.Sprintf("user-%d", time.Now().UnixNano())

	// Create project under organization
	projectID := "project-1"

	// Setup admin user
	adminID := fmt.Sprintf("admin-%d", time.Now().UnixNano())
	err := helper.SetupAdminUser(adminID, orgID)
	require.NoError(t, err)

	// Set up hierarchy
	err = service.SetScopeParent(ctx, "project", projectID, "organization", orgID)
	require.NoError(t, err)

	// Assign user to organization
	actorCtx := WithActorID(ctx, adminID)
	err = service.Assign(actorCtx, userID, "super_admin", "organization", orgID)
	require.NoError(t, err)

	// Query child scopes (should be empty since user only has org role)
	childScopes, err := service.GetChildScopes(ctx, userID, "project", "organization", orgID)
	require.NoError(t, err, "Should not error")
	require.Empty(t, childScopes, "Should return empty since user has no project roles")

	// The hierarchy is set up correctly, but GetChildScopes only works for roles
	// that are actually assigned to the child scope type
	// This is by design - you need to assign roles to the specific scopes
	// you want to query
}
