package rolekit

import (
	"fmt"
	"log"
	"time"

	"github.com/fernandezvara/dbkit"
)

// ============================================================================
// CONNECTION POOL MANAGEMENT
// ============================================================================

// PoolConfig represents connection pool configuration settings.
type PoolConfig struct {
	// MaxOpenConnections is the maximum number of open connections to the database.
	// If MaxOpenConnections is 0, there is no limit on the number of open connections.
	MaxOpenConnections int `json:"max_open_connections"`

	// MaxIdleConnections is the maximum number of connections in the idle connection pool.
	// If MaxIdleConnections is 0, no idle connections are retained.
	MaxIdleConnections int `json:"max_idle_connections"`

	// ConnectionMaxLifetime is the maximum amount of time a connection may be reused.
	// If ConnectionMaxLifetime is 0, connections are reused forever.
	ConnectionMaxLifetime time.Duration `json:"connection_max_lifetime"`

	// ConnectionMaxIdleTime is the maximum amount of time a connection may be idle.
	// If ConnectionMaxIdleTime is 0, connections are not closed based on idle time.
	ConnectionMaxIdleTime time.Duration `json:"connection_max_idle_time"`
}

// DefaultPoolConfig returns sensible default connection pool settings.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConnections:    25,
		MaxIdleConnections:    25,
		ConnectionMaxLifetime: time.Hour,
		ConnectionMaxIdleTime: 5 * time.Minute,
	}
}

// HighPerformancePoolConfig returns optimized settings for high-performance workloads.
func HighPerformancePoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConnections:    100,
		MaxIdleConnections:    50,
		ConnectionMaxLifetime: 30 * time.Minute,
		ConnectionMaxIdleTime: 1 * time.Minute,
	}
}

// LowResourcePoolConfig returns optimized settings for resource-constrained environments.
func LowResourcePoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConnections:    5,
		MaxIdleConnections:    2,
		ConnectionMaxLifetime: 2 * time.Hour,
		ConnectionMaxIdleTime: 10 * time.Minute,
	}
}

// ConfigureConnectionPool updates the database connection pool settings.
func (s *Service) ConfigureConnectionPool(config PoolConfig) error {
	if db, ok := s.db.(*dbkit.DBKit); ok {
		bunDB := db.Bun()
		if bunDB == nil {
			return fmt.Errorf("database instance not available")
		}

		bunDB.SetMaxOpenConns(config.MaxOpenConnections)
		bunDB.SetMaxIdleConns(config.MaxIdleConnections)
		bunDB.SetConnMaxLifetime(config.ConnectionMaxLifetime)
		bunDB.SetConnMaxIdleTime(config.ConnectionMaxIdleTime)

		log.Printf("Connection pool configured: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v, MaxIdleTime=%v",
			config.MaxOpenConnections, config.MaxIdleConnections,
			config.ConnectionMaxLifetime, config.ConnectionMaxIdleTime)

		return nil
	}

	return fmt.Errorf("connection pool configuration requires a dbkit.DBKit instance")
}

// GetConnectionPoolConfig returns the current connection pool configuration.
func (s *Service) GetConnectionPoolConfig() (*PoolConfig, error) {
	if db, ok := s.db.(*dbkit.DBKit); ok {
		bunDB := db.Bun()
		if bunDB == nil {
			return nil, fmt.Errorf("database instance not available")
		}

		stats := bunDB.Stats()
		return &PoolConfig{
			MaxOpenConnections: stats.MaxOpenConnections,
			MaxIdleConnections: stats.MaxOpenConnections,
		}, nil
	}

	return nil, fmt.Errorf("connection pool configuration requires a dbkit.DBKit instance")
}

// OptimizeConnectionPool automatically adjusts pool settings based on current usage.
func (s *Service) OptimizeConnectionPool() error {
	stats := s.GetPoolStats()

	config, err := s.GetConnectionPoolConfig()
	if err != nil {
		return fmt.Errorf("failed to get current pool config: %w", err)
	}

	newConfig := *config

	// If we're using most of our connections, increase the pool
	if stats.InUse > 0 && float64(stats.InUse)/float64(stats.MaxOpenConnections) > 0.8 {
		newConfig.MaxOpenConnections = int(float64(config.MaxOpenConnections) * 1.5)
		newConfig.MaxIdleConnections = int(float64(config.MaxIdleConnections) * 1.5)
	}

	// If we have many idle connections, reduce the pool
	if stats.Idle > 0 && float64(stats.Idle)/float64(stats.MaxOpenConnections) > 0.8 {
		newConfig.MaxOpenConnections = int(float64(config.MaxOpenConnections) * 0.75)
		newConfig.MaxIdleConnections = int(float64(config.MaxIdleConnections) * 0.75)
	}

	// Ensure minimum values
	if newConfig.MaxOpenConnections < 5 {
		newConfig.MaxOpenConnections = 5
	}
	if newConfig.MaxIdleConnections < 2 {
		newConfig.MaxIdleConnections = 2
	}

	return s.ConfigureConnectionPool(newConfig)
}

// ResetConnectionPool resets the connection pool to default settings.
func (s *Service) ResetConnectionPool() error {
	return s.ConfigureConnectionPool(DefaultPoolConfig())
}

// GetPoolStats returns connection pool statistics for monitoring.
// Returns zero values if the database instance doesn't support pool statistics.
func (s *Service) GetPoolStats() dbkit.PoolStats {
	// Check if we have a DBKit instance
	if db, ok := s.db.(*dbkit.DBKit); ok {
		sqlStats := db.Stats()
		return dbkit.PoolStatsFromSQL(sqlStats)
	}

	// Return zero values for non-DBKit instances
	return dbkit.PoolStats{}
}
