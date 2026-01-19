package rolekit

import (
	"testing"
)

// TestDatabaseAvailabilityCheck tests the database availability checker
func TestDatabaseAvailabilityCheck(t *testing.T) {
	// This test should always run, even without database
	t.Run("Database not available", func(t *testing.T) {
		// When database is not available, isDatabaseAvailable should return false
		if isDatabaseAvailable() {
			t.Error("Expected database to be unavailable")
		}
	})

	t.Run("RequireDatabase skips test", func(t *testing.T) {
		// When database is not available, requireDatabase should skip the test
		if requireDatabase(t) {
			t.Error("Expected requireDatabase to skip test when database unavailable")
		}
	})
}

// TestGetTestDatabaseURL tests the database URL helper
func TestGetTestDatabaseURL(t *testing.T) {
	// This test should always run
	url := getTestDatabaseURL()
	expected := "postgres://postgres:password@localhost:5418/rolekit_test?sslmode=disable"

	if url != expected {
		t.Errorf("Expected URL %s, got %s", expected, url)
	}
}
