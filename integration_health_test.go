package rolekit

import (
	"context"
	"testing"
	"time"
)

// TestHealthMonitoringIntegration tests health monitoring with real database
func TestHealthMonitoringIntegration(t *testing.T) {
	if !requireDatabase(t) {
		return
	}

	ctx := context.Background()
	service, err := setupTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	t.Run("Basic health check", func(t *testing.T) {
		// Test basic health check
		health := service.Health(ctx)
		if !health.Healthy {
			t.Errorf("Database should be healthy, got: %+v", health)
		}
	})

	t.Run("IsHealthy check", func(t *testing.T) {
		// Test simple health check
		healthy := service.IsHealthy(ctx)
		if !healthy {
			t.Error("Database should be healthy")
		}
	})

	t.Run("Ping test", func(t *testing.T) {
		// Test database ping
		err := service.Ping(ctx)
		if err != nil {
			t.Errorf("Ping should succeed: %v", err)
		}
	})

	t.Run("Pool statistics", func(t *testing.T) {
		// Test pool statistics
		stats := service.GetPoolStats()
		// Stats should be available but might be zero values
		if stats.MaxOpenConnections == 0 && stats.OpenConnections == 0 {
			// This is expected for non-DBKit instances
			t.Log("Pool stats not available (not a DBKit instance)")
		}
	})
}

// TestConnectionPoolIntegration tests connection pool management with real database
func TestConnectionPoolIntegration(t *testing.T) {
	if !requireDatabase(t) {
		return
	}

	ctx := context.Background()
	service, err := setupTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	t.Run("Get default pool config", func(t *testing.T) {
		// Test getting current pool configuration
		config, err := service.GetConnectionPoolConfig()
		if err != nil {
			t.Errorf("Should be able to get pool config: %v", err)
		} else {
			// Config should have reasonable values
			if config.MaxOpenConnections <= 0 {
				t.Error("MaxOpenConnections should be positive")
			}
			if config.MaxIdleConnections < 0 {
				t.Error("MaxIdleConnections should be non-negative")
			}
		}
	})

	t.Run("Configure connection pool", func(t *testing.T) {
		// Test configuring connection pool
		config := DefaultPoolConfig()
		config.MaxOpenConnections = 10
		config.MaxIdleConnections = 5

		err := service.ConfigureConnectionPool(config)
		if err != nil {
			t.Errorf("Should be able to configure pool: %v", err)
		}

		// Verify the configuration was applied
		appliedConfig, err := service.GetConnectionPoolConfig()
		if err != nil {
			t.Errorf("Should be able to get updated config: %v", err)
		} else if appliedConfig.MaxOpenConnections != 10 {
			t.Errorf("Expected MaxOpenConnections=10, got %d", appliedConfig.MaxOpenConnections)
		}
	})

	t.Run("Reset connection pool", func(t *testing.T) {
		// Test resetting connection pool to defaults
		err := service.ResetConnectionPool()
		if err != nil {
			t.Errorf("Should be able to reset pool: %v", err)
		}
	})

	t.Run("Optimize connection pool", func(t *testing.T) {
		// Test pool optimization
		err := service.OptimizeConnectionPool()
		if err != nil {
			t.Errorf("Should be able to optimize pool: %v", err)
		}
	})
}

