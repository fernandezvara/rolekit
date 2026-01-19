package rolekit

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/fernandezvara/dbkit"
)

// isDatabaseAvailable checks if the test database is available
func isDatabaseAvailable() bool {
	// Get database URL from environment
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		return false
	}

	// Try to connect with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return false
	}
	defer db.Close()

	// Try to ping the database
	err = db.PingContext(ctx)
	return err == nil
}

// requireDatabase skips the test if database is not available
// Use this as: if !requireDatabase(t) { return }
func requireDatabase(t interface{}) bool {
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

// setupTestDatabase creates a test database connection and runs migrations
func setupTestDatabase(ctx context.Context) (*Service, error) {
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
		Role("viewer").Permissions("project.read", "team.read", "task.read")

	registry.DefineScope("project").
		Role("project_manager").Permissions("project.*", "team.*", "task.*").CanAssign("team_lead", "developer", "viewer").
		Role("team_lead").Permissions("team.*", "task.*").CanAssign("developer").
		Role("developer").Permissions("team.read", "task.*").
		Role("viewer").Permissions("project.read", "team.read", "task.read")

	registry.DefineScope("team").
		Role("team_lead").Permissions("team.*", "task.*").CanAssign("developer").
		Role("developer").Permissions("team.read", "task.*").
		Role("viewer").Permissions("team.read", "task.read")
}

// NewDBKit creates a new dbkit instance (helper to avoid import issues)
func NewDBKit(databaseURL string) (*dbkit.DBKit, error) {
	return dbkit.New(dbkit.Config{URL: databaseURL})
}
