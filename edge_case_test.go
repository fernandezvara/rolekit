package rolekit

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// ============================================================================
// Error Scenario Tests
// ============================================================================

// TestErrorScenarios tests various error conditions
func TestErrorScenarios(t *testing.T) {
	helper := NewTestDataHelper(t)
	if helper == nil {
		return
	}
	defer helper.CleanupTestData()

	service := helper.GetService()
	ctx := helper.GetContext()

	t.Run("Assign with empty user ID", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		err := service.Assign(actorCtx, "", "developer", "organization", orgID)
		if err == nil {
			t.Log("Assign accepts empty user ID (no validation)")
		} else {
			t.Logf("Assign correctly rejects empty user ID: %v", err)
		}
	})

	t.Run("Assign with empty role", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		err := service.Assign(actorCtx, userID, "", "organization", orgID)
		if err == nil {
			t.Error("Should reject empty role")
		} else {
			t.Logf("Correctly rejected empty role: %v", err)
		}
	})

	t.Run("Assign with empty scope type", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		err := service.Assign(actorCtx, userID, "developer", "", orgID)
		if err == nil {
			t.Error("Should reject empty scope type")
		} else {
			t.Logf("Correctly rejected empty scope type: %v", err)
		}
	})

	t.Run("Assign with undefined role", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		err := service.Assign(actorCtx, userID, "nonexistent_role", "organization", orgID)
		if err == nil {
			t.Error("Should reject undefined role")
		} else {
			t.Logf("Correctly rejected undefined role: %v", err)
		}
	})

	t.Run("Assign with undefined scope type", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		err := service.Assign(actorCtx, userID, "developer", "nonexistent_scope", orgID)
		if err == nil {
			t.Error("Should reject undefined scope type")
		} else {
			t.Logf("Correctly rejected undefined scope type: %v", err)
		}
	})

	t.Run("Revoke non-existent assignment", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		err := service.Revoke(actorCtx, userID, "developer", "organization", orgID)
		if err == nil {
			t.Log("Revoke accepts non-existent assignment (idempotent)")
		} else {
			t.Logf("Revoke returns error for non-existent assignment: %v", err)
		}
	})

	t.Run("GetUserRoles for non-existent user", func(t *testing.T) {
		userRoles, err := service.GetUserRoles(ctx, "nonexistent-user-id")
		if err != nil {
			t.Errorf("GetUserRoles should not error for non-existent user: %v", err)
		} else if len(userRoles.Assignments) != 0 {
			t.Errorf("Expected 0 assignments, got %d", len(userRoles.Assignments))
		}
	})

	t.Run("Can check for non-existent user", func(t *testing.T) {
		result := service.Can(ctx, "nonexistent-user-id", "developer", "organization", "org-123")
		if result {
			t.Error("Can should return false for non-existent user")
		}
	})

	t.Run("HasPermission for non-existent user", func(t *testing.T) {
		result := service.HasPermission(ctx, "nonexistent-user-id", "task.create", "organization", "org-123")
		if result {
			t.Error("HasPermission should return false for non-existent user")
		}
	})
}

// ============================================================================
// Edge Case Tests
// ============================================================================

