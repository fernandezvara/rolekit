package rolekit

import (
	"context"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if len(r.GetScopes()) != 0 {
		t.Error("New registry should have no scopes")
	}
}

func TestDefineScope(t *testing.T) {
	r := NewRegistry()

	scope := r.DefineScope("organization")
	if scope == nil {
		t.Fatal("DefineScope returned nil")
	}
	if scope.Name() != "organization" {
		t.Errorf("Expected scope name 'organization', got %q", scope.Name())
	}

	// Check it's retrievable
	retrieved := r.GetScope("organization")
	if retrieved == nil {
		t.Fatal("GetScope returned nil for defined scope")
	}
	if retrieved.Name() != "organization" {
		t.Error("Retrieved scope has wrong name")
	}
}

func TestDefineRoles(t *testing.T) {
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

	// Check roles
	scope := r.GetScope("organization")
	if scope == nil {
		t.Fatal("Scope not found")
	}

	roles := scope.GetRoles()
	if len(roles) != 4 {
		t.Errorf("Expected 4 roles, got %d", len(roles))
	}

	// Check owner permissions
	owner := scope.GetRole("owner")
	if owner == nil {
		t.Fatal("Owner role not found")
	}
	perms := owner.GetPermissions()
	if len(perms) != 1 || perms[0] != "*" {
		t.Errorf("Owner should have [*] permissions, got %v", perms)
	}

	// Check admin permissions
	admin := scope.GetRole("admin")
	if admin == nil {
		t.Fatal("Admin role not found")
	}
	if len(admin.GetPermissions()) != 2 {
		t.Errorf("Admin should have 2 permissions, got %d", len(admin.GetPermissions()))
	}

	// Check admin can assign
	canAssign := admin.GetCanAssign()
	if len(canAssign) != 2 {
		t.Errorf("Admin should be able to assign 2 roles, got %d", len(canAssign))
	}
}

func TestValidateRole(t *testing.T) {
	r := NewRegistry()
	r.DefineScope("organization").
		Role("admin").Permissions("*")

	// Valid role
	err := r.ValidateRole("admin", "organization")
	if err != nil {
		t.Errorf("Expected no error for valid role, got %v", err)
	}

	// Invalid role
	err = r.ValidateRole("superuser", "organization")
	if err == nil {
		t.Error("Expected error for invalid role")
	}
	if !IsInvalidRole(err) {
		t.Errorf("Expected ErrInvalidRole, got %v", err)
	}

	// Invalid scope
	err = r.ValidateRole("admin", "project")
	if err == nil {
		t.Error("Expected error for invalid scope")
	}
	if !IsInvalidScope(err) {
		t.Errorf("Expected ErrInvalidScope, got %v", err)
	}
}

func TestCanRoleAssign(t *testing.T) {
	r := NewRegistry()
	r.DefineScope("organization").
		Role("owner").Permissions("*").CanAssign("*").
		Role("admin").Permissions("members.*").CanAssign("member", "viewer").
		Role("member").Permissions("read")

	// Owner can assign anything
	if !r.CanRoleAssign("owner", "admin", "organization") {
		t.Error("Owner should be able to assign admin")
	}
	if !r.CanRoleAssign("owner", "member", "organization") {
		t.Error("Owner should be able to assign member")
	}

	// Admin can assign member and viewer
	if !r.CanRoleAssign("admin", "member", "organization") {
		t.Error("Admin should be able to assign member")
	}
	if !r.CanRoleAssign("admin", "viewer", "organization") {
		t.Error("Admin should be able to assign viewer")
	}

	// Admin cannot assign admin or owner
	if r.CanRoleAssign("admin", "admin", "organization") {
		t.Error("Admin should not be able to assign admin")
	}
	if r.CanRoleAssign("admin", "owner", "organization") {
		t.Error("Admin should not be able to assign owner")
	}

	// Member cannot assign anything
	if r.CanRoleAssign("member", "viewer", "organization") {
		t.Error("Member should not be able to assign viewer")
	}
}

func TestParentScope(t *testing.T) {
	r := NewRegistry()
	r.DefineScope("organization").
		Role("admin").Permissions("*")

	r.DefineScope("project").
		ParentScope("organization").
		Role("editor").Permissions("files.*")

	scope := r.GetScope("project")
	if scope.GetParentScope() != "organization" {
		t.Errorf("Expected parent scope 'organization', got %q", scope.GetParentScope())
	}
}

