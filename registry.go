package rolekit

import (
	"fmt"
	"sync"
)

// Registry holds all scope and role definitions for the application.
// It is created at startup and should be treated as immutable after initialization.
type Registry struct {
	mu     sync.RWMutex
	scopes map[string]*ScopeDefinition
}

// ScopeDefinition defines a scope type (e.g., "organization", "project")
// and all roles available within that scope.
type ScopeDefinition struct {
	name        string
	parentScope string // Optional parent scope for hierarchy
	roles       map[string]*RoleDefinition
	registry    *Registry
}

// RoleDefinition defines a role within a scope, including its permissions
// and which roles it can assign to others.
type RoleDefinition struct {
	name           string
	scopeName      string
	permissions    []string // Permissions this role grants
	canAssignRoles []string // Roles this role can assign to others
	scope          *ScopeDefinition
}

// NewRegistry creates a new role registry.
func NewRegistry() *Registry {
	return &Registry{
		scopes: make(map[string]*ScopeDefinition),
	}
}

// DefineScope starts defining a new scope type.
// Returns a ScopeDefinition builder for fluent configuration.
//
// Example:
//
//	registry.DefineScope("organization").
//	    Role("owner").Permissions("*").CanAssign("*").
//	    Role("admin").Permissions("members.*").CanAssign("member")
func (r *Registry) DefineScope(name string) *ScopeDefinition {
	r.mu.Lock()
	defer r.mu.Unlock()

	scope := &ScopeDefinition{
		name:     name,
		roles:    make(map[string]*RoleDefinition),
		registry: r,
	}
	r.scopes[name] = scope
	return scope
}

// GetScope returns the scope definition for a scope type.
// Returns nil if the scope is not defined.
func (r *Registry) GetScope(name string) *ScopeDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.scopes[name]
}

// GetScopes returns all defined scope names.
func (r *Registry) GetScopes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.scopes))
	for name := range r.scopes {
		names = append(names, name)
	}
	return names
}

// ValidateScope checks if a scope type is defined.
func (r *Registry) ValidateScope(scopeType string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.scopes[scopeType]; !exists {
		return fmt.Errorf("%w: scope type %q not defined", ErrInvalidScope, scopeType)
	}
	return nil
}

// ValidateRole checks if a role is defined for a scope type.
func (r *Registry) ValidateRole(role, scopeType string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	scope, exists := r.scopes[scopeType]
	if !exists {
		return fmt.Errorf("%w: scope type %q not defined", ErrInvalidScope, scopeType)
	}

	if _, exists := scope.roles[role]; !exists {
		return fmt.Errorf("%w: role %q not defined for scope %q", ErrInvalidRole, role, scopeType)
	}
	return nil
}

// GetRole returns the role definition for a role in a scope.
func (r *Registry) GetRole(role, scopeType string) *RoleDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	scope, exists := r.scopes[scopeType]
	if !exists {
		return nil
	}
	return scope.roles[role]
}

// GetPermissions returns all permissions for a role in a scope.
func (r *Registry) GetPermissions(role, scopeType string) []string {
	roleDef := r.GetRole(role, scopeType)
	if roleDef == nil {
		return nil
	}
	return roleDef.permissions
}

// CanRoleAssign checks if a role can assign another role in the same scope.
func (r *Registry) CanRoleAssign(assignerRole, targetRole, scopeType string) bool {
	roleDef := r.GetRole(assignerRole, scopeType)
	if roleDef == nil {
		return false
	}

	for _, allowed := range roleDef.canAssignRoles {
		if allowed == "*" || allowed == targetRole {
			return true
		}
	}
	return false
}

// ParentScope sets the parent scope type for hierarchical queries.
// This creates awareness but does NOT grant automatic access.
//
// Example:
//
//	registry.DefineScope("project").ParentScope("organization")
//
// This allows queries like "get all projects in org where user has role X"
func (s *ScopeDefinition) ParentScope(parentName string) *ScopeDefinition {
	s.parentScope = parentName
	return s
}

// GetParentScope returns the parent scope name, or empty string if none.
func (s *ScopeDefinition) GetParentScope() string {
	return s.parentScope
}

// Role starts defining a new role within this scope.
// Returns a RoleDefinition builder for fluent configuration.
//
// Example:
//
//	scope.Role("admin").
//	    Permissions("members.*", "settings.*").
//	    CanAssign("member", "viewer")
func (s *ScopeDefinition) Role(name string) *RoleDefinition {
	role := &RoleDefinition{
		name:      name,
		scopeName: s.name,
		scope:     s,
	}
	s.roles[name] = role
	return role
}

// GetRole returns a role definition by name within this scope.
func (s *ScopeDefinition) GetRole(name string) *RoleDefinition {
	return s.roles[name]
}

// GetRoles returns all role names defined in this scope.
func (s *ScopeDefinition) GetRoles() []string {
	names := make([]string, 0, len(s.roles))
	for name := range s.roles {
		names = append(names, name)
	}
	return names
}

// Name returns the scope name.
func (s *ScopeDefinition) Name() string {
	return s.name
}

// Permissions sets the permissions granted by this role.
// Supports wildcards: "*", "resource.*", "*.action"
//
// Example:
//
//	role.Permissions("files.read", "files.write", "comments.*")
func (r *RoleDefinition) Permissions(perms ...string) *RoleDefinition {
	r.permissions = append(r.permissions, perms...)
	return r
}

// CanAssign sets which roles this role can assign to other users.
// Use "*" to allow assigning any role.
//
// Example:
//
//	role.CanAssign("member", "viewer")  // Can assign member or viewer
//	role.CanAssign("*")                  // Can assign any role
func (r *RoleDefinition) CanAssign(roles ...string) *RoleDefinition {
	r.canAssignRoles = append(r.canAssignRoles, roles...)
	return r
}

// Role continues defining roles in the parent scope (fluent API).
// This allows chaining role definitions.
//
// Example:
//
//	scope.Role("admin").Permissions("*").
//	    Role("member").Permissions("read")  // Continues on scope
func (r *RoleDefinition) Role(name string) *RoleDefinition {
	return r.scope.Role(name)
}

// DefineScope continues defining scopes on the registry (fluent API).
// This allows chaining scope definitions.
func (r *RoleDefinition) DefineScope(name string) *ScopeDefinition {
	return r.scope.registry.DefineScope(name)
}

// GetPermissions returns the permissions for this role.
func (r *RoleDefinition) GetPermissions() []string {
	return r.permissions
}

// GetCanAssign returns the roles this role can assign.
func (r *RoleDefinition) GetCanAssign() []string {
	return r.canAssignRoles
}

// Name returns the role name.
func (r *RoleDefinition) Name() string {
	return r.name
}

// ScopeName returns the scope this role belongs to.
func (r *RoleDefinition) ScopeName() string {
	return r.scopeName
}
