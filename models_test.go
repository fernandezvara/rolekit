package rolekit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRoleAssignment tests RoleAssignment struct
func TestRoleAssignment(t *testing.T) {
	t.Run("Create new assignment", func(t *testing.T) {
		assignment := RoleAssignment{
			UserID:    "user123",
			Role:      "admin",
			ScopeType: "organization",
			ScopeID:   "org123",
		}

		assert.Equal(t, "user123", assignment.UserID)
		assert.Equal(t, "admin", assignment.Role)
		assert.Equal(t, "organization", assignment.ScopeType)
		assert.Equal(t, "org123", assignment.ScopeID)
	})

	t.Run("With wildcard scope", func(t *testing.T) {
		assignment := RoleAssignment{
			UserID:    "user123",
			Role:      "admin",
			ScopeType: "organization",
			ScopeID:   "*",
		}

		assert.Equal(t, "*", assignment.ScopeID)
	})

	t.Run("With parent scope", func(t *testing.T) {
		assignment := RoleAssignment{
			UserID:          "user123",
			Role:            "admin",
			ScopeType:       "project",
			ScopeID:         "proj123",
			ParentScopeType: "organization",
			ParentScopeID:   "org123",
		}

		assert.Equal(t, "organization", assignment.ParentScopeType)
		assert.Equal(t, "org123", assignment.ParentScopeID)
	})
}

// TestRoleAuditLog tests RoleAuditLog struct
func TestRoleAuditLog(t *testing.T) {
	t.Run("Create new audit log", func(t *testing.T) {
		log := RoleAuditLog{
			ActorID:      "actor123",
			Action:       "assigned",
			TargetUserID: "user456",
			Role:         "admin",
			ScopeType:    "organization",
			ScopeID:      "org123",
			IPAddress:    "192.168.1.1",
			UserAgent:    "Mozilla/5.0",
			RequestID:    "req-123",
		}

		assert.Equal(t, "actor123", log.ActorID)
		assert.Equal(t, "assigned", log.Action)
		assert.Equal(t, "user456", log.TargetUserID)
		assert.Equal(t, "admin", log.Role)
		assert.Equal(t, "organization", log.ScopeType)
		assert.Equal(t, "org123", log.ScopeID)
		assert.Equal(t, "192.168.1.1", log.IPAddress)
		assert.Equal(t, "Mozilla/5.0", log.UserAgent)
		assert.Equal(t, "req-123", log.RequestID)
	})

	t.Run("With role arrays", func(t *testing.T) {
		log := RoleAuditLog{
			ActorRoles:    []string{"admin", "member"},
			PreviousRoles: []string{"viewer"},
			NewRoles:      []string{"admin", "member", "viewer"},
		}

		assert.Equal(t, []string{"admin", "member"}, log.ActorRoles)
		assert.Equal(t, []string{"viewer"}, log.PreviousRoles)
		assert.Equal(t, []string{"admin", "member", "viewer"}, log.NewRoles)
	})

	t.Run("With metadata", func(t *testing.T) {
		log := RoleAuditLog{
			Metadata: map[string]any{
				"ip_country": "US",
				"session_id": "sess-123",
			},
		}

		assert.Equal(t, "US", log.Metadata["ip_country"])
		assert.Equal(t, "sess-123", log.Metadata["session_id"])
	})
}

// TestScopeHierarchy tests ScopeHierarchy struct
func TestScopeHierarchy(t *testing.T) {
	t.Run("Create new hierarchy", func(t *testing.T) {
		hierarchy := ScopeHierarchy{
			ScopeType:       "project",
			ScopeID:         "proj123",
			ParentScopeType: "organization",
			ParentScopeID:   "org123",
		}

		assert.Equal(t, "project", hierarchy.ScopeType)
		assert.Equal(t, "proj123", hierarchy.ScopeID)
		assert.Equal(t, "organization", hierarchy.ParentScopeType)
		assert.Equal(t, "org123", hierarchy.ParentScopeID)
	})
}

