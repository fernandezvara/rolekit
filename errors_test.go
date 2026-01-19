package rolekit

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSentinelErrors tests that all sentinel errors are properly defined
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrInvalidScope", ErrInvalidScope, "rolekit: invalid scope"},
		{"ErrInvalidRole", ErrInvalidRole, "rolekit: invalid role"},
		{"ErrInvalidPermission", ErrInvalidPermission, "rolekit: invalid permission"},
		{"ErrUnauthorized", ErrUnauthorized, "rolekit: unauthorized"},
		{"ErrCannotAssign", ErrCannotAssign, "rolekit: cannot assign role"},
		{"ErrRoleAlreadyAssigned", ErrRoleAlreadyAssigned, "rolekit: role already assigned"},
		{"ErrRoleNotAssigned", ErrRoleNotAssigned, "rolekit: role not assigned"},
		{"ErrNoUserID", ErrNoUserID, "rolekit: no user ID in context"},
		{"ErrNoActorID", ErrNoActorID, "rolekit: no actor ID in context"},
		{"ErrDatabaseError", ErrDatabaseError, "rolekit: database error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.msg, tt.err.Error())
			assert.NotNil(t, tt.err)
		})
	}
}

// TestError_Error tests the Error method of Error struct
func TestError_Error(t *testing.T) {
	t.Run("With message", func(t *testing.T) {
		err := &Error{
			Err:     ErrInvalidRole,
			Message: "role 'admin' not defined for scope 'organization'",
		}
		expected := "rolekit: invalid role: role 'admin' not defined for scope 'organization'"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Without message", func(t *testing.T) {
		err := &Error{
			Err: ErrInvalidRole,
		}
		assert.Equal(t, "rolekit: invalid role", err.Error())
	})

	t.Run("Empty message", func(t *testing.T) {
		err := &Error{
			Err:     ErrInvalidRole,
			Message: "",
		}
		assert.Equal(t, "rolekit: invalid role", err.Error())
	})
}

// TestError_Unwrap tests the Unwrap method
func TestError_Unwrap(t *testing.T) {
	err := &Error{
		Err:     ErrInvalidRole,
		Message: "test message",
	}

	assert.Equal(t, ErrInvalidRole, err.Unwrap())
}

// TestError_Is tests the Is method
func TestError_Is(t *testing.T) {
	err := &Error{
		Err:     ErrInvalidRole,
		Message: "test message",
	}

	assert.True(t, err.Is(ErrInvalidRole))
	assert.False(t, err.Is(ErrInvalidScope))
	assert.False(t, err.Is(errors.New("some other error")))
}

// TestNewError tests creating new Error instances
func TestNewError(t *testing.T) {
	err := NewError(ErrInvalidRole, "role not defined")

	assert.Equal(t, ErrInvalidRole, err.Err)
	assert.Equal(t, "role not defined", err.Message)
	assert.Equal(t, "rolekit: invalid role: role not defined", err.Error())
}

// TestError_WithScope tests adding scope information
func TestError_WithScope(t *testing.T) {
	err := NewError(ErrInvalidRole, "role not defined")

	result := err.WithScope("organization", "org123")

	// Should return the same instance (method receiver is a pointer)
	assert.Same(t, err, result)
	assert.Equal(t, "organization", result.Scope)
	assert.Equal(t, "org123", result.ScopeID)
}

// TestError_WithRole tests adding role information
func TestError_WithRole(t *testing.T) {
	err := NewError(ErrInvalidRole, "role not defined")

	result := err.WithRole("admin")

	assert.Same(t, err, result)
	assert.Equal(t, "admin", result.Role)
}

// TestError_WithUser tests adding user information
func TestError_WithUser(t *testing.T) {
	err := NewError(ErrInvalidRole, "role not defined")

	result := err.WithUser("user123")

	assert.Same(t, err, result)
	assert.Equal(t, "user123", result.UserID)
}

// TestError_WithActor tests adding actor information
func TestError_WithActor(t *testing.T) {
	err := NewError(ErrInvalidRole, "role not defined")

	result := err.WithActor("actor123")

	assert.Same(t, err, result)
	assert.Equal(t, "actor123", result.ActorID)
}

