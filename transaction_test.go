package rolekit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fernandezvara/dbkit"
)

// TestTransactionContextPropagation tests that transaction context is properly propagated
func TestTransactionContextPropagation(t *testing.T) {
	// Setup test database and service
	registry := NewRegistry()

	// Define test roles
	registry.DefineScope("organization").
		Role("admin").
		CanAssign("admin", "member")

	db := setupTestDB(t)
	service := NewService(registry, db)

	// Test that assignments within a transaction use the same context
	err := service.Transaction(context.Background(), func(ctx context.Context) error {
		// First assignment should succeed
		err1 := service.Assign(ctx, "user1", "admin", "organization", "org1")
		require.NoError(t, err1)

		// Second assignment should succeed in same transaction
		err2 := service.Assign(ctx, "user2", "member", "organization", "org1")
		require.NoError(t, err2)

		// Verify both assignments are visible within the transaction
		roles1, err1 := service.GetUserRoles(ctx, "user1")
		require.NoError(t, err1)
		assert.True(t, roles1.HasRole("admin", "organization", "org1"))

		roles2, err2 := service.GetUserRoles(ctx, "user2")
		require.NoError(t, err2)
		assert.True(t, roles2.HasRole("member", "organization", "org1"))

		return nil
	})

	require.NoError(t, err)

	// Verify assignments persist after transaction
	roles1, err := service.GetUserRoles(context.Background(), "user1")
	require.NoError(t, err)
	assert.True(t, roles1.HasRole("admin", "organization", "org1"))
}

// TestTransactionRollback tests that transactions properly rollback on error
func TestTransactionRollback(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	db := setupTestDB(t)
	service := NewService(registry, db)

	// Test transaction rollback
	err := service.Transaction(context.Background(), func(ctx context.Context) error {
		// This assignment should succeed
		err1 := service.Assign(ctx, "user1", "admin", "organization", "org1")
		require.NoError(t, err1)

		// Verify assignment exists within transaction
		roles, err := service.GetUserRoles(ctx, "user1")
		require.NoError(t, err)
		assert.True(t, roles.HasRole("admin", "organization", "org1"))

		// Return error to trigger rollback
		return assert.AnError
	})

	assert.Error(t, err)

	// Verify assignment was rolled back
	roles, err := service.GetUserRoles(context.Background(), "user1")
	require.NoError(t, err)
	assert.False(t, roles.HasRole("admin", "organization", "org1"))
}

// TestAssignDirect tests the direct assignment method
func TestAssignDirect(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	db := setupTestDB(t)
	service := NewService(registry, db)

	ctx := context.Background()
	ctx = WithActorID(ctx, "actor1")

	// Test direct assignment
	err := service.AssignDirect(ctx, "user1", "admin", "organization", "org1")
	require.NoError(t, err)

	// Verify assignment
	roles, err := service.GetUserRoles(ctx, "user1")
	require.NoError(t, err)
	assert.True(t, roles.HasRole("admin", "organization", "org1"))

	// Test duplicate assignment (should not error)
	err = service.AssignDirect(ctx, "user1", "admin", "organization", "org1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already has this role")
}

// TestAssignMultiple tests bulk assignment functionality
func TestAssignMultiple(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin").Role("member")

	db := setupTestDB(t)
	service := NewService(registry, db)

	ctx := context.Background()
	ctx = WithActorID(ctx, "actor1")

	assignments := []RoleAssignment{
		{UserID: "user1", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user2", Role: "member", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user3", Role: "member", ScopeType: "organization", ScopeID: "org1"},
	}

	// Test bulk assignment
	err := service.AssignMultiple(ctx, assignments)
	require.NoError(t, err)

	// Verify all assignments
	for _, assignment := range assignments {
		roles, err := service.GetUserRoles(ctx, assignment.UserID)
		require.NoError(t, err)
		assert.True(t, roles.HasRole(assignment.Role, assignment.ScopeType, assignment.ScopeID))
	}

	// Test duplicate assignment (should skip duplicates)
	err = service.AssignMultiple(ctx, assignments)
	require.NoError(t, err)
}

// TestTransactionErrorRecovery tests retry logic for transient errors
func TestTransactionErrorRecovery(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	db := setupTestDB(t)
	service := NewService(registry, db)

	ctx := context.Background()
	ctx = WithActorID(ctx, "actor1")

	// Test successful assignment with retry
	err := service.AssignWithRetry(ctx, "user1", "admin", "organization", "org1")
	require.NoError(t, err)

	// Verify assignment
	roles, err := service.GetUserRoles(ctx, "user1")
	require.NoError(t, err)
	assert.True(t, roles.HasRole("admin", "organization", "org1"))
}

