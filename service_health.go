package rolekit

import (
	"context"

	"github.com/fernandezvara/dbkit"
)

// HealthService provides health monitoring functionality as an extension to Service
type HealthService struct {
	*Service
}

// NewHealthService creates a new health service extension
func NewHealthService(service *Service) *HealthService {
	return &HealthService{Service: service}
}

// Health performs a comprehensive health check of the database connection.
// Returns detailed status including latency, connection pool statistics, and error information.
func (hs *HealthService) Health(ctx context.Context) dbkit.HealthStatus {
	// Check if we have a DBKit instance
	if db, ok := hs.db.(*dbkit.DBKit); ok {
		return db.Health(ctx)
	}

	// If we're in a transaction or have a different type, do a basic ping
	return dbkit.HealthStatus{
		Healthy: hs.IsHealthy(ctx),
		Error:   "Limited health check - not a DBKit instance",
	}
}

// IsHealthy performs a simple health check of the database connection.
// Returns true if the database is reachable, false otherwise.
func (hs *HealthService) IsHealthy(ctx context.Context) bool {
	// Check if we have a DBKit instance
	if db, ok := hs.db.(*dbkit.DBKit); ok {
		return db.IsHealthy(ctx)
	}

	// If we're in a transaction or have a different type, try to ping
	var count int
	err := hs.db.NewSelect().Model((*struct{})(nil)).ColumnExpr("1").Limit(1).Scan(ctx, &count)
	return err == nil
}

// GetPoolStats returns connection pool statistics for monitoring.
// Returns zero values if the database instance doesn't support pool statistics.
func (hs *HealthService) GetPoolStats() dbkit.PoolStats {
	// Check if we have a DBKit instance
	if db, ok := hs.db.(*dbkit.DBKit); ok {
		sqlStats := db.Stats()
		return dbkit.PoolStatsFromSQL(sqlStats)
	}

	// Return zero values for non-DBKit instances
	return dbkit.PoolStats{}
}

// Ping performs a basic connectivity test to the database.
// Returns an error if the database is not reachable.
func (hs *HealthService) Ping(ctx context.Context) error {
	// Use a simple query to test connectivity
	var result int
	return hs.db.NewSelect().Model((*struct{})(nil)).ColumnExpr("1").Limit(1).Scan(ctx, &result)
}
