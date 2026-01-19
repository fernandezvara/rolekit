package rolekit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewAuditLogFilter tests creating a new audit log filter
func TestNewAuditLogFilter(t *testing.T) {
	filter := NewAuditLogFilter()

	assert.Equal(t, 100, filter.Limit)
	assert.Equal(t, 0, filter.Offset)
	assert.Equal(t, "", filter.ActorID)
	assert.Equal(t, "", filter.TargetUserID)
	assert.Equal(t, "", filter.ScopeType)
	assert.Equal(t, "", filter.ScopeID)
	assert.Equal(t, "", filter.Action)
	assert.Equal(t, "", filter.Role)
	assert.True(t, filter.Since.IsZero())
	assert.True(t, filter.Until.IsZero())
}

// TestAuditLogFilterWithActor tests setting actor filter
func TestAuditLogFilterWithActor(t *testing.T) {
	filter := NewAuditLogFilter()

	result := filter.WithActor("actor123")

	assert.Equal(t, "actor123", result.ActorID)
	assert.Equal(t, 100, result.Limit) // Other fields unchanged
	assert.Equal(t, 0, result.Offset)
}

// TestAuditLogFilterWithTargetUser tests setting target user filter
func TestAuditLogFilterWithTargetUser(t *testing.T) {
	filter := NewAuditLogFilter()

	result := filter.WithTargetUser("user123")

	assert.Equal(t, "user123", result.TargetUserID)
	assert.Equal(t, 100, result.Limit) // Other fields unchanged
}

// TestAuditLogFilterWithScope tests setting scope filter
func TestAuditLogFilterWithScope(t *testing.T) {
	filter := NewAuditLogFilter()

	result := filter.WithScope("organization", "org123")

	assert.Equal(t, "organization", result.ScopeType)
	assert.Equal(t, "org123", result.ScopeID)
	assert.Equal(t, 100, result.Limit) // Other fields unchanged
}

// TestAuditLogFilterWithScopeType tests setting scope type filter only
func TestAuditLogFilterWithScopeType(t *testing.T) {
	filter := NewAuditLogFilter()

	result := filter.WithScopeType("project")

	assert.Equal(t, "project", result.ScopeType)
	assert.Equal(t, "", result.ScopeID) // ScopeID unchanged
	assert.Equal(t, 100, result.Limit)  // Other fields unchanged
}

// TestAuditLogFilterWithAction tests setting action filter
func TestAuditLogFilterWithAction(t *testing.T) {
	filter := NewAuditLogFilter()

	result := filter.WithAction(AuditActionAssigned)

	assert.Equal(t, "assigned", result.Action)
	assert.Equal(t, 100, result.Limit) // Other fields unchanged
}

// TestAuditLogFilterWithRole tests setting role filter
func TestAuditLogFilterWithRole(t *testing.T) {
	filter := NewAuditLogFilter()

	result := filter.WithRole("admin")

	assert.Equal(t, "admin", result.Role)
	assert.Equal(t, 100, result.Limit) // Other fields unchanged
}

// TestAuditLogFilterWithTimeRange tests setting time range filter
func TestAuditLogFilterWithTimeRange(t *testing.T) {
	filter := NewAuditLogFilter()

	since := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)

	result := filter.WithTimeRange(since, until)

	assert.Equal(t, since, result.Since)
	assert.Equal(t, until, result.Until)
	assert.Equal(t, 100, result.Limit) // Other fields unchanged
}

// TestAuditLogFilterWithSince tests setting start time filter
func TestAuditLogFilterWithSince(t *testing.T) {
	filter := NewAuditLogFilter()

	since := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)

	result := filter.WithSince(since)

	assert.Equal(t, since, result.Since)
	assert.True(t, result.Until.IsZero()) // Until unchanged
	assert.Equal(t, 100, result.Limit)    // Other fields unchanged
}

// TestAuditLogFilterWithUntil tests setting end time filter
func TestAuditLogFilterWithUntil(t *testing.T) {
	filter := NewAuditLogFilter()

	until := time.Date(2023, 6, 30, 23, 59, 59, 0, time.UTC)

	result := filter.WithUntil(until)

	assert.True(t, result.Since.IsZero()) // Since unchanged
	assert.Equal(t, until, result.Until)
	assert.Equal(t, 100, result.Limit) // Other fields unchanged
}

// TestAuditLogFilterWithLimit tests setting limit
func TestAuditLogFilterWithLimit(t *testing.T) {
	filter := NewAuditLogFilter()

	result := filter.WithLimit(50)

	assert.Equal(t, 50, result.Limit)
	assert.Equal(t, 0, result.Offset) // Other fields unchanged
}

// TestAuditLogFilterWithOffset tests setting offset
func TestAuditLogFilterWithOffset(t *testing.T) {
	filter := NewAuditLogFilter()

	result := filter.WithOffset(10)

	assert.Equal(t, 10, result.Offset)
	assert.Equal(t, 100, result.Limit) // Other fields unchanged
}

