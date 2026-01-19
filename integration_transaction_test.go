package rolekit

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestTransactionSupportIntegration tests transaction functionality with real database
func TestTransactionSupportIntegration(t *testing.T) {
	if !requireDatabase(t) {
		return
	}

	ctx := context.Background()
	service, err := setupTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create test data
	userID := "test-user-" + t.Name() + "-" + fmt.Sprintf("%d", time.Now().UnixNano())
	orgID := "test-org-" + t.Name() + "-" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Set up admin
	ctx = WithActorID(ctx, userID)
	if err := service.Assign(ctx, userID, "super_admin", "organization", orgID); err != nil {
		t.Fatalf("Failed to assign admin role: %v", err)
	}

	t.Run("Transaction commit", func(t *testing.T) {
		// Test successful transaction
		err := service.Transaction(ctx, func(ctx context.Context) error {
			// Assign a role within transaction
			return service.Assign(ctx, "user1", "developer", "organization", orgID)
		})

		if err != nil {
			t.Errorf("Transaction should have succeeded: %v", err)
		}

		// Verify the role was assigned
		if !service.Can(ctx, "user1", "developer", "organization", orgID) {
			t.Error("Role should be assigned after successful transaction")
		}
	})

	t.Run("Transaction rollback", func(t *testing.T) {
		// Test transaction rollback on error
		err := service.Transaction(ctx, func(ctx context.Context) error {
			// Assign a role
			if err := service.Assign(ctx, "user2", "developer", "organization", orgID); err != nil {
				return err
			}

			// Return an error to trigger rollback
			return errors.New("intentional error for rollback test")
		})

		if err == nil {
			t.Error("Transaction should have failed")
		}

		// Verify the role was NOT assigned (rollback worked)
		if service.Can(ctx, "user2", "developer", "organization", orgID) {
			t.Error("Role should not be assigned after failed transaction")
		}
	})

	t.Run("Nested transaction", func(t *testing.T) {
		// Test nested transactions (savepoints)
		err := service.Transaction(ctx, func(ctx context.Context) error {
			// Outer transaction
			if err := service.Assign(ctx, "user3", "developer", "organization", orgID); err != nil {
				return err
			}

			// Inner transaction (should use savepoint)
			return service.Transaction(ctx, func(ctx context.Context) error {
				return service.Assign(ctx, "user4", "developer", "organization", orgID)
			})
		})

		if err != nil {
			t.Errorf("Nested transaction should have succeeded: %v", err)
		}

		// Verify both roles were assigned
		if !service.Can(ctx, "user3", "developer", "organization", orgID) {
			t.Error("User3 should have role after nested transaction")
		}
		if !service.Can(ctx, "user4", "developer", "organization", orgID) {
			t.Error("User4 should have role after nested transaction")
		}
	})

	t.Run("Read-only transaction", func(t *testing.T) {
		// Test read-only transaction
		err := service.ReadOnlyTransaction(ctx, func(ctx context.Context) error {
			// Should be able to read
			roles, err := service.GetUserRoles(ctx, userID)
			if err != nil {
				return err
			}

			if !roles.HasRole("super_admin", "organization", orgID) {
				return errors.New("admin role not found")
			}

			// Should NOT be able to write in read-only transaction
			return service.Assign(ctx, "user5", "developer", "organization", orgID)
		})

		// Read-only transaction should fail on write attempt
		if err == nil {
			t.Error("Read-only transaction should have failed on write attempt")
		}

		// Verify the role was NOT assigned
		if service.Can(ctx, "user5", "developer", "organization", orgID) {
			t.Error("Role should not be assigned after failed read-only transaction")
		}
	})
}