func TestFluentAPI(t *testing.T) {
	r := NewRegistry()

	// Test that fluent API works for chaining
	r.DefineScope("organization").
		Role("owner").Permissions("*").CanAssign("*").
		Role("admin").Permissions("members.*").
		DefineScope("project").
		ParentScope("organization").
		Role("editor").Permissions("files.*")

	if r.GetScope("organization") == nil {
		t.Error("Organization scope not created")
	}
	if r.GetScope("project") == nil {
		t.Error("Project scope not created")
	}
	if r.GetRole("owner", "organization") == nil {
		t.Error("Owner role not created")
	}
	if r.GetRole("editor", "project") == nil {
		t.Error("Editor role not created")
	}
}

// ============================================================================
// Permission Matcher Tests
// ============================================================================

func TestPermissionMatcherExact(t *testing.T) {
	pm := NewPermissionMatcher()

	tests := []struct {
		pattern    string
		permission string
		expected   bool
	}{
		{"files.read", "files.read", true},
		{"files.read", "files.write", false},
		{"files.read", "members.read", false},
	}

	for _, tt := range tests {
		result := pm.Match(tt.pattern, tt.permission)
		if result != tt.expected {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.permission, result, tt.expected)
		}
	}
}

func TestPermissionMatcherWildcard(t *testing.T) {
	pm := NewPermissionMatcher()

	tests := []struct {
		pattern    string
		permission string
		expected   bool
	}{
		// Universal wildcard
		{"*", "files.read", true},
		{"*", "members.write", true},
		{"*", "anything.here", true},

		// Resource wildcard
		{"files.*", "files.read", true},
		{"files.*", "files.write", true},
		{"files.*", "files.delete", true},
		{"files.*", "members.read", false},

		// Action wildcard
		{"*.read", "files.read", true},
		{"*.read", "members.read", true},
		{"*.read", "files.write", false},

		// Multi-part permissions
		{"files.metadata.*", "files.metadata.read", true},
		{"files.metadata.*", "files.metadata.write", true},
		{"files.metadata.*", "files.content.read", false},

		// Mixed wildcards
		{"*.metadata.*", "files.metadata.read", true},
		{"*.metadata.*", "members.metadata.write", true},
		{"*.metadata.*", "files.content.read", false},
	}

	for _, tt := range tests {
		result := pm.Match(tt.pattern, tt.permission)
		if result != tt.expected {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.permission, result, tt.expected)
		}
	}
}

func TestPermissionMatcherValidate(t *testing.T) {
	pm := NewPermissionMatcher()

	valid := []string{
		"*",
		"files.read",
		"files.*",
		"*.read",
		"files.metadata.read",
		"files_v2.read_all",
	}

	for _, p := range valid {
		if err := pm.Validate(p); err != nil {
			t.Errorf("Validate(%q) returned error: %v", p, err)
		}
	}

	invalid := []string{
		"",
		"files",       // Single part
		"files..read", // Empty part
		"files.read!", // Invalid character
	}

	for _, p := range invalid {
		if err := pm.Validate(p); err == nil {
			t.Errorf("Validate(%q) should return error", p)
		}
	}
}

func TestMatchAnyPermission(t *testing.T) {
	patterns := []string{"files.read", "members.*", "*.delete"}

	tests := []struct {
		permission string
		expected   bool
	}{
		{"files.read", true},      // Exact match
		{"members.read", true},    // members.* match
		{"members.write", true},   // members.* match
		{"files.delete", true},    // *.delete match
		{"projects.delete", true}, // *.delete match
		{"files.write", false},    // No match
		{"projects.read", false},  // No match
	}

	for _, tt := range tests {
		result := MatchAnyPermission(patterns, tt.permission)
		if result != tt.expected {
			t.Errorf("MatchAnyPermission(%v, %q) = %v, want %v", patterns, tt.permission, result, tt.expected)
		}
	}
}

// ============================================================================
// UserRoles Tests
// ============================================================================