// TestAuditLogFilterWithPagination tests setting both limit and offset
func TestAuditLogFilterWithPagination(t *testing.T) {
	filter := NewAuditLogFilter()

	result := filter.WithPagination(25, 50)

	assert.Equal(t, 25, result.Limit)
	assert.Equal(t, 50, result.Offset)
}

// TestAuditLogFilterChaining tests method chaining
func TestAuditLogFilterChaining(t *testing.T) {
	since := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)

	filter := NewAuditLogFilter().
		WithActor("actor123").
		WithTargetUser("user123").
		WithScope("organization", "org123").
		WithAction(AuditActionAssigned).
		WithRole("admin").
		WithTimeRange(since, until).
		WithLimit(50).
		WithOffset(10)

	assert.Equal(t, "actor123", filter.ActorID)
	assert.Equal(t, "user123", filter.TargetUserID)
	assert.Equal(t, "organization", filter.ScopeType)
	assert.Equal(t, "org123", filter.ScopeID)
	assert.Equal(t, "assigned", filter.Action)
	assert.Equal(t, "admin", filter.Role)
	assert.Equal(t, since, filter.Since)
	assert.Equal(t, until, filter.Until)
	assert.Equal(t, 50, filter.Limit)
	assert.Equal(t, 10, filter.Offset)
}

// TestAuditLogFilterEdgeCases tests edge cases and special values
func TestAuditLogFilterEdgeCases(t *testing.T) {
	t.Run("Empty strings", func(t *testing.T) {
		filter := NewAuditLogFilter()

		result := filter.WithActor("")
		assert.Equal(t, "", result.ActorID)

		result2 := filter.WithTargetUser("")
		assert.Equal(t, "", result2.TargetUserID)

		result3 := filter.WithScope("", "")
		assert.Equal(t, "", result3.ScopeType)
		assert.Equal(t, "", result3.ScopeID)
	})

	t.Run("Zero values", func(t *testing.T) {
		filter := NewAuditLogFilter()

		result := filter.WithLimit(0)
		assert.Equal(t, 0, result.Limit)

		result2 := filter.WithOffset(0)
		assert.Equal(t, 0, result2.Offset)

		result3 := filter.WithPagination(0, 0)
		assert.Equal(t, 0, result3.Limit)
		assert.Equal(t, 0, result3.Offset)
	})

	t.Run("Negative values", func(t *testing.T) {
		filter := NewAuditLogFilter()

		result := filter.WithLimit(-1)
		assert.Equal(t, -1, result.Limit)

		result2 := filter.WithOffset(-1)
		assert.Equal(t, -1, result2.Offset)

		result3 := filter.WithPagination(-5, -10)
		assert.Equal(t, -5, result3.Limit)
		assert.Equal(t, -10, result3.Offset)
	})

	t.Run("Zero time", func(t *testing.T) {
		filter := NewAuditLogFilter()

		zero := time.Time{}

		result := filter.WithTimeRange(zero, zero)
		assert.True(t, result.Since.IsZero())
		assert.True(t, result.Until.IsZero())

		result = filter.WithSince(zero)
		assert.True(t, result.Since.IsZero())

		result2 := filter.WithUntil(zero)
		assert.True(t, result2.Until.IsZero())
	})

	t.Run("Unix epoch time", func(t *testing.T) {
		filter := NewAuditLogFilter()

		epoch := time.Unix(0, 0)

		result := filter.WithSince(epoch)
		assert.Equal(t, epoch, result.Since)
		assert.False(t, result.Since.IsZero()) // Unix epoch is not zero

		result2 := filter.WithUntil(epoch)
		assert.Equal(t, epoch, result2.Until)
		assert.False(t, result2.Until.IsZero()) // Unix epoch is not zero
	})
}

// TestAuditLogFilterImmutability tests that methods return new instances
func TestAuditLogFilterImmutability(t *testing.T) {
	original := NewAuditLogFilter()

	// Modify the filter
	modified := original.WithActor("actor123")

	// Original should be unchanged
	assert.Equal(t, "", original.ActorID)
	assert.Equal(t, "actor123", modified.ActorID)

	// Modify again
	modified2 := modified.WithTargetUser("user123")

	// Previous modified should be unchanged
	assert.Equal(t, "actor123", modified.ActorID)
	assert.Equal(t, "", modified.TargetUserID)
	assert.Equal(t, "user123", modified2.TargetUserID)

	// Original still unchanged
	assert.Equal(t, "", original.ActorID)
	assert.Equal(t, "", original.TargetUserID)
}

// TestAuditLogFilterActionConversion tests action string conversion
func TestAuditLogFilterActionConversion(t *testing.T) {
	filter := NewAuditLogFilter()

	// Test with assigned action
	result1 := filter.WithAction(AuditActionAssigned)
	assert.Equal(t, "assigned", result1.Action)

	// Test with revoked action
	result2 := filter.WithAction(AuditActionRevoked)
	assert.Equal(t, "revoked", result2.Action)

	// Test with custom action
	result3 := filter.WithAction("custom_action")
	assert.Equal(t, "custom_action", result3.Action)
}