// TestScope tests Scope struct and methods
func TestScope(t *testing.T) {
	t.Run("NewScope", func(t *testing.T) {
		scope := NewScope("organization", "org123")

		assert.Equal(t, "organization", scope.Type)
		assert.Equal(t, "org123", scope.ID)
	})

	t.Run("String", func(t *testing.T) {
		scope := NewScope("organization", "org123")
		assert.Equal(t, "organization:org123", scope.String())
	})

	t.Run("IsWildcard", func(t *testing.T) {
		t.Run("Wildcard scope", func(t *testing.T) {
			scope := NewScope("organization", "*")
			assert.True(t, scope.IsWildcard())
		})

		t.Run("Non-wildcard scope", func(t *testing.T) {
			scope := NewScope("organization", "org123")
			assert.False(t, scope.IsWildcard())
		})
	})
}

// TestNewUserRoles tests UserRoles constructor
func TestNewUserRoles(t *testing.T) {
	t.Run("Empty assignments", func(t *testing.T) {
		ur := NewUserRoles("user123", []RoleAssignment{})

		assert.Equal(t, "user123", ur.UserID)
		assert.Empty(t, ur.Assignments)
		assert.Empty(t, ur.byScope)
	})

	t.Run("Single assignment", func(t *testing.T) {
		assignments := []RoleAssignment{
			{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org123"},
		}

		ur := NewUserRoles("user123", assignments)

		assert.Equal(t, "user123", ur.UserID)
		assert.Len(t, ur.Assignments, 1)
		assert.Contains(t, ur.byScope, "organization:org123")
		assert.Equal(t, []string{"admin"}, ur.byScope["organization:org123"])
	})

	t.Run("Multiple assignments", func(t *testing.T) {
		assignments := []RoleAssignment{
			{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org123"},
			{UserID: "user123", Role: "member", ScopeType: "organization", ScopeID: "org123"},
			{UserID: "user123", Role: "owner", ScopeType: "project", ScopeID: "proj456"},
			{UserID: "user123", Role: "admin", ScopeType: "project", ScopeID: "*"},
		}

		ur := NewUserRoles("user123", assignments)

		assert.Equal(t, "user123", ur.UserID)
		assert.Len(t, ur.Assignments, 4)
		assert.Equal(t, []string{"admin", "member"}, ur.byScope["organization:org123"])
		assert.Equal(t, []string{"owner"}, ur.byScope["project:proj456"])
		assert.Equal(t, []string{"admin"}, ur.byScope["project:*"])
	})

	t.Run("Multiple users in assignments", func(t *testing.T) {
		assignments := []RoleAssignment{
			{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org123"},
			{UserID: "user456", Role: "member", ScopeType: "organization", ScopeID: "org123"},
		}

		ur := NewUserRoles("user123", assignments)

		// NewUserRoles stores all assignments but indexes by scope
		assert.Equal(t, "user123", ur.UserID)
		assert.Len(t, ur.Assignments, 2)                                                // Both assignments are kept
		assert.Equal(t, []string{"admin", "member"}, ur.byScope["organization:org123"]) // All roles in scope
	})
}

// TestUserRoles_GetRoles tests role retrieval
func TestUserRoles_GetRoles(t *testing.T) {
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org123"},
		{UserID: "user123", Role: "member", ScopeType: "organization", ScopeID: "org123"},
		{UserID: "user123", Role: "owner", ScopeType: "project", ScopeID: "proj456"},
		{UserID: "user123", Role: "admin", ScopeType: "project", ScopeID: "*"},
	}

	ur := NewUserRoles("user123", assignments)

	t.Run("Exact match", func(t *testing.T) {
		roles := ur.GetRoles("organization", "org123")
		assert.Equal(t, []string{"admin", "member"}, roles)
	})

	t.Run("Wildcard match", func(t *testing.T) {
		roles := ur.GetRoles("project", "proj789")
		assert.Equal(t, []string{"admin"}, roles) // From wildcard
	})

	t.Run("Both exact and wildcard", func(t *testing.T) {
		roles := ur.GetRoles("project", "proj456")
		assert.Contains(t, roles, "owner") // From exact match
		assert.Contains(t, roles, "admin") // From wildcard
		assert.Len(t, roles, 2)
	})

	t.Run("No match", func(t *testing.T) {
		roles := ur.GetRoles("team", "team123")
		assert.Empty(t, roles)
	})

	t.Run("Empty scope", func(t *testing.T) {
		roles := ur.GetRoles("", "")
		assert.Empty(t, roles)
	})
}

// TestUserRoles_HasRole tests role checking
func TestUserRoles_HasRole(t *testing.T) {
	assignments := []RoleAssignment{
		{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org123"},
		{UserID: "user123", Role: "member", ScopeType: "organization", ScopeID: "org123"},
		{UserID: "user123", Role: "owner", ScopeType: "project", ScopeID: "proj456"},
		{UserID: "user123", Role: "admin", ScopeType: "project", ScopeID: "*"},
	}

	ur := NewUserRoles("user123", assignments)

	t.Run("Has role - exact match", func(t *testing.T) {
		assert.True(t, ur.HasRole("admin", "organization", "org123"))
		assert.True(t, ur.HasRole("member", "organization", "org123"))
		assert.True(t, ur.HasRole("owner", "project", "proj456"))
	})

	t.Run("Has role - wildcard match", func(t *testing.T) {
		assert.True(t, ur.HasRole("admin", "project", "proj789"))
	})

	t.Run("Does not have role", func(t *testing.T) {
		assert.False(t, ur.HasRole("viewer", "organization", "org123"))
		assert.False(t, ur.HasRole("member", "project", "proj456"))
	})

	t.Run("Empty scope", func(t *testing.T) {
		assert.False(t, ur.HasRole("admin", "", ""))
	})

	t.Run("Empty role", func(t *testing.T) {
		assert.False(t, ur.HasRole("", "organization", "org123"))
	})
}

// TestAuditAction tests audit action constants
func TestAuditAction(t *testing.T) {
	t.Run("Constants", func(t *testing.T) {
		assert.Equal(t, AuditAction("assigned"), AuditActionAssigned)
		assert.Equal(t, AuditAction("revoked"), AuditActionRevoked)
	})

	t.Run("String values", func(t *testing.T) {
		assert.Equal(t, "assigned", string(AuditActionAssigned))
		assert.Equal(t, "revoked", string(AuditActionRevoked))
	})
}

// TestAuditEntry tests AuditEntry struct
func TestAuditEntry(t *testing.T) {
	t.Run("Create new entry", func(t *testing.T) {
		entry := AuditEntry{
			ActorID:       "actor123",
			Action:        AuditActionAssigned,
			TargetUserID:  "user456",
			Role:          "admin",
			ScopeType:     "organization",
			ScopeID:       "org123",
			ActorRoles:    []string{"admin", "member"},
			PreviousRoles: []string{"viewer"},
			NewRoles:      []string{"admin", "member", "viewer"},
			IPAddress:     "192.168.1.1",
			UserAgent:     "Mozilla/5.0",
			RequestID:     "req-123",
			Metadata:      map[string]any{"key": "value"},
		}

		assert.Equal(t, "actor123", entry.ActorID)
		assert.Equal(t, AuditActionAssigned, entry.Action)
		assert.Equal(t, "user456", entry.TargetUserID)
		assert.Equal(t, "admin", entry.Role)
		assert.Equal(t, "organization", entry.ScopeType)
		assert.Equal(t, "org123", entry.ScopeID)
		assert.Equal(t, []string{"admin", "member"}, entry.ActorRoles)
		assert.Equal(t, []string{"viewer"}, entry.PreviousRoles)
		assert.Equal(t, []string{"admin", "member", "viewer"}, entry.NewRoles)
		assert.Equal(t, "192.168.1.1", entry.IPAddress)
		assert.Equal(t, "Mozilla/5.0", entry.UserAgent)
		assert.Equal(t, "req-123", entry.RequestID)
		assert.Equal(t, "value", entry.Metadata["key"])
	})
}

// TestAuditEntry_ToModel tests conversion to RoleAuditLog
func TestAuditEntry_ToModel(t *testing.T) {
	entry := AuditEntry{
		ActorID:       "actor123",
		Action:        AuditActionAssigned,
		TargetUserID:  "user456",
		Role:          "admin",
		ScopeType:     "organization",
		ScopeID:       "org123",
		ActorRoles:    []string{"admin", "member"},
		PreviousRoles: []string{"viewer"},
		NewRoles:      []string{"admin", "member", "viewer"},
		IPAddress:     "192.168.1.1",
		UserAgent:     "Mozilla/5.0",
		RequestID:     "req-123",
		Metadata:      map[string]any{"key": "value"},
	}

	model := entry.ToModel()

	assert.Equal(t, "actor123", model.ActorID)
	assert.Equal(t, "assigned", model.Action)
	assert.Equal(t, "user456", model.TargetUserID)
	assert.Equal(t, "admin", model.Role)
	assert.Equal(t, "organization", model.ScopeType)
	assert.Equal(t, "org123", model.ScopeID)
	assert.Equal(t, []string{"admin", "member"}, model.ActorRoles)
	assert.Equal(t, []string{"viewer"}, model.PreviousRoles)
	assert.Equal(t, []string{"admin", "member", "viewer"}, model.NewRoles)
	assert.Equal(t, "192.168.1.1", model.IPAddress)
	assert.Equal(t, "Mozilla/5.0", model.UserAgent)
	assert.Equal(t, "req-123", model.RequestID)
	assert.Equal(t, "value", model.Metadata["key"])
	assert.NotZero(t, model.Timestamp)
	assert.WithinDuration(t, time.Now(), model.Timestamp, time.Second)
}

// TestModelsEdgeCases tests edge cases and special values
func TestModelsEdgeCases(t *testing.T) {
	t.Run("Empty strings in models", func(t *testing.T) {
		assignment := RoleAssignment{
			UserID:    "",
			Role:      "",
			ScopeType: "",
			ScopeID:   "",
		}

		assert.Equal(t, "", assignment.UserID)
		assert.Equal(t, "", assignment.Role)
		assert.Equal(t, "", assignment.ScopeType)
		assert.Equal(t, "", assignment.ScopeID)
	})

	t.Run("Special characters in IDs", func(t *testing.T) {
		assignment := RoleAssignment{
			UserID:    "user@domain.com",
			Role:      "admin",
			ScopeType: "organization",
			ScopeID:   "org-123_456",
		}

		assert.Equal(t, "user@domain.com", assignment.UserID)
		assert.Equal(t, "org-123_456", assignment.ScopeID)
	})

	t.Run("Unicode characters", func(t *testing.T) {
		assignment := RoleAssignment{
			UserID:    "用户123",
			Role:      "管理员",
			ScopeType: "组织",
			ScopeID:   "组织123",
		}

		assert.Equal(t, "用户123", assignment.UserID)
		assert.Equal(t, "管理员", assignment.Role)
		assert.Equal(t, "组织", assignment.ScopeType)
		assert.Equal(t, "组织123", assignment.ScopeID)
	})

	t.Run("Nil slices", func(t *testing.T) {
		log := RoleAuditLog{
			ActorRoles:    nil,
			PreviousRoles: nil,
			NewRoles:      nil,
		}

		assert.Nil(t, log.ActorRoles)
		assert.Nil(t, log.PreviousRoles)
		assert.Nil(t, log.NewRoles)
	})

	t.Run("Empty slices", func(t *testing.T) {
		log := RoleAuditLog{
			ActorRoles:    []string{},
			PreviousRoles: []string{},
			NewRoles:      []string{},
		}

		assert.Empty(t, log.ActorRoles)
		assert.Empty(t, log.PreviousRoles)
		assert.Empty(t, log.NewRoles)
	})

	t.Run("Nil metadata", func(t *testing.T) {
		entry := AuditEntry{
			Metadata: nil,
		}

		assert.Nil(t, entry.Metadata)

		model := entry.ToModel()
		assert.Nil(t, model.Metadata)
	})

	t.Run("Empty metadata", func(t *testing.T) {
		entry := AuditEntry{
			Metadata: map[string]any{},
		}

		assert.Empty(t, entry.Metadata)

		model := entry.ToModel()
		assert.Empty(t, model.Metadata)
	})
}

// TestModelsTimeFields tests time-related fields
func TestModelsTimeFields(t *testing.T) {
	t.Run("Zero time", func(t *testing.T) {
		zero := time.Time{}
		assignment := RoleAssignment{
			CreatedAt: zero,
			UpdatedAt: zero,
		}

		assert.True(t, assignment.CreatedAt.IsZero())
		assert.True(t, assignment.UpdatedAt.IsZero())
	})

	t.Run("Current time", func(t *testing.T) {
		now := time.Now()
		assignment := RoleAssignment{
			CreatedAt: now,
			UpdatedAt: now,
		}

		assert.Equal(t, now, assignment.CreatedAt)
		assert.Equal(t, now, assignment.UpdatedAt)
	})
}

// TestModelsComplexScenarios tests complex scenarios
func TestModelsComplexScenarios(t *testing.T) {
	t.Run("User with multiple roles in multiple scopes", func(t *testing.T) {
		assignments := []RoleAssignment{
			{UserID: "user123", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
			{UserID: "user123", Role: "member", ScopeType: "organization", ScopeID: "org1"},
			{UserID: "user123", Role: "owner", ScopeType: "project", ScopeID: "proj1"},
			{UserID: "user123", Role: "member", ScopeType: "project", ScopeID: "proj2"},
			{UserID: "user123", Role: "viewer", ScopeType: "project", ScopeID: "*"},
		}

		ur := NewUserRoles("user123", assignments)

		// Check organization roles
		orgRoles := ur.GetRoles("organization", "org1")
		assert.Equal(t, []string{"admin", "member"}, orgRoles)

		// Check specific project roles
		proj1Roles := ur.GetRoles("project", "proj1")
		assert.Equal(t, []string{"owner"}, proj1Roles[:1])

		// Check project with wildcard
		proj3Roles := ur.GetRoles("project", "proj3")
		assert.Equal(t, []string{"viewer"}, proj3Roles)

		// Check combined roles
		proj2Roles := ur.GetRoles("project", "proj2")
		assert.Equal(t, []string{"member", "viewer"}, proj2Roles)
	})

	t.Run("Audit trail for role assignment", func(t *testing.T) {
		entry := AuditEntry{
			ActorID:       "admin123",
			Action:        AuditActionAssigned,
			TargetUserID:  "user456",
			Role:          "member",
			ScopeType:     "project",
			ScopeID:       "proj789",
			ActorRoles:    []string{"admin", "owner"},
			PreviousRoles: []string{"viewer"},
			NewRoles:      []string{"viewer", "member"},
			IPAddress:     "10.0.0.1",
			UserAgent:     "curl/7.68.0",
			RequestID:     "req-abc-123",
			Metadata: map[string]any{
				"source": "api",
				"reason": "project invitation",
			},
		}

		model := entry.ToModel()

		assert.Equal(t, "admin123", model.ActorID)
		assert.Equal(t, "assigned", model.Action)
		assert.Equal(t, "user456", model.TargetUserID)
		assert.Equal(t, "member", model.Role)
		assert.Equal(t, "project", model.ScopeType)
		assert.Equal(t, "proj789", model.ScopeID)
		assert.Equal(t, []string{"admin", "owner"}, model.ActorRoles)
		assert.Equal(t, []string{"viewer"}, model.PreviousRoles)
		assert.Equal(t, []string{"viewer", "member"}, model.NewRoles)
		assert.Equal(t, "10.0.0.1", model.IPAddress)
		assert.Equal(t, "curl/7.68.0", model.UserAgent)
		assert.Equal(t, "req-abc-123", model.RequestID)
		assert.Equal(t, "api", model.Metadata["source"])
		assert.Equal(t, "project invitation", model.Metadata["reason"])
	})
}
