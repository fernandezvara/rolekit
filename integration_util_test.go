package rolekit

import (
	"testing"
)

// TestDatabaseAvailabilityCheck tests the database availability checker
func TestDatabaseAvailabilityCheck(t *testing.T) {
	// This test should always run, even without database
	t.Run("Database availability check", func(t *testing.T) {
		// The function should work regardless of database availability
		// It should return true if database is available, false otherwise
		// We don't assert a specific value since it depends on environment
		_ = isDatabaseAvailable()
	})
}

// TestGetTestDatabaseURL tests the database URL helper
func TestGetTestDatabaseURL(t *testing.T) {
	// This test should always run
	_ = getTestDatabaseURL()
}