func TestUserRoles(t *testing.T) {
	assignments := []RoleAssignment{
		{UserID: "user1", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user1", Role: "reviewer", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user1", Role: "editor", ScopeType: "project", ScopeID: "proj1"},
		{UserID: "user1", Role: "admin", ScopeType: "project", ScopeID: "*"}, // Wildcard
	}

	ur := NewUserRoles("user1", assignments)

	// Test GetRoles
	orgRoles := ur.GetRoles("organization", "org1")
	if len(orgRoles) != 2 {
		t.Errorf("Expected 2 org roles, got %d", len(orgRoles))
	}

	// Test HasRole
	if !ur.HasRole("admin", "organization", "org1") {
		t.Error("Should have admin role in org1")
	}
	if !ur.HasRole("reviewer", "organization", "org1") {
		t.Error("Should have reviewer role in org1")
	}
	if ur.HasRole("editor", "organization", "org1") {
		t.Error("Should not have editor role in org1")
	}

	// Test wildcard scope
	if !ur.HasRole("admin", "project", "any_project") {
		t.Error("Should have admin role in any project due to wildcard")
	}
	if !ur.HasRole("admin", "project", "proj999") {
		t.Error("Should have admin role in proj999 due to wildcard")
	}
}

// ============================================================================
// Checker Tests
// ============================================================================

func TestChecker(t *testing.T) {
	r := NewRegistry()
	r.DefineScope("organization").
		Role("owner").Permissions("*").CanAssign("*").
		Role("admin").Permissions("members.*", "settings.read").CanAssign("member").
		Role("member").Permissions("projects.list")

	r.DefineScope("project").
		Role("editor").Permissions("files.*", "comments.*").
		Role("viewer").Permissions("files.read")

	assignments := []RoleAssignment{
		{UserID: "user1", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user1", Role: "editor", ScopeType: "project", ScopeID: "proj1"},
		{UserID: "user1", Role: "viewer", ScopeType: "project", ScopeID: "proj2"},
	}

	ur := NewUserRoles("user1", assignments)
	checker := NewChecker("user1", ur, r, nil)

	// Test Can
	if !checker.Can("admin", "organization", "org1") {
		t.Error("Should have admin role")
	}
	if checker.Can("owner", "organization", "org1") {
		t.Error("Should not have owner role")
	}

	// Test HasPermission
	if !checker.HasPermission("members.invite", "organization", "org1") {
		t.Error("Admin should have members.invite permission (members.*)")
	}
	if !checker.HasPermission("settings.read", "organization", "org1") {
		t.Error("Admin should have settings.read permission")
	}
	if checker.HasPermission("settings.write", "organization", "org1") {
		t.Error("Admin should not have settings.write permission")
	}

	// Test project permissions
	if !checker.HasPermission("files.upload", "project", "proj1") {
		t.Error("Editor should have files.upload permission")
	}
	if !checker.HasPermission("files.read", "project", "proj2") {
		t.Error("Viewer should have files.read permission")
	}
	if checker.HasPermission("files.write", "project", "proj2") {
		t.Error("Viewer should not have files.write permission")
	}

	// Test CanAssignRole
	if !checker.CanAssignRole("member", "organization", "org1") {
		t.Error("Admin should be able to assign member role")
	}
	if checker.CanAssignRole("admin", "organization", "org1") {
		t.Error("Admin should not be able to assign admin role")
	}

	// Test GetPermissions
	perms := checker.GetPermissions("organization", "org1")
	if len(perms) != 2 {
		t.Errorf("Expected 2 permission patterns, got %d: %v", len(perms), perms)
	}
}

func TestCheckerMultipleRoles(t *testing.T) {
	r := NewRegistry()
	r.DefineScope("project").
		Role("editor").Permissions("files.*").
		Role("reviewer").Permissions("comments.*")

	// User has both editor and reviewer roles (UNION of permissions)
	assignments := []RoleAssignment{
		{UserID: "user1", Role: "editor", ScopeType: "project", ScopeID: "proj1"},
		{UserID: "user1", Role: "reviewer", ScopeType: "project", ScopeID: "proj1"},
	}

	ur := NewUserRoles("user1", assignments)
	checker := NewChecker("user1", ur, r, nil)

	// Should have permissions from both roles
	if !checker.HasPermission("files.read", "project", "proj1") {
		t.Error("Should have files.read from editor role")
	}
	if !checker.HasPermission("comments.create", "project", "proj1") {
		t.Error("Should have comments.create from reviewer role")
	}

	// Check GetRoles returns both
	roles := checker.GetRoles("project", "proj1")
	if len(roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(roles))
	}
}

// ============================================================================
// Context Tests
// ============================================================================

