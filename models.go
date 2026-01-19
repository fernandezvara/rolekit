package rolekit

import (
	"time"

	"github.com/uptrace/bun"
)

// RoleAssignment represents a user's role in a specific scope.
// A user can have multiple roles in the same scope (permissions are UNION).
type RoleAssignment struct {
	bun.BaseModel `bun:"table:role_assignments,alias:ra"`

	ID        string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	UserID    string    `bun:"user_id,notnull"`
	Role      string    `bun:"role,notnull"`
	ScopeType string    `bun:"scope_type,notnull"`
	ScopeID   string    `bun:"scope_id,notnull"` // Can be "*" for wildcard
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp"`

	// Optional: parent scope for hierarchical queries
	ParentScopeType string `bun:"parent_scope_type"`
	ParentScopeID   string `bun:"parent_scope_id"`
}

// RoleAuditLog records all role assignment changes for compliance and debugging.
type RoleAuditLog struct {
	bun.BaseModel `bun:"table:role_audit_log,alias:ral"`

	ID        string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	Timestamp time.Time `bun:"timestamp,notnull,default:current_timestamp"`

	// Who performed the action
	ActorID string `bun:"actor_id,notnull"`

	// What action was performed
	Action string `bun:"action,notnull"` // "assigned", "revoked"

	// Target of the action
	TargetUserID string `bun:"target_user_id,notnull"`
	Role         string `bun:"role,notnull"`
	ScopeType    string `bun:"scope_type,notnull"`
	ScopeID      string `bun:"scope_id,notnull"`

	// Context: what roles did the users have at the time?
	ActorRoles    []string `bun:"actor_roles,type:text[]"`    // Actor's roles in this scope
	PreviousRoles []string `bun:"previous_roles,type:text[]"` // Target's roles before change
	NewRoles      []string `bun:"new_roles,type:text[]"`      // Target's roles after change

	// Request metadata for forensics
	IPAddress string `bun:"ip_address"`
	UserAgent string `bun:"user_agent"`
	RequestID string `bun:"request_id"`

	// Additional context (JSON)
	Metadata map[string]any `bun:"metadata,type:jsonb"`
}

// ScopeHierarchy stores the parent-child relationships between scopes.
// This is used for hierarchical queries like "get all projects in org where user has role X".
type ScopeHierarchy struct {
	bun.BaseModel `bun:"table:scope_hierarchy,alias:sh"`

	ID              string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ScopeType       string    `bun:"scope_type,notnull"`
	ScopeID         string    `bun:"scope_id,notnull"`
	ParentScopeType string    `bun:"parent_scope_type,notnull"`
	ParentScopeID   string    `bun:"parent_scope_id,notnull"`
	CreatedAt       time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

// Scope represents a scope context for permission checks.
type Scope struct {
	Type string // e.g., "organization", "project"
	ID   string // e.g., "org_123", "proj_456", or "*" for wildcard
}

// NewScope creates a new Scope.
func NewScope(scopeType, scopeID string) Scope {
	return Scope{Type: scopeType, ID: scopeID}
}

// String returns a string representation of the scope.
func (s Scope) String() string {
	return s.Type + ":" + s.ID
}

// IsWildcard returns true if this is a wildcard scope (matches all IDs).
func (s Scope) IsWildcard() bool {
	return s.ID == "*"
}

// UserRoles holds all roles for a user, organized by scope.
type UserRoles struct {
	UserID      string
	Assignments []RoleAssignment

	// Indexed for fast lookup
	byScope map[string][]string // scope_type:scope_id -> []roles
}

// NewUserRoles creates a UserRoles from a list of assignments.
func NewUserRoles(userID string, assignments []RoleAssignment) *UserRoles {
	ur := &UserRoles{
		UserID:      userID,
		Assignments: assignments,
		byScope:     make(map[string][]string),
	}

	for _, a := range assignments {
		key := a.ScopeType + ":" + a.ScopeID
		ur.byScope[key] = append(ur.byScope[key], a.Role)
	}

	return ur
}

// GetRoles returns all roles for a specific scope.
// Also checks for wildcard assignments (scope_id = "*").
func (ur *UserRoles) GetRoles(scopeType, scopeID string) []string {
	var roles []string

	// Check exact match
	key := scopeType + ":" + scopeID
	if r, ok := ur.byScope[key]; ok {
		roles = append(roles, r...)
	}

	// Check wildcard match
	wildcardKey := scopeType + ":*"
	if r, ok := ur.byScope[wildcardKey]; ok {
		roles = append(roles, r...)
	}

	return roles
}

// HasRole checks if the user has a specific role in a scope.
func (ur *UserRoles) HasRole(role, scopeType, scopeID string) bool {
	roles := ur.GetRoles(scopeType, scopeID)
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// AuditAction represents the type of action in the audit log.
type AuditAction string

const (
	AuditActionAssigned AuditAction = "assigned"
	AuditActionRevoked  AuditAction = "revoked"
)

// AuditEntry is used to create new audit log entries.
type AuditEntry struct {
	ActorID       string
	Action        AuditAction
	TargetUserID  string
	Role          string
	ScopeType     string
	ScopeID       string
	ActorRoles    []string
	PreviousRoles []string
	NewRoles      []string
	IPAddress     string
	UserAgent     string
	RequestID     string
	Metadata      map[string]any
}

// ToModel converts an AuditEntry to a RoleAuditLog model.
func (e *AuditEntry) ToModel() *RoleAuditLog {
	return &RoleAuditLog{
		ActorID:       e.ActorID,
		Action:        string(e.Action),
		TargetUserID:  e.TargetUserID,
		Role:          e.Role,
		ScopeType:     e.ScopeType,
		ScopeID:       e.ScopeID,
		ActorRoles:    e.ActorRoles,
		PreviousRoles: e.PreviousRoles,
		NewRoles:      e.NewRoles,
		IPAddress:     e.IPAddress,
		UserAgent:     e.UserAgent,
		RequestID:     e.RequestID,
		Metadata:      e.Metadata,
		Timestamp:     time.Now(),
	}
}
