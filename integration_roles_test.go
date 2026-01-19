package rolekit

import (
	"context"
	"testing"
)

// TestBasicRoleAssignment tests basic role assignment and checking with real database
func TestBasicRoleAssignment(t *testing.T) {
	if !RequireDatabase(t) {
		return
	}

	ctx := context.Background()
	service, err := SetupTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create sample users with unique IDs to avoid conflicts
	users := []struct {
		id    string
		email string
		name  string
	}{
		{"test-admin-" + t.Name(), "admin@test.com", "Admin User"},
		{"test-manager-" + t.Name(), "manager@test.com", "Project Manager"},
		{"test-dev-" + t.Name(), "dev@test.com", "Developer"},
		{"test-viewer-" + t.Name(), "viewer@test.com", "Viewer"},
	}

	// Create sample organization
	orgID := "test-org-" + t.Name()

	// Test role assignments
	testCases := []struct {
		name    string
		userID  string
		role    string
		scope   string
		scopeID string
		actorID string
		wantErr bool
	}{
		{
			name:    "Assign super admin role",
			userID:  users[0].id,
			role:    "super_admin",
			scope:   "organization",
			scopeID: orgID,
			actorID: users[0].id, // Self-assignment for bootstrap
			wantErr: false,
		},
		{
			name:    "Assign project manager role",
			userID:  users[1].id,
			role:    "project_manager",
			scope:   "organization",
			scopeID: orgID,
			actorID: users[0].id, // Admin assigns
			wantErr: false,
		},
		{
			name:    "Assign developer role",
			userID:  users[2].id,
			role:    "developer",
			scope:   "organization",
			scopeID: orgID,
			actorID: users[1].id, // Manager assigns
			wantErr: false,
		},
		{
			name:    "Assign viewer role",
			userID:  users[3].id,
			role:    "viewer",
			scope:   "organization",
			scopeID: orgID,
			actorID: users[1].id, // Manager assigns
			wantErr: false,
		},
		{
			name:    "Assign invalid role",
			userID:  users[3].id,
			role:    "invalid_role",
			scope:   "organization",
			scopeID: orgID,
			actorID: users[0].id,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set actor context
			ctx = WithActorID(ctx, tc.actorID)

			// Attempt role assignment
			err := service.Assign(ctx, tc.userID, tc.role, tc.scope, tc.scopeID)

			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error but assignment succeeded")
				}
				return
			}

			if err != nil {
				t.Errorf("Failed to assign role: %v", err)
				return
			}

			// Verify the role was assigned
			if !service.Can(ctx, tc.userID, tc.role, tc.scope, tc.scopeID) {
				t.Errorf("User %s should have role %s in scope %s", tc.userID, tc.role, tc.scope)
			}

			// Verify GetUserRoles returns the assigned role
			userRoles, err := service.GetUserRoles(ctx, tc.userID)
			if err != nil {
				t.Errorf("Failed to get user roles: %v", err)
				return
			}

			if !userRoles.HasRole(tc.role, tc.scope, tc.scopeID) {
				t.Errorf("GetUserRoles should show user has role %s in scope %s", tc.role, tc.scope)
			}
		})
	}
}

// TestPermissionChecking tests permission checking with real database
func TestPermissionChecking(t *testing.T) {
	if !RequireDatabase(t) {
		return
	}

	ctx := context.Background()
	service, err := SetupTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create users and organization
	adminID := "test-admin-" + t.Name()
	managerID := "test-manager-" + t.Name()
	devID := "test-dev-" + t.Name()
	orgID := "test-org-" + t.Name()

	// Set up roles
	ctx = WithActorID(ctx, adminID)

	if err := service.Assign(ctx, adminID, "super_admin", "organization", orgID); err != nil {
		t.Fatalf("Failed to assign admin role: %v", err)
	}

	if err := service.Assign(ctx, managerID, "project_manager", "organization", orgID); err != nil {
		t.Fatalf("Failed to assign manager role: %v", err)
	}

	if err := service.Assign(ctx, devID, "developer", "organization", orgID); err != nil {
		t.Fatalf("Failed to assign developer role: %v", err)
	}

	// Test permission checks
	testCases := []struct {
		name       string
		userID     string
		permission string
		scope      string
		scopeID    string
		want       bool
	}{
		{
			name:       "Super admin has all permissions",
			userID:     adminID,
			permission: "organization.delete",
			scope:      "organization",
			scopeID:    orgID,
			want:       true,
		},
		{
			name:       "Project manager can manage projects",
			userID:     managerID,
			permission: "project.create",
			scope:      "organization",
			scopeID:    orgID,
			want:       true,
		},
		{
			name:       "Developer can read teams",
			userID:     devID,
			permission: "team.read",
			scope:      "organization",
			scopeID:    orgID,
			want:       true,
		},
		{
			name:       "Developer cannot delete organizations",
			userID:     devID,
			permission: "organization.delete",
			scope:      "organization",
			scopeID:    orgID,
			want:       false,
		},
		{
			name:       "Non-existent permission",
			userID:     devID,
			permission: "non.existent.permission",
			scope:      "organization",
			scopeID:    orgID,
			want:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			has := service.HasPermission(ctx, tc.userID, tc.permission, tc.scope, tc.scopeID)
			if has != tc.want {
				t.Errorf("HasPermission() = %v, want %v", has, tc.want)
			}
		})
	}
}

