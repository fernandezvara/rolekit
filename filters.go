package rolekit

import "time"

// AuditLogFilter provides options for filtering audit log queries.
type AuditLogFilter struct {
	// Filter by actor who performed the action
	ActorID string

	// Filter by target user of the action
	TargetUserID string

	// Filter by scope type
	ScopeType string

	// Filter by scope ID
	ScopeID string

	// Filter by action type ("assigned" or "revoked")
	Action string

	// Filter by role
	Role string

	// Filter by time range
	Since time.Time
	Until time.Time

	// Pagination
	Limit  int
	Offset int
}

// NewAuditLogFilter creates a new AuditLogFilter with default values.
func NewAuditLogFilter() AuditLogFilter {
	return AuditLogFilter{
		Limit: 100,
	}
}

// WithActor sets the actor ID filter.
func (f AuditLogFilter) WithActor(actorID string) AuditLogFilter {
	f.ActorID = actorID
	return f
}

// WithTargetUser sets the target user ID filter.
func (f AuditLogFilter) WithTargetUser(userID string) AuditLogFilter {
	f.TargetUserID = userID
	return f
}

// WithScope sets the scope filter.
func (f AuditLogFilter) WithScope(scopeType, scopeID string) AuditLogFilter {
	f.ScopeType = scopeType
	f.ScopeID = scopeID
	return f
}

// WithScopeType sets only the scope type filter.
func (f AuditLogFilter) WithScopeType(scopeType string) AuditLogFilter {
	f.ScopeType = scopeType
	return f
}

// WithAction sets the action filter.
func (f AuditLogFilter) WithAction(action AuditAction) AuditLogFilter {
	f.Action = string(action)
	return f
}

// WithRole sets the role filter.
func (f AuditLogFilter) WithRole(role string) AuditLogFilter {
	f.Role = role
	return f
}

// WithTimeRange sets the time range filter.
func (f AuditLogFilter) WithTimeRange(since, until time.Time) AuditLogFilter {
	f.Since = since
	f.Until = until
	return f
}

// WithSince sets the start time filter.
func (f AuditLogFilter) WithSince(since time.Time) AuditLogFilter {
	f.Since = since
	return f
}

// WithUntil sets the end time filter.
func (f AuditLogFilter) WithUntil(until time.Time) AuditLogFilter {
	f.Until = until
	return f
}

// WithLimit sets the limit for results.
func (f AuditLogFilter) WithLimit(limit int) AuditLogFilter {
	f.Limit = limit
	return f
}

// WithOffset sets the offset for pagination.
func (f AuditLogFilter) WithOffset(offset int) AuditLogFilter {
	f.Offset = offset
	return f
}

// WithPagination sets both limit and offset.
func (f AuditLogFilter) WithPagination(limit, offset int) AuditLogFilter {
	f.Limit = limit
	f.Offset = offset
	return f
}