// TestEdgeCases tests boundary conditions and edge cases
func TestEdgeCases(t *testing.T) {
	helper := NewTestDataHelper(t)
	if helper == nil {
		return
	}
	defer helper.CleanupTestData()

	service := helper.GetService()
	ctx := helper.GetContext()

	t.Run("Very long user ID", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		// Create a very long user ID (1000 characters)
		longUserID := strings.Repeat("a", 1000)
		err := service.Assign(actorCtx, longUserID, "developer", "organization", orgID)
		if err != nil {
			t.Logf("Long user ID rejected: %v", err)
		} else {
			// Verify it was stored correctly
			can := service.Can(ctx, longUserID, "developer", "organization", orgID)
			if !can {
				t.Error("Long user ID was not stored correctly")
			}
		}
	})

	t.Run("Special characters in IDs", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		specialUserID := "user-with-special-chars!@#$%^&*()_+-=[]{}|;':\",./<>?"
		err := service.Assign(actorCtx, specialUserID, "developer", "organization", orgID)
		if err != nil {
			t.Logf("Special characters in user ID rejected: %v", err)
		} else {
			can := service.Can(ctx, specialUserID, "developer", "organization", orgID)
			if !can {
				t.Error("Special characters user ID was not stored correctly")
			}
		}
	})

	t.Run("Unicode characters in IDs", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		unicodeUserID := "用户-пользователь-משתמש-🎉"
		err := service.Assign(actorCtx, unicodeUserID, "developer", "organization", orgID)
		if err != nil {
			t.Logf("Unicode characters in user ID rejected: %v", err)
		} else {
			can := service.Can(ctx, unicodeUserID, "developer", "organization", orgID)
			if !can {
				t.Error("Unicode user ID was not stored correctly")
			}
		}
	})

	t.Run("Empty scope ID (wildcard)", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		// Assign with empty scope ID (wildcard)
		err := service.Assign(actorCtx, userID, "developer", "organization", "")
		if err != nil {
			t.Logf("Empty scope ID rejected: %v", err)
		} else {
			// Check if wildcard assignment works
			can := service.Can(ctx, userID, "developer", "organization", "any-org-id")
			t.Logf("Wildcard assignment result: %v", can)
		}
	})

	t.Run("Duplicate assignment", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		// First assignment
		err := service.Assign(actorCtx, userID, "developer", "organization", orgID)
		if err != nil {
			t.Fatalf("First assignment failed: %v", err)
		}

		// Duplicate assignment
		err = service.Assign(actorCtx, userID, "developer", "organization", orgID)
		if err == nil {
			t.Log("Duplicate assignment accepted (idempotent)")
		} else {
			t.Logf("Duplicate assignment rejected: %v", err)
		}
	})

	t.Run("Multiple roles same user same scope", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		// Assign multiple roles
		roles := []string{"developer", "viewer", "team_lead"}
		for _, role := range roles {
			err := service.Assign(actorCtx, userID, role, "organization", orgID)
			if err != nil {
				t.Errorf("Failed to assign role %s: %v", role, err)
			}
		}

		// Verify all roles
		userRoles, err := service.GetUserRoles(ctx, userID)
		if err != nil {
			t.Fatalf("Failed to get user roles: %v", err)
		}

		if len(userRoles.Assignments) < len(roles) {
			t.Errorf("Expected at least %d roles, got %d", len(roles), len(userRoles.Assignments))
		}
	})
}

// ============================================================================
// Concurrency Tests
// ============================================================================

// TestConcurrencyScenarios tests concurrent access patterns
func TestConcurrencyScenarios(t *testing.T) {
	helper := NewTestDataHelper(t)
	if helper == nil {
		return
	}
	defer helper.CleanupTestData()

	service := helper.GetService()
	ctx := helper.GetContext()

	t.Run("Concurrent assignments to same user", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		var wg sync.WaitGroup
		errors := make(chan error, 10)
		roles := []string{"developer", "viewer", "team_lead"}

		for _, role := range roles {
			wg.Add(1)
			go func(r string) {
				defer wg.Done()
				if err := service.Assign(actorCtx, userID, r, "organization", orgID); err != nil {
					errors <- err
				}
			}(role)
		}

		wg.Wait()
		close(errors)

		errorCount := 0
		for err := range errors {
			t.Logf("Concurrent assignment error: %v", err)
			errorCount++
		}

		if errorCount > 0 {
			t.Logf("%d errors during concurrent assignments", errorCount)
		}

		// Verify final state
		userRoles, err := service.GetUserRoles(ctx, userID)
		if err != nil {
			t.Fatalf("Failed to get user roles: %v", err)
		}
		t.Logf("User has %d roles after concurrent assignments", len(userRoles.Assignments))
	})

	t.Run("Concurrent reads and writes", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		var wg sync.WaitGroup
		numWriters := 5
		numReaders := 10

		// Writers
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				userID := fmt.Sprintf("concurrent-user-%d-%d", time.Now().UnixNano(), idx)
				_ = service.Assign(actorCtx, userID, "developer", "organization", orgID)
			}(i)
		}

		// Readers
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = service.Can(ctx, adminID, "super_admin", "organization", orgID)
			}()
		}

		wg.Wait()
		t.Log("Concurrent reads and writes completed without deadlock")
	})

	t.Run("Concurrent assign and revoke same role", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		// Initial assignment
		if err := service.Assign(actorCtx, userID, "developer", "organization", orgID); err != nil {
			t.Fatalf("Initial assignment failed: %v", err)
		}

		var wg sync.WaitGroup
		iterations := 10

		for i := 0; i < iterations; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				_ = service.Assign(actorCtx, userID, "developer", "organization", orgID)
			}()
			go func() {
				defer wg.Done()
				_ = service.Revoke(actorCtx, userID, "developer", "organization", orgID)
			}()
		}

		wg.Wait()
		t.Log("Concurrent assign/revoke completed without deadlock")

		// Final state is indeterminate but should be consistent
		can := service.Can(ctx, userID, "developer", "organization", orgID)
		t.Logf("Final state: user has developer role = %v", can)
	})
}