// TestAssignDirectIntegration tests direct assignment without pre-checks
func TestAssignDirectIntegration(t *testing.T) {
	if !requireDatabase(t) {
		return
	}

	ctx := context.Background()
	service, err := setupTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create test data
	userID := "test-user-" + t.Name() + "-" + fmt.Sprintf("%d", time.Now().UnixNano())
	orgID := "test-org-" + t.Name() + "-" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Set up admin
	ctx = WithActorID(ctx, userID)
	if err := service.Assign(ctx, userID, "super_admin", "organization", orgID); err != nil {
		t.Fatalf("Failed to assign admin role: %v", err)
	}

	t.Run("Direct assignment", func(t *testing.T) {
		// Test direct assignment (more performant)
		err := service.AssignDirect(ctx, "user1", "developer", "organization", orgID)
		if err != nil {
			t.Errorf("Direct assignment should have succeeded: %v", err)
		}

		// Verify the role was assigned
		if !service.Can(ctx, "user1", "developer", "organization", orgID) {
			t.Error("Role should be assigned after direct assignment")
		}
	})

	t.Run("Duplicate direct assignment", func(t *testing.T) {
		// Test duplicate assignment (should not error but indicate already exists)
		err := service.AssignDirect(ctx, "user1", "developer", "organization", orgID)
		if err == nil {
			t.Error("Duplicate direct assignment should return an error")
		}

		// Should be a specific error type for already assigned
		if !errors.Is(err, ErrRoleAlreadyAssigned) {
			t.Errorf("Expected role already assigned error, got: %v", err)
		}
	})
}

// TestAssignWithRetryIntegration tests retry logic for transient errors
func TestAssignWithRetryIntegration(t *testing.T) {
	if !requireDatabase(t) {
		return
	}

	ctx := context.Background()
	service, err := setupTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create test data
	userID := "test-user-" + t.Name() + "-" + fmt.Sprintf("%d", time.Now().UnixNano())
	orgID := "test-org-" + t.Name() + "-" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Set up admin
	ctx = WithActorID(ctx, userID)
	if err := service.Assign(ctx, userID, "super_admin", "organization", orgID); err != nil {
		t.Fatalf("Failed to assign admin role: %v", err)
	}

	t.Run("Retry on transient error", func(t *testing.T) {
		// This test would need to simulate transient errors
		// For now, just test that the method exists and works for successful case
		err := service.AssignWithRetry(ctx, "user2", "developer", "organization", orgID)
		if err != nil {
			t.Errorf("AssignWithRetry should have succeeded: %v", err)
		}

		// Verify the role was assigned
		if !service.Can(ctx, "user2", "developer", "organization", orgID) {
			t.Error("Role should be assigned after retry")
		}
	})
}

// TestTransactionMetricsIntegration tests transaction monitoring
func TestTransactionMetricsIntegration(t *testing.T) {
	if !requireDatabase(t) {
		return
	}

	ctx := context.Background()
	service, err := setupTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create test data
	userID := "test-user-" + t.Name() + "-" + fmt.Sprintf("%d", time.Now().UnixNano())
	orgID := "test-org-" + t.Name() + "-" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Set up admin
	ctx = WithActorID(ctx, userID)
	if err := service.Assign(ctx, userID, "super_admin", "organization", orgID); err != nil {
		t.Fatalf("Failed to assign admin role: %v", err)
	}

	// Reset metrics to start fresh
	service.ResetTransactionMetrics()

	// Perform some transactions
	for i := 0; i < 5; i++ {
		err := service.Transaction(ctx, func(ctx context.Context) error {
			return service.Assign(ctx, "user"+string(rune(i+'0')), "developer", "organization", orgID)
		})
		if err != nil {
			t.Errorf("Transaction %d failed: %v", i, err)
		}
	}

	// Check metrics
	metrics := service.GetTransactionMetrics()
	if metrics.TotalTransactions != 5 {
		t.Errorf("Expected 5 total transactions, got %d", metrics.TotalTransactions)
	}

	if metrics.FailedTransactions != 0 {
		t.Errorf("Expected 0 failed transactions, got %d", metrics.FailedTransactions)
	}

	// Test health check
	if !service.IsTransactionHealthy() {
		t.Error("Transaction system should be healthy")
	}
}