// TestRoleRevocation tests role revocation with real database
func TestRoleRevocation(t *testing.T) {
	if !RequireDatabase(t) {
		return
	}

	ctx := context.Background()
	service, err := SetupTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create user and organization
	userID := "test-user-" + t.Name()
	orgID := "test-org-" + t.Name()

	// Assign role
	ctx = WithActorID(ctx, userID)
	if err := service.Assign(ctx, userID, "admin", "organization", orgID); err != nil {
		t.Fatalf("Failed to assign role: %v", err)
	}

	// Verify role is assigned
	if !service.Can(ctx, userID, "admin", "organization", orgID) {
		t.Error("Role should be assigned before revocation")
	}

	// Revoke role
	if err := service.Revoke(ctx, userID, "admin", "organization", orgID); err != nil {
		t.Errorf("Failed to revoke role: %v", err)
	}

	// Verify role is revoked
	if service.Can(ctx, userID, "admin", "organization", orgID) {
		t.Error("Role should be revoked")
	}

	// Verify GetUserRoles reflects the change
	userRoles, err := service.GetUserRoles(ctx, userID)
	if err != nil {
		t.Errorf("Failed to get user roles: %v", err)
	}

	if userRoles.HasRole("admin", "organization", orgID) {
		t.Error("GetUserRoles should not show revoked role")
	}
}

// TestBulkOperations tests bulk role operations with real database
func TestBulkOperations(t *testing.T) {
	if !RequireDatabase(t) {
		return
	}

	ctx := context.Background()
	service, err := SetupTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create users and organization
	adminID := "test-admin-" + t.Name()
	orgID := "test-org-" + t.Name()

	// Set up admin
	ctx = WithActorID(ctx, adminID)
	if err := service.Assign(ctx, adminID, "super_admin", "organization", orgID); err != nil {
		t.Fatalf("Failed to assign admin role: %v", err)
	}

	// Prepare bulk assignments
	assignments := []RoleAssignment{
		{UserID: "user1", Role: "project_manager", ScopeType: "organization", ScopeID: orgID},
		{UserID: "user2", Role: "team_lead", ScopeType: "organization", ScopeID: orgID},
		{UserID: "user3", Role: "developer", ScopeType: "organization", ScopeID: orgID},
		{UserID: "user4", Role: "viewer", ScopeType: "organization", ScopeID: orgID},
	}

	// Test bulk assignment
	if err := service.AssignMultiple(ctx, assignments); err != nil {
		t.Errorf("Failed to assign multiple roles: %v", err)
	}

	// Verify all roles were assigned
	for _, assignment := range assignments {
		if !service.Can(ctx, assignment.UserID, assignment.Role, assignment.ScopeType, assignment.ScopeID) {
			t.Errorf("User %s should have role %s", assignment.UserID, assignment.Role)
		}
	}

	// Prepare bulk revocations
	revocations := []RoleRevocation{
		{UserID: "user1", Role: "project_manager", ScopeType: "organization", ScopeID: orgID},
		{UserID: "user2", Role: "team_lead", ScopeType: "organization", ScopeID: orgID},
	}

	// Test bulk revocation
	if err := service.RevokeMultiple(ctx, revocations); err != nil {
		t.Errorf("Failed to revoke multiple roles: %v", err)
	}

	// Verify roles were revoked
	for _, revocation := range revocations {
		if service.Can(ctx, revocation.UserID, revocation.Role, revocation.ScopeType, revocation.ScopeID) {
			t.Errorf("User %s should not have role %s after revocation", revocation.UserID, revocation.Role)
		}
	}

	// Verify other roles remain
	if !service.Can(ctx, "user3", "developer", "organization", orgID) {
		t.Error("User3 should still have developer role")
	}
	if !service.Can(ctx, "user4", "viewer", "organization", orgID) {
		t.Error("User4 should still have viewer role")
	}
}
