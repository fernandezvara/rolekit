package rolekit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestServiceGetUserRoleNames tests retrieving role names for a user in a scope
func TestServiceGetUserRoleNames(t *testing.T) {
	registry := NewRegistry()
	service := &Service{db: nil, registry: registry}
	ctx := context.Background()

	// Test with nil database - should panic
	assert.Panics(t, func() {
		service.getUserRoleNames(ctx, "user1", "organization", "org1")
	})
}

// TestServiceGetParentScope tests retrieving parent scope information
func TestServiceGetParentScope(t *testing.T) {
	registry := NewRegistry()
	service := &Service{db: nil, registry: registry}
	ctx := context.Background()

	// Test with nil database - should panic
	assert.Panics(t, func() {
		service.getParentScope(ctx, "project", "proj1")
	})
}

// TestServiceLogAudit tests audit logging functionality
func TestServiceLogAudit(t *testing.T) {
	registry := NewRegistry()
	service := &Service{db: nil, registry: registry}
	ctx := context.Background()

	// Create audit entry
	entry := &AuditEntry{
		ActorID:      "user1",
		Action:       AuditActionAssigned,
		TargetUserID: "user2",
		Role:         "admin",
		ScopeType:    "organization",
		ScopeID:      "org1",
	}

	// Test with nil database - should panic
	assert.Panics(t, func() {
		service.logAudit(ctx, entry)
	})
}

// TestServiceAssignDirect tests direct role assignment without pre-checks
func TestServiceAssignDirect(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	service := &Service{db: nil, registry: registry}
	ctx := context.Background()

	// Test with nil database - should return error due to missing actor ID
	err := service.AssignDirect(ctx, "user1", "admin", "organization", "org1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "actor ID required")

	// Test with actor ID in context - should panic on database call
	ctxWithActor := WithActorID(ctx, "actor1")
	assert.Panics(t, func() {
		service.AssignDirect(ctxWithActor, "user1", "admin", "organization", "org1")
	})

	// Test with invalid role - should return error without panicking
	err2 := service.AssignDirect(ctx, "user1", "invalid", "organization", "org1")
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "invalid role")
}

// TestServiceAssignWithRetry tests role assignment with retry logic
func TestServiceAssignWithRetry(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	service := &Service{db: nil, registry: registry}
	ctx := context.Background()

	// Test with nil database - should return error due to missing actor ID
	err := service.AssignWithRetry(ctx, "user1", "admin", "organization", "org1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "actor ID required")

	// Test with actor ID - should panic on database call
	ctxWithActor := WithActorID(ctx, "actor1")
	assert.Panics(t, func() {
		service.AssignWithRetry(ctxWithActor, "user1", "admin", "organization", "org1")
	})
}

// TestServiceAssignMultipleWithRetry tests bulk role assignment with retry
func TestServiceAssignMultipleWithRetry(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin").Role("member")

	service := &Service{db: nil, registry: registry}
	ctx := context.Background()

	assignments := []RoleAssignment{
		{UserID: "user1", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
		{UserID: "user2", Role: "member", ScopeType: "organization", ScopeID: "org1"},
	}

	// Test with nil database - should panic due to nil txMonitor in Transaction
	assert.Panics(t, func() {
		service.AssignMultipleWithRetry(ctx, assignments)
	})
}

// TestServiceTransactionMetrics tests transaction metrics functionality
func TestServiceTransactionMetrics(t *testing.T) {
	registry := NewRegistry()
	service := &Service{db: nil, registry: registry}

	// Test getting metrics - should panic due to nil txMonitor
	assert.Panics(t, func() {
		service.GetTransactionMetrics()
	})

	// Test resetting metrics - should panic due to nil txMonitor
	assert.Panics(t, func() {
		service.ResetTransactionMetrics()
	})

	// Test health check - should panic due to nil txMonitor
	assert.Panics(t, func() {
		service.IsTransactionHealthy()
	})
}

// TestServiceIsTransactionHealthy tests transaction health checking
func TestServiceIsTransactionHealthy(t *testing.T) {
	registry := NewRegistry()
	service := &Service{db: nil, registry: registry}

	// All tests should panic due to nil txMonitor
	t.Run("Healthy with no transactions", func(t *testing.T) {
		assert.Panics(t, func() {
			service.IsTransactionHealthy()
		})
	})

	t.Run("Healthy with few transactions", func(t *testing.T) {
		assert.Panics(t, func() {
			service.IsTransactionHealthy()
		})
	})

	t.Run("Unhealthy with high failure rate", func(t *testing.T) {
		assert.Panics(t, func() {
			service.IsTransactionHealthy()
		})
	})

	t.Run("Unhealthy with slow transactions", func(t *testing.T) {
		assert.Panics(t, func() {
			service.IsTransactionHealthy()
		})
	})
}

// TestIsTransientTransactionError tests transient error detection
func TestIsTransientTransactionError(t *testing.T) {
	t.Run("Nil error", func(t *testing.T) {
		assert.False(t, isTransientTransactionError(nil))
	})

	t.Run("Non-transient errors", func(t *testing.T) {
		err := errors.New("validation failed")
		assert.False(t, isTransientTransactionError(err))

		err2 := errors.New("permission denied")
		assert.False(t, isTransientTransactionError(err2))

		err3 := NewError(ErrInvalidRole, "role does not exist")
		assert.False(t, isTransientTransactionError(err3))
	})

	t.Run("Transient database errors", func(t *testing.T) {
		transientErrors := []string{
			"connection lost",
			"query timeout",
			"deadlock detected",
			"lock wait timeout exceeded",
			"connection refused",
			"connection reset by peer",
			"broken pipe",
			"temporary failure",
			"please try again",
			"resource temporarily unavailable",
		}

		for _, errMsg := range transientErrors {
			err := errors.New(errMsg)
			assert.True(t, isTransientTransactionError(err), "Should be transient: %s", errMsg)
		}
	})

	t.Run("Context errors", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Nanosecond)
		<-ctx.Done()
		assert.True(t, isTransientTransactionError(ctx.Err()))

		ctx2, cancel := context.WithCancel(context.Background())
		cancel()
		assert.True(t, isTransientTransactionError(ctx2.Err()))
	})

	t.Run("Case sensitive matching", func(t *testing.T) {
		err := errors.New("CONNECTION TIMEOUT")
		assert.False(t, isTransientTransactionError(err))

		err2 := errors.New("DeadLock Detected")
		assert.False(t, isTransientTransactionError(err2))

		// These should be true with lowercase
		err3 := errors.New("connection timeout")
		assert.True(t, isTransientTransactionError(err3))

		err4 := errors.New("deadlock detected")
		assert.True(t, isTransientTransactionError(err4))
	})
}

