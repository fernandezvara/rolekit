package rolekit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPermissionMatcherNewPermissionMatcher tests the matcher constructor
func TestPermissionMatcherNewPermissionMatcher(t *testing.T) {
	matcher := NewPermissionMatcher()
	assert.NotNil(t, matcher)
}

// TestPermissionMatcherMatch tests permission pattern matching
func TestPermissionMatcherMatch(t *testing.T) {
	matcher := NewPermissionMatcher()

	tests := []struct {
		name       string
		pattern    string
		permission string
		expected   bool
	}{
		// Exact matches
		{
			name:       "Exact match",
			pattern:    "files.read",
			permission: "files.read",
			expected:   true,
		},
		{
			name:       "Exact match different",
			pattern:    "files.read",
			permission: "files.write",
			expected:   false,
		},

		// Universal wildcard
		{
			name:       "Universal wildcard matches all",
			pattern:    "*",
			permission: "files.read",
			expected:   true,
		},
		{
			name:       "Universal wildcard matches complex",
			pattern:    "*",
			permission: "organization.users.create",
			expected:   true,
		},

		// Resource wildcard
		{
			name:       "Resource wildcard matches read",
			pattern:    "files.*",
			permission: "files.read",
			expected:   true,
		},
		{
			name:       "Resource wildcard matches write",
			pattern:    "files.*",
			permission: "files.write",
			expected:   true,
		},
		{
			name:       "Resource wildcard matches delete",
			pattern:    "files.*",
			permission: "files.delete",
			expected:   true,
		},
		{
			name:       "Resource wildcard no match different resource",
			pattern:    "files.*",
			permission: "users.read",
			expected:   false,
		},

		// Action wildcard
		{
			name:       "Action wildcard matches files",
			pattern:    "*.read",
			permission: "files.read",
			expected:   true,
		},
		{
			name:       "Action wildcard matches users",
			pattern:    "*.read",
			permission: "users.read",
			expected:   true,
		},
		{
			name:       "Action wildcard no match different action",
			pattern:    "*.read",
			permission: "files.write",
			expected:   false,
		},

		// Mixed wildcards
		{
			name:       "Mixed wildcard first part",
			pattern:    "*.users.*",
			permission: "admin.users.create",
			expected:   true,
		},
		{
			name:       "Mixed wildcard middle part",
			pattern:    "files.*.private",
			permission: "files.read.private",
			expected:   true,
		},
		{
			name:       "Mixed wildcard last part",
			pattern:    "files.public.*",
			permission: "files.public.read",
			expected:   true,
		},

		// Complex permissions
		{
			name:       "Complex permission exact match",
			pattern:    "organization.users.invite",
			permission: "organization.users.invite",
			expected:   true,
		},
		{
			name:       "Complex permission resource wildcard",
			pattern:    "organization.users.*",
			permission: "organization.users.invite",
			expected:   true,
		},
		{
			name:       "Complex permission action wildcard",
			pattern:    "*.users.invite",
			permission: "organization.users.invite",
			expected:   true,
		},

		// Edge cases
		{
			name:       "Different number of parts",
			pattern:    "files.read",
			permission: "files.read.write",
			expected:   false,
		},
		{
			name:       "Pattern has more parts",
			pattern:    "files.read.write",
			permission: "files.read",
			expected:   false,
		},
		{
			name:       "Single part pattern",
			pattern:    "files",
			permission: "files.read",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(tt.pattern, tt.permission)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestPermissionMatcherMatchAny tests matching any of multiple patterns
func TestPermissionMatcherMatchAny(t *testing.T) {
	matcher := NewPermissionMatcher()

	tests := []struct {
		name       string
		patterns   []string
		permission string
		expected   bool
	}{
		{
			name:       "One pattern matches",
			patterns:   []string{"files.read", "users.write"},
			permission: "files.read",
			expected:   true,
		},
		{
			name:       "Second pattern matches",
			patterns:   []string{"files.read", "users.write"},
			permission: "users.write",
			expected:   true,
		},
		{
			name:       "No pattern matches",
			patterns:   []string{"files.read", "users.write"},
			permission: "files.delete",
			expected:   false,
		},
		{
			name:       "Universal wildcard in patterns",
			patterns:   []string{"files.read", "*"},
			permission: "anything.goes",
			expected:   true,
		},
		{
			name:       "Empty pattern list",
			patterns:   []string{},
			permission: "files.read",
			expected:   false,
		},
		{
			name:       "Multiple wildcards",
			patterns:   []string{"files.*", "*.read"},
			permission: "files.read",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.MatchAny(tt.patterns, tt.permission)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestPermissionMatcherExpandPermissions tests expanding patterns to known permissions
func TestPermissionMatcherExpandPermissions(t *testing.T) {
	matcher := NewPermissionMatcher()

	allPermissions := []string{
		"files.read",
		"files.write",
		"files.delete",
		"users.read",
		"users.write",
		"users.delete",
		"admin.users.invite",
		"admin.users.remove",
	}

	tests := []struct {
		name     string
		patterns []string
		expected []string
	}{
		{
			name:     "Exact pattern",
			patterns: []string{"files.read"},
			expected: []string{"files.read"},
		},
		{
			name:     "Resource wildcard",
			patterns: []string{"files.*"},
			expected: []string{"files.read", "files.write", "files.delete"},
		},
		{
			name:     "Action wildcard",
			patterns: []string{"*.read"},
			expected: []string{"files.read", "users.read"},
		},
		{
			name:     "Universal wildcard",
			patterns: []string{"*"},
			expected: allPermissions,
		},
		{
			name:     "Multiple patterns",
			patterns: []string{"files.read", "users.*"},
			expected: []string{"files.read", "users.read", "users.write", "users.delete"},
		},
		{
			name:     "Overlapping patterns",
			patterns: []string{"files.*", "*.read"},
			expected: []string{"files.read", "files.write", "files.delete", "users.read"},
		},
		{
			name:     "No matches",
			patterns: []string{"files.create"},
			expected: []string{},
		},
		{
			name:     "Empty patterns",
			patterns: []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ExpandPermissions(tt.patterns, allPermissions)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

// TestPermissionMatcherValidatePermission tests permission validation
func TestPermissionMatcherValidatePermission(t *testing.T) {
	matcher := NewPermissionMatcher()

	tests := []struct {
		name        string
		permission  string
		expectError bool
		errorMsg    string
	}{
		// Valid permissions
		{
			name:        "Universal wildcard",
			permission:  "*",
			expectError: false,
		},
		{
			name:        "Simple permission",
			permission:  "files.read",
			expectError: false,
		},
		{
			name:        "Complex permission",
			permission:  "organization.users.invite",
			expectError: false,
		},
		{
			name:        "Permission with underscore",
			permission:  "files.read_private",
			expectError: false,
		},
		{
			name:        "Permission with numbers",
			permission:  "api.v1.read",
			expectError: false,
		},
		{
			name:        "Permission with uppercase",
			permission:  "Files.Read",
			expectError: false,
		},
		{
			name:        "Permission with wildcard part",
			permission:  "files.*.private",
			expectError: false,
		},

		// Invalid permissions
		{
			name:        "Empty permission",
			permission:  "",
			expectError: true,
			errorMsg:    "permission cannot be empty",
		},
		{
			name:        "Single part",
			permission:  "files",
			expectError: true,
			errorMsg:    "permission must have at least two parts",
		},
		{
			name:        "Empty part in middle",
			permission:  "files..read",
			expectError: true,
			errorMsg:    "permission parts cannot be empty",
		},
		{
			name:        "Empty part at start",
			permission:  ".files.read",
			expectError: true,
			errorMsg:    "permission parts cannot be empty",
		},
		{
			name:        "Empty part at end",
			permission:  "files.read.",
			expectError: true,
			errorMsg:    "permission parts cannot be empty",
		},
		{
			name:        "Invalid character - space",
			permission:  "files. read",
			expectError: true,
			errorMsg:    "permission contains invalid character",
		},
		{
			name:        "Invalid character - dash",
			permission:  "files.-read",
			expectError: true,
			errorMsg:    "permission contains invalid character",
		},
		{
			name:        "Invalid character - special",
			permission:  "files.@read",
			expectError: true,
			errorMsg:    "permission contains invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := matcher.Validate(tt.permission)

			if tt.expectError {
				assert.Error(t, err)
				assert.IsType(t, &Error{}, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestIsValidPermissionChar tests the character validation helper
func TestIsValidPermissionChar(t *testing.T) {
	tests := []struct {
		char     rune
		expected bool
	}{
		// Valid lowercase
		{'a', true},
		{'z', true},
		// Valid uppercase
		{'A', true},
		{'Z', true},
		// Valid numbers
		{'0', true},
		{'9', true},
		// Valid underscore
		{'_', true},
		// Invalid characters
		{'-', false},
		{' ', false},
		{'@', false},
		{'#', false},
		{'.', false},
		{'/', false},
		{'\\', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			result := isValidPermissionChar(tt.char)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDefaultMatcher tests the default matcher instance
func TestDefaultMatcher(t *testing.T) {
	assert.NotNil(t, DefaultMatcher)

	// Test that it works like any other matcher
	assert.True(t, DefaultMatcher.Match("*", "files.read"))
	assert.True(t, DefaultMatcher.Match("files.*", "files.read"))
	assert.False(t, DefaultMatcher.Match("files.read", "files.write"))
}

// TestMatchPermission tests the convenience function
func TestMatchPermission(t *testing.T) {
	// Test that it uses the default matcher
	assert.True(t, MatchPermission("*", "files.read"))
	assert.True(t, MatchPermission("files.*", "files.read"))
	assert.False(t, MatchPermission("files.read", "files.write"))
}

// TestMatchAnyPermissionFunc tests the convenience function
func TestMatchAnyPermissionFunc(t *testing.T) {
	// Test that it uses the default matcher
	assert.True(t, MatchAnyPermission([]string{"files.read", "users.write"}, "files.read"))
	assert.True(t, MatchAnyPermission([]string{"files.read", "users.write"}, "users.write"))
	assert.False(t, MatchAnyPermission([]string{"files.read", "users.write"}, "files.delete"))
	assert.False(t, MatchAnyPermission([]string{}, "files.read"))
}

// TestPermissionEdgeCases tests edge cases and complex scenarios
func TestPermissionEdgeCases(t *testing.T) {
	matcher := NewPermissionMatcher()

	t.Run("Deeply nested permissions", func(t *testing.T) {
		pattern := "a.b.c.d.e"
		permission := "a.b.c.d.e"
		assert.True(t, matcher.Match(pattern, permission))

		permission2 := "a.b.c.d.x"
		assert.False(t, matcher.Match(pattern, permission2))
	})

	t.Run("Wildcards in complex positions", func(t *testing.T) {
		// Mixed wildcards
		assert.True(t, matcher.Match("a.*.c.*", "a.b.c.d"))
		assert.True(t, matcher.Match("a.*.c.*", "a.x.c.y"))
		assert.False(t, matcher.Match("a.*.c.*", "a.b.x.d"))

		// Multiple wildcards
		assert.True(t, matcher.Match("*.*.*", "a.b.c"))
		assert.True(t, matcher.Match("*.*.*", "x.y.z"))
		assert.False(t, matcher.Match("*.*.*", "a.b"))
	})

	t.Run("Permission validation edge cases", func(t *testing.T) {
		// Valid edge cases
		assert.NoError(t, matcher.Validate("a.b"))
		assert.NoError(t, matcher.Validate("A_1.B_2"))

		// Invalid edge cases
		assert.Error(t, matcher.Validate("a"))
		assert.Error(t, matcher.Validate("a..b"))
		assert.Error(t, matcher.Validate("a.b."))
		assert.Error(t, matcher.Validate(".a.b"))
	})

	t.Run("Expand with empty all permissions", func(t *testing.T) {
		result := matcher.ExpandPermissions([]string{"*"}, []string{})
		assert.Empty(t, result)

		result2 := matcher.ExpandPermissions([]string{"files.read"}, []string{})
		assert.Empty(t, result2)
	})
}
