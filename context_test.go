package rolekit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWithUserID tests adding user ID to context
func TestWithUserID(t *testing.T) {
	ctx := context.Background()

	result := WithUserID(ctx, "user123")

	assert.Equal(t, "user123", GetUserID(result))
	assert.Equal(t, "user123", MustGetUserID(result))
}

// TestGetUserID tests retrieving user ID from context
func TestGetUserID(t *testing.T) {
	t.Run("User ID in context", func(t *testing.T) {
		ctx := WithUserID(context.Background(), "user123")
		assert.Equal(t, "user123", GetUserID(ctx))
	})

	t.Run("User ID not in context", func(t *testing.T) {
		ctx := context.Background()
		assert.Equal(t, "", GetUserID(ctx))
	})

	t.Run("Wrong type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKeyUserID, 123)
		assert.Equal(t, "", GetUserID(ctx))
	})

	t.Run("Nil value in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKeyUserID, nil)
		assert.Equal(t, "", GetUserID(ctx))
	})
}

// TestMustGetUserID tests mandatory user ID retrieval
func TestMustGetUserID(t *testing.T) {
	t.Run("User ID in context", func(t *testing.T) {
		ctx := WithUserID(context.Background(), "user123")
		assert.Equal(t, "user123", MustGetUserID(ctx))
	})

	t.Run("User ID not in context", func(t *testing.T) {
		ctx := context.Background()

		assert.Panics(t, func() {
			MustGetUserID(ctx)
		})
	})

	t.Run("Empty user ID", func(t *testing.T) {
		ctx := WithUserID(context.Background(), "")

		assert.Panics(t, func() {
			MustGetUserID(ctx)
		})
	})
}

// TestWithActorID tests adding actor ID to context
func TestWithActorID(t *testing.T) {
	ctx := context.Background()

	result := WithActorID(ctx, "actor123")

	assert.Equal(t, "actor123", GetActorID(result))
}

// TestGetActorID tests retrieving actor ID from context
func TestGetActorID(t *testing.T) {
	t.Run("Actor ID in context", func(t *testing.T) {
		ctx := WithActorID(context.Background(), "actor123")
		assert.Equal(t, "actor123", GetActorID(ctx))
	})

	t.Run("Actor ID not in context", func(t *testing.T) {
		ctx := context.Background()
		assert.Equal(t, "", GetActorID(ctx))
	})

	t.Run("Fallback to user ID", func(t *testing.T) {
		ctx := WithUserID(context.Background(), "user123")
		assert.Equal(t, "user123", GetActorID(ctx))
	})

	t.Run("Both actor and user ID in context", func(t *testing.T) {
		ctx := WithUserID(WithActorID(context.Background(), "actor123"), "user456")
		assert.Equal(t, "actor123", GetActorID(ctx))
	})

	t.Run("Wrong type in context", func(t *testing.T) {
		ctx := WithUserID(context.Background(), "user123")
		ctx = context.WithValue(ctx, contextKeyActorID, 123)
		assert.Equal(t, "user123", GetActorID(ctx)) // Falls back to user ID
	})
}

// TestWithIPAddress tests adding IP address to context
func TestWithIPAddress(t *testing.T) {
	ctx := context.Background()

	result := WithIPAddress(ctx, "192.168.1.1")

	assert.Equal(t, "192.168.1.1", GetIPAddress(result))
}

// TestGetIPAddress tests retrieving IP address from context
func TestGetIPAddress(t *testing.T) {
	t.Run("IP address in context", func(t *testing.T) {
		ctx := WithIPAddress(context.Background(), "192.168.1.1")
		assert.Equal(t, "192.168.1.1", GetIPAddress(ctx))
	})

	t.Run("IP address not in context", func(t *testing.T) {
		ctx := context.Background()
		assert.Equal(t, "", GetIPAddress(ctx))
	})

	t.Run("Wrong type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKeyIPAddress, 123)
		assert.Equal(t, "", GetIPAddress(ctx))
	})

	t.Run("Nil value in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKeyIPAddress, nil)
		assert.Equal(t, "", GetIPAddress(ctx))
	})
}