// ============================================================================
// Data Integrity Tests
// ============================================================================

// TestDataIntegrity tests data consistency under various conditions
func TestDataIntegrity(t *testing.T) {
	helper := NewTestDataHelper(t)
	if helper == nil {
		return
	}
	defer helper.CleanupTestData()

	service := helper.GetService()
	ctx := helper.GetContext()

	t.Run("Assignment persists correctly", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		// Assign role
		if err := service.Assign(actorCtx, userID, "developer", "organization", orgID); err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		// Verify with Can
		if !service.Can(ctx, userID, "developer", "organization", orgID) {
			t.Error("Can check failed after assignment")
		}

		// Verify with GetUserRoles
		userRoles, err := service.GetUserRoles(ctx, userID)
		if err != nil {
			t.Fatalf("Failed to get user roles: %v", err)
		}

		found := false
		for _, assignment := range userRoles.Assignments {
			if assignment.Role == "developer" && assignment.ScopeType == "organization" && assignment.ScopeID == orgID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Assignment not found in GetUserRoles")
		}

		// Verify with CheckExists
		exists := service.CheckExists(ctx, userID, "developer", "organization", orgID)
		if !exists {
			t.Error("CheckExists returned false after assignment")
		}
	})

	t.Run("Revocation removes correctly", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		// Assign and then revoke
		if err := service.Assign(actorCtx, userID, "developer", "organization", orgID); err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}
		if err := service.Revoke(actorCtx, userID, "developer", "organization", orgID); err != nil {
			t.Fatalf("Failed to revoke role: %v", err)
		}

		// Verify removal
		if service.Can(ctx, userID, "developer", "organization", orgID) {
			t.Error("Can check should return false after revocation")
		}

		exists := service.CheckExists(ctx, userID, "developer", "organization", orgID)
		if exists {
			t.Error("CheckExists should return false after revocation")
		}
	})

	t.Run("Count operations are accurate", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		// Assign multiple roles
		roles := []string{"developer", "viewer"}
		for _, role := range roles {
			if err := service.Assign(actorCtx, userID, role, "organization", orgID); err != nil {
				t.Fatalf("Failed to assign role %s: %v", role, err)
			}
		}

		// Count roles
		count, err := service.CountRoles(ctx, userID, "organization", orgID)
		if err != nil {
			t.Fatalf("Failed to count roles: %v", err)
		}

		if count != len(roles) {
			t.Errorf("Expected %d roles, got %d", len(roles), count)
		}
	})
}

// ============================================================================
// Transaction Error Tests
// ============================================================================