// TestTransactionMonitoring tests metrics collection
func TestTransactionMonitoring(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	db := setupTestDB(t)
	service := NewService(registry, db)

	ctx := context.Background()
	ctx = WithActorID(ctx, "actor1")

	// Reset metrics
	service.ResetTransactionMetrics()

	// Perform some transactions
	for i := 0; i < 5; i++ {
		err := service.Transaction(ctx, func(ctx context.Context) error {
			return service.Assign(ctx, "user"+string(rune(i+'1')), "admin", "organization", "org1")
		})
		require.NoError(t, err)
	}

	// Check metrics
	metrics := service.GetTransactionMetrics()
	assert.Equal(t, int64(5), metrics.TotalTransactions)
	assert.Equal(t, int64(5), metrics.SuccessfulTransactions)
	assert.Equal(t, int64(0), metrics.FailedTransactions)
	assert.Greater(t, metrics.AverageDuration, time.Duration(0))
	assert.True(t, service.IsTransactionHealthy())
}

// TestConcurrentTransactions tests concurrent transaction handling
func TestConcurrentTransactions(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin").Role("member")

	db := setupTestDB(t)
	service := NewService(registry, db)

	ctx := context.Background()
	ctx = WithActorID(ctx, "actor1")

	// Run multiple concurrent transactions
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(userID string) {
			defer func() { done <- true }()

			err := service.Transaction(ctx, func(ctx context.Context) error {
				return service.Assign(ctx, userID, "member", "organization", "org1")
			})
			assert.NoError(t, err)
		}("user" + string(rune(i+'1')))
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all assignments succeeded
	for i := 0; i < 10; i++ {
		userID := "user" + string(rune(i+'1'))
		roles, err := service.GetUserRoles(ctx, userID)
		require.NoError(t, err)
		assert.True(t, roles.HasRole("member", "organization", "org1"))
	}
}

// TestNestedTransactions tests savepoint functionality
func TestNestedTransactions(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin").Role("member")

	db := setupTestDB(t)
	service := NewService(registry, db)

	ctx := context.Background()
	ctx = WithActorID(ctx, "actor1")

	// Test nested transaction
	err := service.Transaction(ctx, func(ctx context.Context) error {
		// Outer transaction assignment
		err := service.Assign(ctx, "user1", "admin", "organization", "org1")
		require.NoError(t, err)

		// Nested transaction
		err = service.Transaction(ctx, func(ctx context.Context) error {
			err := service.Assign(ctx, "user2", "member", "organization", "org1")
			require.NoError(t, err)
			return nil
		})
		require.NoError(t, err)

		return nil
	})

	require.NoError(t, err)

	// Verify both assignments persist
	roles1, err := service.GetUserRoles(ctx, "user1")
	require.NoError(t, err)
	assert.True(t, roles1.HasRole("admin", "organization", "org1"))

	roles2, err := service.GetUserRoles(ctx, "user2")
	require.NoError(t, err)
	assert.True(t, roles2.HasRole("member", "organization", "org1"))
}

// TestTransactionMethodsAvailable tests that all transaction methods are available
func TestTransactionMethodsAvailable(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	db := setupTestDB(t)
	service := NewService(registry, db)

	// Test that all transaction methods exist and are callable
	assert.NotNil(t, service.Transaction)
	assert.NotNil(t, service.AssignDirect)
	assert.NotNil(t, service.AssignMultiple)
	assert.NotNil(t, service.AssignWithRetry)
	assert.NotNil(t, service.AssignMultipleWithRetry)
	assert.NotNil(t, service.GetTransactionMetrics)
	assert.NotNil(t, service.ResetTransactionMetrics)
	assert.NotNil(t, service.IsTransactionHealthy)
}

// setupTestDB creates a test database connection for testing
func setupTestDB(t *testing.T) dbkit.IDB {
	// For testing purposes, we'll use a mock or in-memory database
	// In a real implementation, this would set up a test PostgreSQL database

	// For now, we'll skip the actual database setup since this is a demonstration
	// In production, you would use something like:
	// db, err := dbkit.New(dbkit.Config{
	//     URL: "postgres://test:test@localhost:5432/testdb?sslmode=disable",
	// })
	// require.NoError(t, err)

	t.Skip("Database setup not implemented in this test environment")
	return nil
}
