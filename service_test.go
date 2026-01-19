package rolekit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewService tests service construction.
func TestNewService(t *testing.T) {
	registry := NewRegistry()
	service := NewService(registry, nil)

	assert.NotNil(t, service)
	assert.Same(t, registry, service.Registry())
	assert.NotNil(t, service.txMonitor)
}

// TestServiceRegistryGetter tests the Registry accessor.
func TestServiceRegistryGetter(t *testing.T) {
	registry := NewRegistry()
	service := NewService(registry, nil)

	assert.Same(t, registry, service.Registry())
}

// TestServiceGetAuditLogNilDB verifies panic behavior when db is nil.
func TestServiceGetAuditLogNilDB(t *testing.T) {
	service := NewService(NewRegistry(), nil)
	ctx := context.Background()

	assert.Panics(t, func() {
		_, _ = service.GetAuditLog(ctx, NewAuditLogFilter())
	})
}

// TestServiceGetAuditLogFiltersNilDB checks filters still panic when db is nil.
func TestServiceGetAuditLogFiltersNilDB(t *testing.T) {
	service := NewService(NewRegistry(), nil)
	ctx := context.Background()

	filter := NewAuditLogFilter().
		WithActor("actor123").
		WithTargetUser("user456").
		WithScope("organization", "org123").
		WithAction(AuditActionAssigned).
		WithRole("admin").
		WithLimit(10).
		WithOffset(5)

	assert.Panics(t, func() {
		_, _ = service.GetAuditLog(ctx, filter)
	})
}