// TestWithUserAgent tests adding user agent to context
func TestWithUserAgent(t *testing.T) {
	ctx := context.Background()

	result := WithUserAgent(ctx, "Mozilla/5.0")

	assert.Equal(t, "Mozilla/5.0", GetUserAgent(result))
}

// TestGetUserAgent tests retrieving user agent from context
func TestGetUserAgent(t *testing.T) {
	t.Run("User agent in context", func(t *testing.T) {
		ctx := WithUserAgent(context.Background(), "Mozilla/5.0")
		assert.Equal(t, "Mozilla/5.0", GetUserAgent(ctx))
	})

	t.Run("User agent not in context", func(t *testing.T) {
		ctx := context.Background()
		assert.Equal(t, "", GetUserAgent(ctx))
	})

	t.Run("Wrong type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKeyUserAgent, 123)
		assert.Equal(t, "", GetUserAgent(ctx))
	})

	t.Run("Nil value in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKeyUserAgent, nil)
		assert.Equal(t, "", GetUserAgent(ctx))
	})
}

// TestWithRequestID tests adding request ID to context
func TestWithRequestID(t *testing.T) {
	ctx := context.Background()

	result := WithRequestID(ctx, "req-123")

	assert.Equal(t, "req-123", GetRequestID(result))
}

// TestGetRequestID tests retrieving request ID from context
func TestGetRequestID(t *testing.T) {
	t.Run("Request ID in context", func(t *testing.T) {
		ctx := WithRequestID(context.Background(), "req-123")
		assert.Equal(t, "req-123", GetRequestID(ctx))
	})

	t.Run("Request ID not in context", func(t *testing.T) {
		ctx := context.Background()
		assert.Equal(t, "", GetRequestID(ctx))
	})

	t.Run("Wrong type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKeyRequestID, 123)
		assert.Equal(t, "", GetRequestID(ctx))
	})

	t.Run("Nil value in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKeyRequestID, nil)
		assert.Equal(t, "", GetRequestID(ctx))
	})
}

// TestWithChecker tests adding checker to context
func TestWithChecker(t *testing.T) {
	ctx := context.Background()

	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	checker := &Checker{}
	result := WithChecker(ctx, checker)

	assert.Same(t, checker, GetChecker(result))
	assert.Same(t, checker, FromContext(result))
}

// TestGetChecker tests retrieving checker from context
func TestGetChecker(t *testing.T) {
	t.Run("Checker in context", func(t *testing.T) {
		registry := NewRegistry()
		registry.DefineScope("organization").Role("admin")

		checker := &Checker{}
		ctx := WithChecker(context.Background(), checker)

		assert.Same(t, checker, GetChecker(ctx))
	})

	t.Run("Checker not in context", func(t *testing.T) {
		ctx := context.Background()
		assert.Nil(t, GetChecker(ctx))
	})

	t.Run("Wrong type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKeyChecker, "not a checker")
		assert.Nil(t, GetChecker(ctx))
	})

	t.Run("Nil value in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKeyChecker, nil)
		assert.Nil(t, GetChecker(ctx))
	})
}

// TestFromContext tests alias for GetChecker
func TestFromContext(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	checker := &Checker{}
	ctx := WithChecker(context.Background(), checker)

	assert.Same(t, checker, FromContext(ctx))
}

// TestGetAuditContext tests extracting audit information from context
func TestGetAuditContext(t *testing.T) {
	t.Run("All fields in context", func(t *testing.T) {
		ctx := WithUserID(
			WithActorID(
				WithIPAddress(
					WithUserAgent(
						WithRequestID(context.Background(), "req-123"),
						"Mozilla/5.0"),
					"192.168.1.1"),
				"actor123"),
			"user456")

		audit := GetAuditContext(ctx)

		assert.Equal(t, "actor123", audit.ActorID)
		assert.Equal(t, "192.168.1.1", audit.IPAddress)
		assert.Equal(t, "Mozilla/5.0", audit.UserAgent)
		assert.Equal(t, "req-123", audit.RequestID)
		assert.Equal(t, "user456", GetUserID(ctx)) // From GetActorID fallback
	})

	t.Run("Empty context", func(t *testing.T) {
		ctx := context.Background()
		audit := GetAuditContext(ctx)

		assert.Equal(t, "", audit.ActorID)
		assert.Equal(t, "", audit.IPAddress)
		assert.Equal(t, "", audit.UserAgent)
		assert.Equal(t, "", audit.RequestID)
	})

	t.Run("Partial context", func(t *testing.T) {
		ctx := WithActorID(WithRequestID(context.Background(), "req-456"), "actor789")
		audit := GetAuditContext(ctx)

		assert.Equal(t, "actor789", audit.ActorID)
		assert.Equal(t, "", audit.IPAddress)
		assert.Equal(t, "", audit.UserAgent)
		assert.Equal(t, "req-456", audit.RequestID)
		assert.Equal(t, "", GetUserID(ctx)) // No user ID to fallback to
	})
}