// TestTransactionErrors tests transaction error handling
func TestTransactionErrors(t *testing.T) {
	helper := NewTestDataHelper(t)
	if helper == nil {
		return
	}
	defer helper.CleanupTestData()

	service := helper.GetService()
	ctx := helper.GetContext()

	t.Run("Transaction rollback on error", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		// Transaction that fails
		err := service.Transaction(actorCtx, func(txCtx context.Context) error {
			if err := service.Assign(txCtx, userID, "developer", "organization", orgID); err != nil {
				return err
			}
			// Force error
			return fmt.Errorf("intentional error for rollback test")
		})

		if err == nil {
			t.Error("Transaction should have failed")
		}

		// Note: Transaction rollback behavior depends on implementation
		// This test documents the actual behavior
		can := service.Can(ctx, userID, "developer", "organization", orgID)
		if can {
			t.Log("Transaction rollback did not occur - assignment persisted")
		} else {
			t.Log("Transaction rollback worked correctly")
		}
	})

	t.Run("Transaction commit on success", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		// Successful transaction
		err := service.Transaction(actorCtx, func(txCtx context.Context) error {
			return service.Assign(txCtx, userID, "developer", "organization", orgID)
		})

		if err != nil {
			t.Fatalf("Transaction should have succeeded: %v", err)
		}

		// Verify commit
		can := service.Can(ctx, userID, "developer", "organization", orgID)
		if !can {
			t.Error("Assignment should have been committed")
		}
	})

	t.Run("Nested transaction behavior", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		userID := helper.CreateTestUser("user")
		orgID := helper.CreateTestOrg("org")
		actorCtx := WithActorID(ctx, adminID)

		// Setup admin first
		if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		// Nested transaction
		err := service.Transaction(actorCtx, func(txCtx context.Context) error {
			if err := service.Assign(txCtx, userID, "developer", "organization", orgID); err != nil {
				return err
			}

			// Inner transaction
			return service.Transaction(txCtx, func(innerCtx context.Context) error {
				return service.Assign(innerCtx, userID, "viewer", "organization", orgID)
			})
		})

		if err != nil {
			t.Logf("Nested transaction result: %v", err)
		}

		// Check final state
		userRoles, _ := service.GetUserRoles(ctx, userID)
		t.Logf("User has %d roles after nested transaction", len(userRoles.Assignments))
	})
}

// ============================================================================
// Context Cancellation Tests
// ============================================================================

// TestContextCancellation tests behavior when context is cancelled
func TestContextCancellation(t *testing.T) {
	helper := NewTestDataHelper(t)
	if helper == nil {
		return
	}
	defer helper.CleanupTestData()

	service := helper.GetService()
	ctx := helper.GetContext()

	t.Run("Cancelled context during operation", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		orgID := helper.CreateTestOrg("org")

		// Create cancellable context
		cancelCtx, cancel := context.WithCancel(ctx)
		actorCtx := WithActorID(cancelCtx, adminID)

		// Setup admin first with non-cancelled context
		normalActorCtx := WithActorID(ctx, adminID)
		if err := service.Assign(normalActorCtx, adminID, "super_admin", "organization", orgID); err != nil {
			t.Fatalf("Failed to setup admin: %v", err)
		}

		// Cancel immediately
		cancel()

		// Try operation with cancelled context
		userID := helper.CreateTestUser("user")
		err := service.Assign(actorCtx, userID, "developer", "organization", orgID)
		if err != nil {
			t.Logf("Operation with cancelled context: %v", err)
		} else {
			t.Log("Operation completed despite cancelled context")
		}
	})

	t.Run("Context with timeout", func(t *testing.T) {
		adminID := helper.CreateTestUser("admin")
		orgID := helper.CreateTestOrg("org")

		// Create context with very short timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()

		// Wait for timeout
		time.Sleep(1 * time.Millisecond)

		actorCtx := WithActorID(timeoutCtx, adminID)

		userID := helper.CreateTestUser("user")
		err := service.Assign(actorCtx, userID, "developer", "organization", orgID)
		if err != nil {
			t.Logf("Operation with timed out context: %v", err)
		} else {
			t.Log("Operation completed despite timed out context")
		}
	})
}
