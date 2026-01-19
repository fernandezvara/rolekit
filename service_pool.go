package rolekit

import (
	"fmt"
	"log"

	"github.com/fernandezvara/dbkit"
)

// PoolService provides connection pool management functionality as an extension to Service
type PoolService struct {
	*Service
}

// NewPoolService creates a new pool service extension
func NewPoolService(service *Service) *PoolService {
	return &PoolService{Service: service}
}

// ConfigureConnectionPool updates the database connection pool settings.
func (ps *PoolService) ConfigureConnectionPool(config PoolConfig) error {
	if db, ok := ps.db.(*dbkit.DBKit); ok {
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
func (ps *PoolService) GetConnectionPoolConfig() (*PoolConfig, error) {
	if db, ok := ps.db.(*dbkit.DBKit); ok {
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
func (ps *PoolService) OptimizeConnectionPool() error {
	stats := ps.GetPoolStats()

	config, err := ps.GetConnectionPoolConfig()
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

	return ps.ConfigureConnectionPool(newConfig)
}

// ResetConnectionPool resets the connection pool to default settings.
func (ps *PoolService) ResetConnectionPool() error {
	return ps.ConfigureConnectionPool(DefaultPoolConfig())
}