// TestError_Chaining tests chaining multiple With methods
func TestError_Chaining(t *testing.T) {
	err := NewError(ErrUnauthorized, "access denied").
		WithScope("organization", "org123").
		WithRole("admin").
		WithUser("user123").
		WithActor("actor456")

	assert.Equal(t, ErrUnauthorized, err.Err)
	assert.Equal(t, "access denied", err.Message)
	assert.Equal(t, "organization", err.Scope)
	assert.Equal(t, "org123", err.ScopeID)
	assert.Equal(t, "admin", err.Role)
	assert.Equal(t, "user123", err.UserID)
	assert.Equal(t, "actor456", err.ActorID)
}

// TestIsUnauthorized tests checking for unauthorized errors
func TestIsUnauthorized(t *testing.T) {
	t.Run("Direct sentinel error", func(t *testing.T) {
		assert.True(t, IsUnauthorized(ErrUnauthorized))
		assert.False(t, IsUnauthorized(ErrInvalidRole))
	})

	t.Run("Wrapped error", func(t *testing.T) {
		err := NewError(ErrUnauthorized, "access denied")
		assert.True(t, IsUnauthorized(err))
		assert.False(t, IsUnauthorized(NewError(ErrInvalidRole, "invalid role")))
	})

	t.Run("Nil error", func(t *testing.T) {
		assert.False(t, IsUnauthorized(nil))
	})

	t.Run("Different error", func(t *testing.T) {
		assert.False(t, IsUnauthorized(errors.New("some other error")))
	})
}

// TestIsInvalidScope tests checking for invalid scope errors
func TestIsInvalidScope(t *testing.T) {
	t.Run("Direct sentinel error", func(t *testing.T) {
		assert.True(t, IsInvalidScope(ErrInvalidScope))
		assert.False(t, IsInvalidScope(ErrInvalidRole))
	})

	t.Run("Wrapped error", func(t *testing.T) {
		err := NewError(ErrInvalidScope, "scope not defined")
		assert.True(t, IsInvalidScope(err))
		assert.False(t, IsInvalidScope(NewError(ErrInvalidRole, "invalid role")))
	})

	t.Run("Nil error", func(t *testing.T) {
		assert.False(t, IsInvalidScope(nil))
	})

	t.Run("Different error", func(t *testing.T) {
		assert.False(t, IsInvalidScope(errors.New("some other error")))
	})
}

// TestIsInvalidRole tests checking for invalid role errors
func TestIsInvalidRole(t *testing.T) {
	t.Run("Direct sentinel error", func(t *testing.T) {
		assert.True(t, IsInvalidRole(ErrInvalidRole))
		assert.False(t, IsInvalidRole(ErrInvalidScope))
	})

	t.Run("Wrapped error", func(t *testing.T) {
		err := NewError(ErrInvalidRole, "role not defined")
		assert.True(t, IsInvalidRole(err))
		assert.False(t, IsInvalidRole(NewError(ErrInvalidScope, "invalid scope")))
	})

	t.Run("Nil error", func(t *testing.T) {
		assert.False(t, IsInvalidRole(nil))
	})

	t.Run("Different error", func(t *testing.T) {
		assert.False(t, IsInvalidRole(errors.New("some other error")))
	})
}

// TestIsCannotAssign tests checking for cannot assign errors
func TestIsCannotAssign(t *testing.T) {
	t.Run("Direct sentinel error", func(t *testing.T) {
		assert.True(t, IsCannotAssign(ErrCannotAssign))
		assert.False(t, IsCannotAssign(ErrInvalidRole))
	})

	t.Run("Wrapped error", func(t *testing.T) {
		err := NewError(ErrCannotAssign, "cannot assign role")
		assert.True(t, IsCannotAssign(err))
		assert.False(t, IsCannotAssign(NewError(ErrInvalidRole, "invalid role")))
	})

	t.Run("Nil error", func(t *testing.T) {
		assert.False(t, IsCannotAssign(nil))
	})

	t.Run("Different error", func(t *testing.T) {
		assert.False(t, IsCannotAssign(errors.New("some other error")))
	})
}

