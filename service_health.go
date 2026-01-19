package rolekit

import (
	"context"

	"github.com/fernandezvara/dbkit"
)

// Health performs a comprehensive health check of the database connection.
// Returns detailed status including latency, connection pool statistics, and error information.
func (s *Service) Health(ctx context.Context) dbkit.HealthStatus {
	// Check if we have a DBKit instance
	if db, ok := s.db.(*dbkit.DBKit); ok {
		return db.Health(ctx)
	}

	// If we're in a transaction or have a different type, do a basic ping
	return dbkit.HealthStatus{
		Healthy: s.IsHealthy(ctx),
		Error:   "Limited health check - not a DBKit instance",
	}
}

// IsHealthy performs a simple health check of the database connection.
// Returns true if the database is reachable, false otherwise.
func (s *Service) IsHealthy(ctx context.Context) bool {
	// Check if we have a DBKit instance
	if db, ok := s.db.(*dbkit.DBKit); ok {
		return db.IsHealthy(ctx)
	}

	// If we're in a transaction or have a different type, try to ping
	var count int
	err := s.db.NewSelect().ColumnExpr("1").Limit(1).Scan(ctx, &count)
	return err == nil
}

// Ping performs a basic connectivity test to the database.
// Returns an error if the database is not reachable.
func (s *Service) Ping(ctx context.Context) error {
	// Use a simple query to test connectivity
	var result int
	return s.db.NewSelect().ColumnExpr("1").Limit(1).Scan(ctx, &result)
}