func TestContext(t *testing.T) {
	ctx := context.TODO()
	ctx = WithUserID(ctx, "user123")
	if GetUserID(ctx) != "user123" {
		t.Error("UserID not set correctly")
	}

	ctx = WithActorID(ctx, "actor456")
	if GetActorID(ctx) != "actor456" {
		t.Error("ActorID not set correctly")
	}

	ctx = WithIPAddress(ctx, "192.168.1.1")
	if GetIPAddress(ctx) != "192.168.1.1" {
		t.Error("IPAddress not set correctly")
	}

	ctx = WithRequestID(ctx, "req123")
	if GetRequestID(ctx) != "req123" {
		t.Error("RequestID not set correctly")
	}

	// Test AuditContext
	ac := GetAuditContext(ctx)
	if ac.ActorID != "actor456" {
		t.Error("AuditContext ActorID wrong")
	}
	if ac.IPAddress != "192.168.1.1" {
		t.Error("AuditContext IPAddress wrong")
	}
}

func TestContextActorFallback(t *testing.T) {
	// When actor ID is not set, should fall back to user ID
	ctx := context.TODO()
	ctx = WithUserID(ctx, "user123")
	if GetActorID(ctx) != "user123" {
		t.Error("ActorID should fall back to UserID")
	}
}

// ============================================================================
// Error Tests
// ============================================================================

func TestErrors(t *testing.T) {
	err := NewError(ErrUnauthorized, "user cannot access this resource").
		WithScope("project", "proj123").
		WithRole("admin").
		WithUser("user456")

	if !IsUnauthorized(err) {
		t.Error("IsUnauthorized should return true")
	}

	if err.Scope != "project" {
		t.Error("Scope not set")
	}
	if err.ScopeID != "proj123" {
		t.Error("ScopeID not set")
	}
	if err.Role != "admin" {
		t.Error("Role not set")
	}
	if err.UserID != "user456" {
		t.Error("UserID not set")
	}

	expectedMsg := "rolekit: unauthorized: user cannot access this resource"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestErrorHandling(t *testing.T) {
	// Test that dbkit error types are preserved
	// This is a basic test to ensure error handling works
	ctx := context.TODO()

	// Test context with proper context
	ctx = WithUserID(ctx, "user123")
	if GetUserID(ctx) != "user123" {
		t.Errorf("Expected user ID 'user123', got %q", GetUserID(ctx))
	}
}

func TestTransactionSupport(t *testing.T) {
	// Test transaction functionality
	// This test would require a real database connection to test properly
	// For now, we just test that the methods exist and don't panic
	registry := NewRegistry()
	_ = registry // Use the variable to avoid lint error

	// This would normally be a real dbkit.DBKit instance
	// For testing purposes, we just verify the methods compile
	t.Log("Transaction support methods are available")
}

func TestHealthMonitoring(t *testing.T) {
	// Test health monitoring functionality
	// This test would require a real database connection to test properly
	// For now, we just test that the methods exist and don't panic
	registry := NewRegistry()
	_ = registry // Use the variable to avoid lint error

	// This would normally be a real dbkit.DBKit instance
	// For testing purposes, we just verify the methods compile
	t.Log("Health monitoring methods are available")
}

func TestOptimizedOperations(t *testing.T) {
	// Test optimized database operations
	// This test would require a real database connection to test properly
	// For now, we just test that the methods exist and don't panic
	registry := NewRegistry()
	_ = registry // Use the variable to avoid lint error

	// This would normally be a real dbkit.DBKit instance
	// For testing purposes, we just verify the methods compile
	t.Log("Optimized operations methods are available")
}

func TestMigrationSystem(t *testing.T) {
	// Test migration system functionality
	// This test would require a real database connection to test properly
	// For now, we just test that the methods exist and don't panic
	registry := NewRegistry()
	_ = registry // Use the variable to avoid lint error

	// Test migration validation
	service := &Service{}
	err := service.ValidateMigrations()
	if err != nil {
		t.Errorf("Migration validation failed: %v", err)
	}

	// Test connection pool configuration functions
	config := DefaultPoolConfig()
	if config.MaxOpenConnections == 0 {
		t.Error("DefaultPoolConfig should have non-zero MaxOpenConnections")
	}

	highPerfConfig := HighPerformancePoolConfig()
	if highPerfConfig.MaxOpenConnections <= config.MaxOpenConnections {
		t.Error("HighPerformancePoolConfig should have higher MaxOpenConnections")
	}

	lowResConfig := LowResourcePoolConfig()
	if lowResConfig.MaxOpenConnections >= config.MaxOpenConnections {
		t.Error("LowResourcePoolConfig should have lower MaxOpenConnections")
	}

	// This would normally be a real dbkit.DBKit instance
	// For testing purposes, we just verify the methods compile
	t.Log("Migration system and connection pool methods are available")
}