// TestMigrationIntegration tests migration system with real database
func TestMigrationIntegration(t *testing.T) {
	if !requireDatabase(t) {
		return
	}

	ctx := context.Background()
	service, err := setupTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	t.Run("Get migrations", func(t *testing.T) {
		// Test getting migration definitions
		migrations := service.Migrations()
		if len(migrations) == 0 {
			t.Error("Should have at least one migration")
		}

		// Verify migration structure
		for _, migration := range migrations {
			if migration.ID == "" {
				t.Error("Migration ID should not be empty")
			}
			if migration.Description == "" {
				t.Error("Migration description should not be empty")
			}
			if migration.SQL == "" {
				t.Error("Migration SQL should not be empty")
			}
		}
	})

	t.Run("Verify tables exist", func(t *testing.T) {
		// Since migrations were run in setupTestDatabase, verify tables exist
		db := service.db

		// Check role_assignments table
		var count int
		err := db.NewSelect().Model((*struct{})(nil)).
			TableExpr("role_assignments").
			ColumnExpr("COUNT(*)").
			Scan(ctx, &count)
		if err != nil {
			t.Errorf("Should be able to query role_assignments table: %v", err)
		}

		// Check role_audit_log table
		err = db.NewSelect().Model((*struct{})(nil)).
			TableExpr("role_audit_log").
			ColumnExpr("COUNT(*)").
			Scan(ctx, &count)
		if err != nil {
			t.Errorf("Should be able to query role_audit_log table: %v", err)
		}

		// Check scope_hierarchy table
		err = db.NewSelect().Model((*struct{})(nil)).
			TableExpr("scope_hierarchy").
			ColumnExpr("COUNT(*)").
			Scan(ctx, &count)
		if err != nil {
			t.Errorf("Should be able to query scope_hierarchy table: %v", err)
		}
	})
}

// TestPerformanceIntegration tests performance-related functionality
func TestPerformanceIntegration(t *testing.T) {
	if !requireDatabase(t) {
		return
	}

	ctx := context.Background()
	service, err := setupTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create test data
	userID := "550e8400-e29b-41d4-a716-446655440000"
	orgID := "550e8400-e29b-41d4-a716-446655440010"

	// Set up admin
	ctx = WithActorID(ctx, userID)
	if err := service.Assign(ctx, userID, "super_admin", "organization", orgID); err != nil {
		t.Fatalf("Failed to assign admin role: %v", err)
	}

	t.Run("Count operations performance", func(t *testing.T) {
		// Test count operations
		start := time.Now()

		count, err := service.CountRoles(ctx, userID, "organization", orgID)
		if err != nil {
			t.Errorf("CountRoles should succeed: %v", err)
		}

		duration := time.Since(start)
		t.Logf("CountRoles took %v", duration)

		if count < 1 {
			t.Error("Should count at least one role")
		}

		// Test count all roles
		start = time.Now()
		total, err := service.CountAllRoles(ctx)
		if err != nil {
			t.Errorf("CountAllRoles should succeed: %v", err)
		}
		duration = time.Since(start)
		t.Logf("CountAllRoles took %v", duration)

		if total < 1 {
			t.Error("Should count at least one role total")
		}
	})

	t.Run("CheckExists performance", func(t *testing.T) {
		// Test CheckExists performance
		start := time.Now()

		exists := service.CheckExists(ctx, userID, "super_admin", "organization", orgID)
		duration := time.Since(start)
		t.Logf("CheckExists took %v", duration)

		if !exists {
			t.Error("Role should exist")
		}

		// Test non-existent role
		start = time.Now()
		exists = service.CheckExists(ctx, userID, "nonexistent", "organization", orgID)
		duration = time.Since(start)
		t.Logf("CheckExists (non-existent) took %v", duration)

		if exists {
			t.Error("Non-existent role should not exist")
		}
	})

	t.Run("Bulk operations performance", func(t *testing.T) {
		// Test bulk assignment performance
		assignments := make([]RoleAssignment, 100)
		for i := 0; i < 100; i++ {
			assignments[i] = RoleAssignment{
				UserID:    "user" + string(rune(i+'0')),
				Role:      "developer",
				ScopeType: "organization",
				ScopeID:   orgID,
			}
		}

		start := time.Now()
		err := service.AssignMultiple(ctx, assignments)
		duration := time.Since(start)
		t.Logf("AssignMultiple (100 assignments) took %v", duration)

		if err != nil {
			t.Errorf("AssignMultiple should succeed: %v", err)
		}
	})
}