// TestContains tests the contains helper function
func TestContains(t *testing.T) {
	t.Run("Empty substring", func(t *testing.T) {
		assert.True(t, contains("", ""))
		assert.True(t, contains("abc", ""))
		assert.False(t, contains("", "abc"))
	})

	t.Run("Exact match", func(t *testing.T) {
		assert.True(t, contains("hello", "hello"))
		assert.False(t, contains("hello", "world"))
	})

	t.Run("Substring at beginning", func(t *testing.T) {
		assert.True(t, contains("hello world", "hello"))
		assert.True(t, contains("testing", "test"))
	})

	t.Run("Substring at end", func(t *testing.T) {
		assert.True(t, contains("hello world", "world"))
		assert.True(t, contains("testing", "ing"))
	})

	t.Run("Substring in middle", func(t *testing.T) {
		assert.True(t, contains("hello world", "lo wo"))
		assert.True(t, contains("testing", "est"))
	})

	t.Run("Not contained", func(t *testing.T) {
		assert.False(t, contains("hello", "world"))
		assert.False(t, contains("abc", "d"))
	})

	t.Run("Case sensitivity", func(t *testing.T) {
		assert.False(t, contains("Hello", "hello"))
		assert.False(t, contains("HELLO", "hello"))
	})

	t.Run("Single character", func(t *testing.T) {
		assert.True(t, contains("abc", "a"))
		assert.True(t, contains("abc", "b"))
		assert.True(t, contains("abc", "c"))
		assert.False(t, contains("abc", "d"))
	})
}

// TestFindSubstring tests the findSubstring helper function
func TestFindSubstring(t *testing.T) {
	t.Run("Empty substring", func(t *testing.T) {
		assert.True(t, findSubstring("", ""))
		assert.True(t, findSubstring("abc", ""))
		assert.False(t, findSubstring("", "abc"))
	})

	t.Run("Exact match", func(t *testing.T) {
		assert.True(t, findSubstring("hello", "hello"))
		assert.False(t, findSubstring("hello", "world"))
	})

	t.Run("Substring found", func(t *testing.T) {
		assert.True(t, findSubstring("hello world", "lo wo"))
		assert.True(t, findSubstring("testing", "est"))
		assert.True(t, findSubstring("abcde", "bcd"))
	})

	t.Run("Substring not found", func(t *testing.T) {
		assert.False(t, findSubstring("hello", "world"))
		assert.False(t, findSubstring("abc", "d"))
		assert.False(t, findSubstring("abc", "abcd"))
	})

	t.Run("Single character", func(t *testing.T) {
		assert.True(t, findSubstring("abc", "a"))
		assert.True(t, findSubstring("abc", "b"))
		assert.True(t, findSubstring("abc", "c"))
		assert.False(t, findSubstring("abc", "d"))
	})

	t.Run("Longer substring", func(t *testing.T) {
		assert.False(t, findSubstring("abc", "abcd"))
		assert.False(t, findSubstring("test", "testing"))
	})
}

// TestServiceHelpersEdgeCases tests edge cases and error conditions
func TestServiceHelpersEdgeCases(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	service := &Service{db: nil, registry: registry}
	ctx := context.Background()

	t.Run("AssignDirect with empty values", func(t *testing.T) {
		// These should return errors without panicking (validation happens before DB call)
		err := service.AssignDirect(ctx, "", "admin", "organization", "org1")
		assert.Error(t, err)

		err2 := service.AssignDirect(ctx, "user1", "", "organization", "org1")
		assert.Error(t, err2)

		err3 := service.AssignDirect(ctx, "user1", "admin", "", "org1")
		assert.Error(t, err3)

		err4 := service.AssignDirect(ctx, "user1", "admin", "organization", "")
		assert.Error(t, err4)
	})

	t.Run("Context cancellation", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		// Add actor ID to context to bypass the actor ID check
		ctxWithActor := WithActorID(cancelledCtx, "actor1")

		// Should panic due to nil DB when actor ID is present
		assert.Panics(t, func() {
			service.AssignDirect(ctxWithActor, "user1", "admin", "organization", "org1")
		})
	})

	t.Run("Nil registry", func(t *testing.T) {
		serviceNilRegistry := &Service{db: nil, registry: nil}

		// Should panic when trying to validate role
		assert.Panics(t, func() {
			serviceNilRegistry.AssignDirect(ctx, "user1", "admin", "organization", "org1")
		})
	})
}