// TestError_EdgeCases tests edge cases and special values
func TestError_EdgeCases(t *testing.T) {
	t.Run("Empty strings in fields", func(t *testing.T) {
		err := &Error{
			Err:     ErrInvalidRole,
			Message: "",
			Scope:   "",
			ScopeID: "",
			Role:    "",
			UserID:  "",
			ActorID: "",
		}
		assert.Equal(t, "rolekit: invalid role", err.Error())
	})

	t.Run("Special characters in message", func(t *testing.T) {
		err := NewError(ErrInvalidRole, "角色 '管理员' 未定义")
		expected := "rolekit: invalid role: 角色 '管理员' 未定义"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Very long message", func(t *testing.T) {
		longMessage := "a" + string(make([]rune, 1000))
		err := NewError(ErrInvalidRole, longMessage)
		assert.Contains(t, err.Error(), longMessage)
	})

	t.Run("Nil underlying error", func(t *testing.T) {
		err := &Error{
			Err:     nil,
			Message: "test message",
		}
		// This should panic when calling Error()
		assert.Panics(t, func() {
			_ = err.Error()
		})
	})

	t.Run("Nil Error pointer", func(t *testing.T) {
		var err *Error
		assert.Nil(t, err)
	})
}

// TestError_WithMethodsReturnSameInstance tests that With methods return the same instance
func TestError_WithMethodsReturnSameInstance(t *testing.T) {
	original := NewError(ErrInvalidRole, "test")

	// Each With method should return the same instance
	withScope := original.WithScope("org", "123")
	assert.Same(t, original, withScope)

	withRole := original.WithRole("admin")
	assert.Same(t, original, withRole)

	withUser := original.WithUser("user123")
	assert.Same(t, original, withUser)

	withActor := original.WithActor("actor123")
	assert.Same(t, original, withActor)
}

// TestError_ImmutabilityOfOtherInstances tests that modifying one error doesn't affect others
func TestError_ImmutabilityOfOtherInstances(t *testing.T) {
	err1 := NewError(ErrInvalidRole, "test1")
	err2 := NewError(ErrInvalidScope, "test2")

	// Modify err1
	err1.WithScope("org", "123").WithRole("admin")

	// err2 should be unchanged
	assert.Equal(t, "", err2.Scope)
	assert.Equal(t, "", err2.ScopeID)
	assert.Equal(t, "", err2.Role)
}

// TestError_CompatibilityWithStandardErrors tests compatibility with Go's error handling
func TestError_CompatibilityWithStandardErrors(t *testing.T) {
	err := NewError(ErrInvalidRole, "test message")

	// Test with errors.Is
	assert.True(t, errors.Is(err, ErrInvalidRole))
	assert.False(t, errors.Is(err, ErrInvalidScope))

	// Test with errors.As
	var target *Error
	assert.True(t, errors.As(err, &target))
	assert.Same(t, err, target)

	// Test with custom error
	customErr := errors.New("custom error")
	assert.False(t, errors.As(customErr, &target))
}

// TestError_AllSentinelErrors tests that all sentinel errors can be wrapped
func TestError_AllSentinelErrors(t *testing.T) {
	sentinelErrors := []error{
		ErrInvalidScope,
		ErrInvalidRole,
		ErrInvalidPermission,
		ErrUnauthorized,
		ErrCannotAssign,
		ErrRoleAlreadyAssigned,
		ErrRoleNotAssigned,
		ErrNoUserID,
		ErrNoActorID,
		ErrDatabaseError,
	}

	for _, sentinel := range sentinelErrors {
		t.Run(sentinel.Error(), func(t *testing.T) {
			wrapped := NewError(sentinel, "test message")

			assert.Equal(t, sentinel, wrapped.Err)
			assert.Equal(t, "test message", wrapped.Message)
			assert.True(t, errors.Is(wrapped, sentinel))

			// Test that the wrapped error can be unwrapped
			assert.Equal(t, sentinel, errors.Unwrap(wrapped))
		})
	}
}
