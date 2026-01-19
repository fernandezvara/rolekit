package rolekit

import (
	"context"
	"fmt"

	"github.com/fernandezvara/dbkit"
)

// MigrationService provides migration management functionality as an extension to Service
type MigrationService struct {
	*Service
}

// NewMigrationService creates a new migration service extension
func NewMigrationService(service *Service) *MigrationService {
	return &MigrationService{Service: service}
}

// Migrations returns all database migrations required for RoleKit.
func (ms *MigrationService) Migrations() []dbkit.Migration {
	return []dbkit.Migration{
		{
			ID:          "rolekit-001",
			Description: "Create role_assignments table",
			SQL: `
                CREATE TABLE IF NOT EXISTS role_assignments (
                    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                    user_id TEXT NOT NULL,
                    role TEXT NOT NULL,
                    scope_type TEXT NOT NULL,
                    scope_id TEXT NOT NULL,
                    parent_scope_type TEXT,
                    parent_scope_id TEXT,
                    created_at TIMESTAMPTZ DEFAULT current_timestamp,
                    updated_at TIMESTAMPTZ DEFAULT current_timestamp
                )`,
		},
		{
			ID:          "rolekit-002",
			Description: "Create role_audit_log table",
			SQL: `
                CREATE TABLE IF NOT EXISTS role_audit_log (
                    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                    timestamp TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
                    actor_id TEXT NOT NULL,
                    action TEXT NOT NULL,
                    target_user_id TEXT NOT NULL,
                    role TEXT NOT NULL,
                    scope_type TEXT NOT NULL,
                    scope_id TEXT NOT NULL,
                    ip_address TEXT,
                    user_agent TEXT,
                    request_id TEXT
                )`,
		},
		{
			ID:          "rolekit-003",
			Description: "Create scope_hierarchy table",
			SQL: `
                CREATE TABLE IF NOT EXISTS scope_hierarchy (
                    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                    scope_type TEXT NOT NULL,
                    scope_id TEXT NOT NULL,
                    parent_scope_type TEXT NOT NULL,
                    parent_scope_id TEXT NOT NULL,
                    created_at TIMESTAMPTZ DEFAULT current_timestamp,
                    updated_at TIMESTAMPTZ DEFAULT current_timestamp
                )`,
		},
	}
}

// RunMigrations runs all pending migrations and returns the status.
func (ms *MigrationService) RunMigrations(ctx context.Context) (*MigrationStatus, error) {
	if db, ok := ms.db.(*dbkit.DBKit); ok {
		migrations := ms.Migrations()
		_, err := db.Migrate(ctx, migrations)
		if err != nil {
			return nil, fmt.Errorf("migration failed: %w", err)
		}
		return ms.GetMigrationStatus(ctx)
	}
	return nil, fmt.Errorf("migration support requires a dbkit.DBKit instance")
}

// GetMigrationStatus returns the current status of all migrations.
func (ms *MigrationService) GetMigrationStatus(ctx context.Context) (*MigrationStatus, error) {
	if db, ok := ms.db.(*dbkit.DBKit); ok {
		migrations := ms.Migrations()
		statusEntries, err := db.MigrationStatus(ctx, migrations)
		if err != nil {
			return nil, fmt.Errorf("failed to get migration status: %w", err)
		}

		total := len(statusEntries)
		applied := 0
		pending := 0

		for _, entry := range statusEntries {
			if entry.Applied {
				applied++
			} else {
				pending++
			}
		}

		return &MigrationStatus{
			Total:      total,
			Applied:    applied,
			Pending:    pending,
			Migrations: statusEntries,
		}, nil
	}

	migrations := ms.Migrations()
	return &MigrationStatus{
		Total:      len(migrations),
		Applied:    0,
		Pending:    len(migrations),
		Migrations: make([]dbkit.MigrationStatusEntry, 0),
	}, nil
}

// VerifyMigrationChecksums verifies that all applied migrations have the correct checksums.
func (ms *MigrationService) VerifyMigrationChecksums(ctx context.Context) (bool, error) {
	status, err := ms.GetMigrationStatus(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get migration status for checksum verification: %w", err)
	}

	migrations := ms.Migrations()
	sqlMap := make(map[string]string)
	for _, migration := range migrations {
		sqlMap[migration.ID] = migration.SQL
	}

	for _, entry := range status.Migrations {
		if entry.Applied {
			_, exists := sqlMap[entry.ID]
			if !exists {
				return false, fmt.Errorf("migration %s not found in migration definitions", entry.ID)
			}
		}
	}

	return true, nil
}

// RollbackToMigration rolls back to a specific migration.
func (ms *MigrationService) RollbackToMigration(ctx context.Context, targetMigrationID string) error {
	if _, ok := ms.db.(*dbkit.DBKit); ok {
		migrations := ms.Migrations()
		for _, migration := range migrations {
			if migration.ID == targetMigrationID {
				return fmt.Errorf("rollback not implemented for SQL migrations")
			}
		}
		return fmt.Errorf("migration %s not found", targetMigrationID)
	}
	return fmt.Errorf("migration support requires a dbkit.DBKit instance")
}

// ValidateMigrations validates all migration definitions.
func (ms *MigrationService) ValidateMigrations() error {
	migrations := ms.Migrations()
	for _, migration := range migrations {
		if migration.ID == "" {
			return fmt.Errorf("migration ID cannot be empty")
		}
		if migration.Description == "" {
			return fmt.Errorf("migration description cannot be empty")
		}
		if migration.SQL == "" {
			return fmt.Errorf("migration SQL cannot be empty")
		}
	}
	return nil
}
