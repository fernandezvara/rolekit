package rolekit

import (
	"context"
	"fmt"
	"time"

	"github.com/fernandezvara/dbkit"
)

// Transaction executes a function within a database transaction with automatic commit/rollback.
// If the function returns an error, the transaction is rolled back. Otherwise, it's committed.
//
// Example:
//
//	err := service.Transaction(ctx, func(ctx context.Context) error {
//	    if err := service.Assign(ctx, "user1", "admin", "organization", "org1"); err != nil {
//	        return err // This will cause a rollback
//	    }
//	    if err := service.Assign(ctx, "user2", "member", "organization", "org1"); err != nil {
//	        return err // This will cause a rollback
//	    }
//	    return nil // This will cause a commit
//	})
func (s *Service) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	start := time.Now()
	var err error

	// Check if we're already in a transaction by casting to dbkit.Tx
	if tx, ok := s.db.(*dbkit.Tx); ok {
		// We're already in a transaction, use savepoint
		err = tx.Transaction(ctx, func(tx *dbkit.Tx) error {
			// Use the transaction directly for operations within this scope
			return fn(ctx)
		})
	} else {
		// We're not in a transaction, start a new one
		if db, ok := s.db.(*dbkit.DBKit); ok {
			err = db.Transaction(ctx, func(tx *dbkit.Tx) error {
				// Use the transaction directly for operations within this scope
				return fn(ctx)
			})
		} else {
			// If we can't determine the type, try to use the generic interface
			// This is a fallback - ideally we'd have better type information
			err = fmt.Errorf("transaction support requires a dbkit.DBKit or dbkit.Tx instance")
		}
	}

	// Record transaction metrics
	duration := time.Since(start)
	s.txMonitor.recordTransaction(duration, err == nil)

	return err
}

// TransactionWithOptions executes a function within a database transaction with custom options.
// Supports read-only transactions, isolation levels, and other transaction parameters.
//
// Example:
//
//	err := service.TransactionWithOptions(ctx, dbkit.SerializableTxOptions(), func(ctx context.Context) error {
//	    // High isolation level operations
//	    return service.Assign(ctx, "user1", "admin", "organization", "org1")
//	})
func (s *Service) TransactionWithOptions(ctx context.Context, opts dbkit.TxOptions, fn func(ctx context.Context) error) error {
	// Check if we're already in a transaction by casting to dbkit.Tx
	if tx, ok := s.db.(*dbkit.Tx); ok {
		// We're already in a transaction, use savepoint (no options support in nested transactions)
		return tx.Transaction(ctx, func(tx *dbkit.Tx) error {
			// Create a new service that uses the transaction
			s.db = tx
			return fn(ctx)
		})
	}

	// We're not in a transaction, start a new one
	if db, ok := s.db.(*dbkit.DBKit); ok {
		return db.TransactionWithOptions(ctx, opts, func(tx *dbkit.Tx) error {
			// Create a new service that uses the transaction
			s.db = tx
			return fn(ctx)
		})
	}

	// If we can't determine the type, try to use the generic interface
	return fmt.Errorf("transaction support requires a dbkit.DBKit or dbkit.Tx instance")
}

// ReadOnlyTransaction executes a function within a read-only database transaction.
// Useful for operations that only read data and want to ensure consistency.
//
// Example:
//
//	err := service.ReadOnlyTransaction(ctx, func(ctx context.Context) error {
//	    roles, err := service.GetUserRoles(ctx, userID)
//	    if err != nil {
//	        return err
//	    }
//	    members, err := service.GetScopeMembers(ctx, "organization", orgID)
//	    return err
//	})
func (s *Service) ReadOnlyTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.TransactionWithOptions(ctx, dbkit.ReadOnlyTxOptions(), fn)
}
