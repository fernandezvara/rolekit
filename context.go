package rolekit

import (
	"context"
)

// Context keys for RoleKit values.
type contextKey string

const (
	contextKeyUserID    contextKey = "rolekit:user_id"
	contextKeyActorID   contextKey = "rolekit:actor_id"
	contextKeyIPAddress contextKey = "rolekit:ip_address"
	contextKeyUserAgent contextKey = "rolekit:user_agent"
	contextKeyRequestID contextKey = "rolekit:request_id"
	contextKeyChecker   contextKey = "rolekit:checker"
)

// WithUserID adds a user ID to the context.
// This is the user being checked for permissions.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, contextKeyUserID, userID)
}

// GetUserID retrieves the user ID from context.
// Returns empty string if not set.
func GetUserID(ctx context.Context) string {
	if v := ctx.Value(contextKeyUserID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// MustGetUserID retrieves the user ID from context.
// Panics if not set.
func MustGetUserID(ctx context.Context) string {
	userID := GetUserID(ctx)
	if userID == "" {
		panic("rolekit: user ID not in context")
	}
	return userID
}

// WithActorID adds an actor ID to the context.
// This is the user performing the action (for audit purposes).
// Often the same as user ID, but can be different for admin actions.
func WithActorID(ctx context.Context, actorID string) context.Context {
	return context.WithValue(ctx, contextKeyActorID, actorID)
}

// GetActorID retrieves the actor ID from context.
// Falls back to user ID if actor ID is not explicitly set.
func GetActorID(ctx context.Context) string {
	if v := ctx.Value(contextKeyActorID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	// Fallback to user ID
	return GetUserID(ctx)
}

// WithIPAddress adds the client IP address to the context (for audit).
func WithIPAddress(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, contextKeyIPAddress, ip)
}

// GetIPAddress retrieves the IP address from context.
func GetIPAddress(ctx context.Context) string {
	if v := ctx.Value(contextKeyIPAddress); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithUserAgent adds the user agent to the context (for audit).
func WithUserAgent(ctx context.Context, ua string) context.Context {
	return context.WithValue(ctx, contextKeyUserAgent, ua)
}

// GetUserAgent retrieves the user agent from context.
func GetUserAgent(ctx context.Context) string {
	if v := ctx.Value(contextKeyUserAgent); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithRequestID adds a request ID to the context (for audit and correlation).
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, contextKeyRequestID, requestID)
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(contextKeyRequestID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithChecker adds a Checker to the context.
// This is set by middleware and can be retrieved in handlers.
func WithChecker(ctx context.Context, checker *Checker) context.Context {
	return context.WithValue(ctx, contextKeyChecker, checker)
}

// GetChecker retrieves the Checker from context.
// Returns nil if not set.
func GetChecker(ctx context.Context) *Checker {
	if v := ctx.Value(contextKeyChecker); v != nil {
		if c, ok := v.(*Checker); ok {
			return c
		}
	}
	return nil
}

// FromContext retrieves the Checker from context.
// Alias for GetChecker for convenience.
func FromContext(ctx context.Context) *Checker {
	return GetChecker(ctx)
}

// AuditContext holds all audit-related information from context.
type AuditContext struct {
	ActorID   string
	IPAddress string
	UserAgent string
	RequestID string
}

// GetAuditContext extracts all audit information from context.
func GetAuditContext(ctx context.Context) AuditContext {
	return AuditContext{
		ActorID:   GetActorID(ctx),
		IPAddress: GetIPAddress(ctx),
		UserAgent: GetUserAgent(ctx),
		RequestID: GetRequestID(ctx),
	}
}

// WithAuditContext adds all audit information to context at once.
func WithAuditContext(ctx context.Context, ac AuditContext) context.Context {
	if ac.ActorID != "" {
		ctx = WithActorID(ctx, ac.ActorID)
	}
	if ac.IPAddress != "" {
		ctx = WithIPAddress(ctx, ac.IPAddress)
	}
	if ac.UserAgent != "" {
		ctx = WithUserAgent(ctx, ac.UserAgent)
	}
	if ac.RequestID != "" {
		ctx = WithRequestID(ctx, ac.RequestID)
	}
	return ctx
}
