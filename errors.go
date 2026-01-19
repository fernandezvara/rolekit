package rolekit

import (
	"errors"
	"fmt"
)

// Sentinel errors for RoleKit operations.
var (
	// ErrInvalidScope is returned when a scope type is not defined in the registry.
	ErrInvalidScope = errors.New("rolekit: invalid scope")

	// ErrInvalidRole is returned when a role is not defined for a scope.
	ErrInvalidRole = errors.New("rolekit: invalid role")

	// ErrInvalidPermission is returned when a permission format is invalid.
	ErrInvalidPermission = errors.New("rolekit: invalid permission")

	// ErrUnauthorized is returned when a user doesn't have the required role/permission.
	ErrUnauthorized = errors.New("rolekit: unauthorized")

	// ErrCannotAssign is returned when a user tries to assign a role they're not allowed to.
	ErrCannotAssign = errors.New("rolekit: cannot assign role")

	// ErrRoleAlreadyAssigned is returned when trying to assign a role the user already has.
	ErrRoleAlreadyAssigned = errors.New("rolekit: role already assigned")

	// ErrRoleNotAssigned is returned when trying to revoke a role the user doesn't have.
	ErrRoleNotAssigned = errors.New("rolekit: role not assigned")

	// ErrNoUserID is returned when user ID is not found in context.
	ErrNoUserID = errors.New("rolekit: no user ID in context")

	// ErrNoActorID is returned when actor ID is not found in context for audit.
	ErrNoActorID = errors.New("rolekit: no actor ID in context")

	// ErrDatabaseError is returned when a database operation fails.
	ErrDatabaseError = errors.New("rolekit: database error")
)

// Error wraps a sentinel error with additional context.
type Error struct {
	Err     error  // Underlying sentinel error
	Message string // Additional context
	Scope   string // Scope type involved
	ScopeID string // Scope ID involved
	Role    string // Role involved (if applicable)
	UserID  string // User involved (if applicable)
	ActorID string // Actor who triggered the error (if applicable)
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Err.Error(), e.Message)
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error for errors.Is/As.
func (e *Error) Unwrap() error {
	return e.Err
}

// Is checks if the error matches a target error.
func (e *Error) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// NewError creates a new Error with context.
func NewError(err error, message string) *Error {
	return &Error{
		Err:     err,
		Message: message,
	}
}

// WithScope adds scope information to the error.
func (e *Error) WithScope(scopeType, scopeID string) *Error {
	e.Scope = scopeType
	e.ScopeID = scopeID
	return e
}

// WithRole adds role information to the error.
func (e *Error) WithRole(role string) *Error {
	e.Role = role
	return e
}

// WithUser adds user information to the error.
func (e *Error) WithUser(userID string) *Error {
	e.UserID = userID
	return e
}

// WithActor adds actor information to the error.
func (e *Error) WithActor(actorID string) *Error {
	e.ActorID = actorID
	return e
}

// IsUnauthorized checks if an error is an authorization error.
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsInvalidScope checks if an error is due to an invalid scope.
func IsInvalidScope(err error) bool {
	return errors.Is(err, ErrInvalidScope)
}

// IsInvalidRole checks if an error is due to an invalid role.
func IsInvalidRole(err error) bool {
	return errors.Is(err, ErrInvalidRole)
}

// IsCannotAssign checks if an error is due to lacking assignment permission.
func IsCannotAssign(err error) bool {
	return errors.Is(err, ErrCannotAssign)
}
