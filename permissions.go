package rolekit

import (
	"strings"
)

// PermissionMatcher handles permission matching with wildcard support.
//
// Supported patterns:
//   - "*" matches all permissions
//   - "resource.*" matches all actions on a resource (e.g., "files.*" matches "files.read")
//   - "*.action" matches an action on all resources (e.g., "*.read" matches "files.read")
//   - "exact.match" matches exactly
type PermissionMatcher struct{}

// NewPermissionMatcher creates a new PermissionMatcher.
func NewPermissionMatcher() *PermissionMatcher {
	return &PermissionMatcher{}
}

// Match checks if a permission pattern matches a required permission.
//
// Examples:
//
//	Match("*", "files.read")           // true - wildcard matches all
//	Match("files.*", "files.read")     // true - resource wildcard
//	Match("files.*", "files.write")    // true - resource wildcard
//	Match("*.read", "files.read")      // true - action wildcard
//	Match("*.read", "members.read")    // true - action wildcard
//	Match("files.read", "files.read")  // true - exact match
//	Match("files.read", "files.write") // false - no match
//	Match("files.*", "members.read")   // false - different resource
func (pm *PermissionMatcher) Match(pattern, permission string) bool {
	// Exact match
	if pattern == permission {
		return true
	}

	// Universal wildcard
	if pattern == "*" {
		return true
	}

	// Split into parts
	patternParts := strings.Split(pattern, ".")
	permParts := strings.Split(permission, ".")

	// Must have same number of parts (or pattern is just "*")
	if len(patternParts) != len(permParts) {
		return false
	}

	// Match each part
	for i, pp := range patternParts {
		if pp == "*" {
			// Wildcard matches anything
			continue
		}
		if pp != permParts[i] {
			return false
		}
	}

	return true
}

// MatchAny checks if any of the patterns match the required permission.
func (pm *PermissionMatcher) MatchAny(patterns []string, permission string) bool {
	for _, pattern := range patterns {
		if pm.Match(pattern, permission) {
			return true
		}
	}
	return false
}

// ExpandPermissions returns all permissions that a set of patterns would grant.
// This is useful for displaying what a role can do.
// Note: This only works for known permissions passed in the 'all' slice.
func (pm *PermissionMatcher) ExpandPermissions(patterns []string, all []string) []string {
	matched := make(map[string]bool)

	for _, permission := range all {
		for _, pattern := range patterns {
			if pm.Match(pattern, permission) {
				matched[permission] = true
				break
			}
		}
	}

	result := make([]string, 0, len(matched))
	for p := range matched {
		result = append(result, p)
	}
	return result
}

// Validate checks if a permission string is valid.
// A valid permission is either "*" or a dot-separated string of identifiers.
func (pm *PermissionMatcher) Validate(permission string) error {
	if permission == "" {
		return NewError(ErrInvalidPermission, "permission cannot be empty")
	}

	if permission == "*" {
		return nil
	}

	parts := strings.Split(permission, ".")
	if len(parts) < 2 {
		return NewError(ErrInvalidPermission, "permission must have at least two parts (resource.action)")
	}

	for _, part := range parts {
		if part == "" {
			return NewError(ErrInvalidPermission, "permission parts cannot be empty")
		}
		// Allow * as a part
		if part == "*" {
			continue
		}
		// Check for valid identifier characters (alphanumeric and underscore)
		for _, c := range part {
			if !isValidPermissionChar(c) {
				return NewError(ErrInvalidPermission, "permission contains invalid character")
			}
		}
	}

	return nil
}

func isValidPermissionChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

// DefaultMatcher is the default permission matcher instance.
var DefaultMatcher = NewPermissionMatcher()

// MatchPermission is a convenience function using the default matcher.
func MatchPermission(pattern, permission string) bool {
	return DefaultMatcher.Match(pattern, permission)
}

// MatchAnyPermission is a convenience function using the default matcher.
func MatchAnyPermission(patterns []string, permission string) bool {
	return DefaultMatcher.MatchAny(patterns, permission)
}