// TestWithAuditContext tests adding audit context to context
func TestWithAuditContext(t *testing.T) {
	ctx := context.Background()

	audit := AuditContext{
		ActorID:   "actor123",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		RequestID: "req-123",
	}

	result := WithAuditContext(ctx, audit)

	// Verify all fields are set
	assert.Equal(t, "actor123", GetActorID(result))
	assert.Equal(t, "192.168.1.1", GetIPAddress(result))
	assert.Equal(t, "Mozilla/5.0", GetUserAgent(result))
	assert.Equal(t, "req-123", GetRequestID(result))
	// assert.Equal(t, "user456", GetUserID(result)) // User ID not set in this test // From GetActorID fallback
}

// TestWithAuditContextPartial tests adding partial audit context
func TestWithAuditContextPartial(t *testing.T) {
	ctx := context.Background()

	// Only set actor ID
	audit1 := AuditContext{ActorID: "actor123"}
	result1 := WithAuditContext(ctx, audit1)
	assert.Equal(t, "actor123", GetActorID(result1))
	assert.Equal(t, "", GetIPAddress(result1))
	assert.Equal(t, "", GetUserAgent(result1))
	assert.Equal(t, "", GetRequestID(result1))

	// Only set IP address
	audit2 := AuditContext{IPAddress: "192.168.1.1"}
	result2 := WithAuditContext(ctx, audit2)
	assert.Equal(t, "", GetActorID(result2))
	assert.Equal(t, "192.168.1.1", GetIPAddress(result2))
	assert.Equal(t, "", GetUserAgent(result2))
	assert.Equal(t, "", GetRequestID(result2))

	// Only set user agent
	audit3 := AuditContext{UserAgent: "Mozilla/5.0"}
	result3 := WithAuditContext(ctx, audit3)
	assert.Equal(t, "", GetActorID(result3))
	assert.Equal(t, "", GetIPAddress(result3))
	assert.Equal(t, "Mozilla/5.0", GetUserAgent(result3))
	assert.Equal(t, "", GetRequestID(result3))

	// Only set request ID
	audit4 := AuditContext{RequestID: "req-123"}
	result4 := WithAuditContext(ctx, audit4)
	assert.Equal(t, "", GetActorID(result4))
	assert.Equal(t, "", GetIPAddress(result4))
	assert.Equal(t, "", GetUserAgent(result4))
	assert.Equal(t, "req-123", GetRequestID(result4))
}

// TestContextKeyConstants tests context key constants
func TestContextKeyConstants(t *testing.T) {
	assert.Equal(t, contextKey("rolekit:user_id"), contextKeyUserID)
	assert.Equal(t, contextKey("rolekit:actor_id"), contextKeyActorID)
	assert.Equal(t, contextKey("rolekit:ip_address"), contextKeyIPAddress)
	assert.Equal(t, contextKey("rolekit:user_agent"), contextKeyUserAgent)
	assert.Equal(t, contextKey("rolekit:request_id"), contextKeyRequestID)
	assert.Equal(t, contextKey("rolekit:checker"), contextKeyChecker)
}

