package rolekit

import (
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
// Use dbkit.Migrate(ctx, service.Migrations()) to run migrations.
// Use dbkit.MigrationStatus(ctx, service.Migrations()) to check status.
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
