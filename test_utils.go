package rolekit

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fernandezvara/dbkit"
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

// NewDBKit creates a new dbkit instance (helper to avoid import issues)
func NewDBKit(databaseURL string) (*dbkit.DBKit, error) {
	return dbkit.New(dbkit.Config{URL: databaseURL})
}

// isDatabaseAvailable checks if the test database is available
func isDatabaseAvailable() bool {
	// Get database URL from environment
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = getTestDatabaseURL()
	}

	// Try to connect to database
	db, err := NewDBKit(dbURL)
	if err != nil {
		return false
	}
	defer db.Close()

	// Try to ping the database
	err = db.PingContext(context.Background())
	return err == nil
}

// RequireDatabase skips the test if database is not available
// Use this as: if !RequireDatabase(t) { return }
func RequireDatabase(t interface{}) bool {
	// Check if we have a testing.TB interface
	type tb interface {
		Skip(args ...interface{})
		Skipf(format string, args ...interface{})
		Log(args ...interface{})
	}

	tester, ok := t.(tb)
	if !ok {
		return isDatabaseAvailable()
	}

	if !isDatabaseAvailable() {
		tester.Log("Database not available - skipping test")
		tester.Log("Run 'make start' to start the test database")
		tester.Skip("database not available")
		return false
	}

	return true
}

// getTestDatabaseURL returns the database URL for testing
func getTestDatabaseURL() string {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		return "postgres://postgres:password@localhost:5418/rolekit_test?sslmode=disable"
	}
	return dbURL
}

// SetupTestDatabase creates a test database connection and runs migrations
func SetupTestDatabase(ctx context.Context) (*Service, error) {
	if !isDatabaseAvailable() {
		return nil, fmt.Errorf("database not available - run 'make start' to start the test database")
	}

	dbURL := getTestDatabaseURL()

	// Initialize dbkit
	db, err := NewDBKit(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create role registry
	registry := NewRegistry()

	// Define test roles
	defineTestRoles(registry)

	// Create service
	service := NewService(registry, db)

	// Run migrations
	result, err := db.Migrate(ctx, service.Migrations())
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if len(result.Applied) > 0 {
		// Log applied migrations for debugging
		for _, migration := range result.Applied {
			fmt.Printf("Applied migration: %s\n", migration.ID)
		}
	}

	return service, nil
}

// defineTestRoles sets up the role hierarchy for testing
func defineTestRoles(registry *Registry) {
	// Define scopes and roles using fluent API
	registry.DefineScope("organization").
		Role("super_admin").Permissions("*").CanAssign("*").
		Role("admin").Permissions("organization.*", "project.*", "team.*").CanAssign("project_manager", "team_lead").
		Role("project_manager").Permissions("project.*", "team.*", "task.*").CanAssign("team_lead", "developer", "viewer").
		Role("team_lead").Permissions("team.*", "task.*").CanAssign("developer").
		Role("developer").Permissions("team.read", "task.*").
		Role("viewer").Permissions("team.read", "task.read")

	registry.DefineScope("project").
		Role("project_manager").Permissions("project.*").CanAssign("developer", "viewer").
		Role("team_lead").Permissions("project.read", "team.*", "task.*").CanAssign("developer").
		Role("developer").Permissions("project.read", "team.read", "task.*").
		Role("viewer").Permissions("project.read", "team.read", "task.read")

	registry.DefineScope("team").
		Role("team_lead").Permissions("team.*", "task.*").CanAssign("developer").
		Role("developer").Permissions("team.read", "task.*").
		Role("viewer").Permissions("team.read", "task.read")
}
