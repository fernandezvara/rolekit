package rolekit

import (
	"testing"
)

// TestMigrations tests that migrations are properly defined
func TestMigrations(t *testing.T) {
	service := &Service{}
	migrations := service.Migrations()

	if len(migrations) == 0 {
		t.Error("Expected at least one migration")
	}

	for _, m := range migrations {
		if m.ID == "" {
			t.Error("Migration ID should not be empty")
		}
		if m.Description == "" {
			t.Error("Migration description should not be empty")
		}
		if m.SQL == "" {
			t.Error("Migration SQL should not be empty")
		}
	}
}