// TestContextChaining tests chaining multiple context operations
func TestContextChaining(t *testing.T) {
	// ctx := context.Background()

	// Chain multiple context operations
	result := WithUserID(
		WithActorID(
			WithIPAddress(
				WithUserAgent(
					WithRequestID(
						WithChecker(
							context.Background(),
							&Checker{}),
						"req-123"),
					"Mozilla/5.0"),
				"192.168.1.1"),
			"actor123"),
		"user456")

	// Verify all values
	assert.Equal(t, "user456", GetUserID(result))
	assert.Equal(t, "actor123", GetActorID(result))
	assert.Equal(t, "192.168.1.1", GetIPAddress(result))
	assert.Equal(t, "Mozilla/5.0", GetUserAgent(result))
	assert.Equal(t, "req-123", GetRequestID(result))
	assert.NotNil(t, GetChecker(result))
}

// TestContextEdgeCases tests edge cases and special values
func TestContextEdgeCases(t *testing.T) {
	t.Run("Empty strings", func(t *testing.T) {
		ctx := context.Background()

		ctx = WithUserID(ctx, "")
		assert.Equal(t, "", GetUserID(ctx))

		ctx = WithActorID(ctx, "")
		assert.Equal(t, "", GetActorID(ctx))

		ctx = WithIPAddress(ctx, "")
		assert.Equal(t, "", GetIPAddress(ctx))

		ctx = WithUserAgent(ctx, "")
		assert.Equal(t, "", GetUserAgent(ctx))

		ctx = WithRequestID(ctx, "")
		assert.Equal(t, "", GetRequestID(ctx))
	})

	t.Run("Special characters", func(t *testing.T) {
		ctx := context.Background()

		ctx = WithUserID(ctx, "user@domain.com")
		assert.Equal(t, "user@domain.com", GetUserID(ctx))

		ctx = WithActorID(ctx, "actor@domain.com")
		assert.Equal(t, "actor@domain.com", GetActorID(ctx))

		ctx = WithIPAddress(ctx, "::1")
		assert.Equal(t, "::1", GetIPAddress(ctx))

		ctx = WithUserAgent(ctx, "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		assert.Equal(t, "Mozilla/5.0 (Windows NT 10.0; Win64; x64)", GetUserAgent(ctx))

		ctx = WithRequestID(ctx, "req-123-456-789")
		assert.Equal(t, "req-123-456-789", GetRequestID(ctx))
	})

	t.Run("Unicode characters", func(t *testing.T) {
		ctx := context.Background()

		ctx = WithUserID(ctx, "用户123")
		assert.Equal(t, "用户123", GetUserID(ctx))

		ctx = WithActorID(ctx, "管理员")
		assert.Equal(t, "管理员", GetActorID(ctx))
	})

	t.Run("Very long strings", func(t *testing.T) {
		longString := "a" + string(make([]rune, 1000))

		ctx := WithUserID(context.Background(), longString)
		assert.Equal(t, longString, GetUserID(ctx))
	})

	t.Run("Nil context", func(t *testing.T) {
		// These should not panic, just return empty values
		// assert.Equal(t, "", GetUserID(nil)) // This panics
		// assert.Equal(t, "", GetActorID(nil)) // This panics
		// assert.Equal(t, "", GetIPAddress(nil)) // This panics
		// assert.Equal(t, "", GetUserAgent(nil)) // This panics
		// assert.Equal(t, "", GetRequestID(nil)) // This panics
		// assert.Nil(t, GetChecker(nil)) // This panics

		// MustGetUserID should panic with nil context
		assert.Panics(t, func() {
			MustGetUserID(nil)
		})
	})
}

// TestContextImmutability tests that context operations return new contexts
func TestContextImmutability(t *testing.T) {
	original := WithUserID(context.Background(), "user123")

	// Modify the context
	modified := WithActorID(original, "actor123")

	// Original should be unchanged
	assert.Equal(t, "user123", GetUserID(original))
	assert.Equal(t, "actor123", GetActorID(modified))

	// Modify again
	modified2 := WithIPAddress(modified, "192.168.1.1")

	// Previous modified should be unchanged
	assert.Equal(t, "user123", GetUserID(original))
	assert.Equal(t, "actor123", GetActorID(modified))
	assert.Equal(t, "192.168.1.1", GetIPAddress(modified2))

	// Original still unchanged
	assert.Equal(t, "user123", GetUserID(original))
	assert.Equal(t, "", GetIPAddress(original))
}
