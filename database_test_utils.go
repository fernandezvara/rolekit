package rolekit

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestDataHelper provides utilities for setting up test data
type TestDataHelper struct {
	service *Service
	ctx     context.Context
	t       *testing.T
}

// NewTestDataHelper creates a new test data helper with database setup
func NewTestDataHelper(t *testing.T) *TestDataHelper {
	if !RequireDatabase(t) {
		return nil
	}

	ctx := context.Background()
	service, err := SetupTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	return &TestDataHelper{
		service: service,
		ctx:     ctx,
		t:       t,
	}
}

// CreateTestUser creates a test user with a unique ID
func (h *TestDataHelper) CreateTestUser(prefix string) string {
	userID := prefix + "-" + fmt.Sprintf("%d", time.Now().UnixNano())
	return userID
}

// CreateTestOrg creates a test organization with a unique ID
func (h *TestDataHelper) CreateTestOrg(prefix string) string {
	orgID := prefix + "-" + fmt.Sprintf("%d", time.Now().UnixNano())
	return orgID
}

// CreateTestProject creates a test project with a unique ID
func (h *TestDataHelper) CreateTestProject(prefix string) string {
	projectID := prefix + "-" + fmt.Sprintf("%d", time.Now().UnixNano())
	return projectID
}

// CreateTestTeam creates a test team with a unique ID
func (h *TestDataHelper) CreateTestTeam(prefix string) string {
	teamID := prefix + "-" + fmt.Sprintf("%d", time.Now().UnixNano())
	return teamID
}

// SetupAdminUser creates and assigns an admin user
func (h *TestDataHelper) SetupAdminUser(userID, orgID string) error {
	ctx := WithActorID(h.ctx, userID)
	return h.service.Assign(ctx, userID, "super_admin", "organization", orgID)
}

// SetupProjectManager creates and assigns a project manager
func (h *TestDataHelper) SetupProjectManager(userID, orgID string) error {
	ctx := WithActorID(h.ctx, userID)
	return h.service.Assign(ctx, userID, "project_manager", "organization", orgID)
}

// SetupTeamLead creates and assigns a team lead
func (h *TestDataHelper) SetupTeamLead(userID, orgID string) error {
	ctx := WithActorID(h.ctx, userID)
	return h.service.Assign(ctx, userID, "team_lead", "organization", orgID)
}

// SetupDeveloper creates and assigns a developer
func (h *TestDataHelper) SetupDeveloper(userID, orgID string) error {
	ctx := WithActorID(h.ctx, userID)
	return h.service.Assign(ctx, userID, "developer", "organization", orgID)
}

// SetupViewer creates and assigns a viewer
func (h *TestDataHelper) SetupViewer(userID, orgID string) error {
	ctx := WithActorID(h.ctx, userID)
	return h.service.Assign(ctx, userID, "viewer", "organization", orgID)
}

// CleanupTestData cleans up test data
func (h *TestDataHelper) CleanupTestData() error {
	// This could be implemented to clean up specific test data
	// For now, we rely on unique test IDs and database cleanup
	return nil
}

// AssertRoleAssigned verifies a role is assigned
func (h *TestDataHelper) AssertRoleAssigned(userID, role, scopeType, scopeID string) {
	if !h.service.Can(h.ctx, userID, role, scopeType, scopeID) {
		h.t.Errorf("User %s should have role %s in scope %s:%s", userID, role, scopeType, scopeID)
	}
}

// AssertRoleNotAssigned verifies a role is not assigned
func (h *TestDataHelper) AssertRoleNotAssigned(userID, role, scopeType, scopeID string) {
	if h.service.Can(h.ctx, userID, role, scopeType, scopeID) {
		h.t.Errorf("User %s should not have role %s in scope %s:%s", userID, role, scopeType, scopeID)
	}
}

// AssertPermissionGranted verifies a permission is granted
func (h *TestDataHelper) AssertPermissionGranted(userID, permission, scopeType, scopeID string) {
	if !h.service.HasPermission(h.ctx, userID, permission, scopeType, scopeID) {
		h.t.Errorf("User %s should have permission %s in scope %s:%s", userID, permission, scopeType, scopeID)
	}
}

// AssertPermissionDenied verifies a permission is denied
func (h *TestDataHelper) AssertPermissionDenied(userID, permission, scopeType, scopeID string) {
	if h.service.HasPermission(h.ctx, userID, permission, scopeType, scopeID) {
		h.t.Errorf("User %s should not have permission %s in scope %s:%s", userID, permission, scopeType, scopeID)
	}
}

// AssertUserHasRoles verifies a user has specific roles
func (h *TestDataHelper) AssertUserHasRoles(userID string, expectedRoles map[string][]string) {
	userRoles, err := h.service.GetUserRoles(h.ctx, userID)
	if err != nil {
		h.t.Fatalf("Failed to get user roles: %v", err)
	}

	for scopeType, roles := range expectedRoles {
		for _, role := range roles {
			if !userRoles.HasRole(role, scopeType, "") {
				h.t.Errorf("User %s should have role %s in scope %s", userID, role, scopeType)
			}
		}
	}
}

// AssertUserDoesNotHaveRoles verifies a user does not have specific roles
func (h *TestDataHelper) AssertUserDoesNotHaveRoles(userID string, forbiddenRoles map[string][]string) {
	userRoles, err := h.service.GetUserRoles(h.ctx, userID)
	if err != nil {
		h.t.Fatalf("Failed to get user roles: %v", err)
	}

	for scopeType, roles := range forbiddenRoles {
		for _, role := range roles {
			if userRoles.HasRole(role, scopeType, "") {
				h.t.Errorf("User %s should not have role %s in scope %s", userID, role, scopeType)
			}
		}
	}
}

// AssertRoleCount verifies the number of roles a user has
func (h *TestDataHelper) AssertRoleCount(userID string, scopeType string, expectedCount int) {
	userRoles, err := h.service.GetUserRoles(h.ctx, userID)
	if err != nil {
		h.t.Fatalf("Failed to get user roles: %v", err)
	}

	actualCount := 0
	for _, assignment := range userRoles.Assignments {
		if assignment.ScopeType == scopeType {
			actualCount++
		}
	}

	if actualCount != expectedCount {
		h.t.Errorf("Expected %d roles in scope %s, got %d", expectedCount, scopeType, actualCount)
	}
}

// GetService returns the service instance
func (h *TestDataHelper) GetService() *Service {
	return h.service
}

// GetContext returns the context instance
func (h *TestDataHelper) GetContext() context.Context {
	return h.ctx
}

// GetT returns the testing.T instance
func (h *TestDataHelper) GetT() *testing.T {
	return h.t
}
