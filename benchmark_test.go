package rolekit

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// skipBenchmarkIfNoDatabase skips the benchmark if database is not available
func skipBenchmarkIfNoDatabase(b *testing.B) (*Service, context.Context) {
	if !isDatabaseAvailable() {
		b.Skip("Database not available, skipping benchmark")
		return nil, nil
	}

	ctx := context.Background()
	service, err := setupTestDatabase(ctx)
	if err != nil {
		b.Fatalf("Failed to setup test database: %v", err)
	}

	return service, ctx
}

// ============================================================================
// Role Assignment Benchmarks
// ============================================================================

// BenchmarkAssign benchmarks the Assign method
func BenchmarkAssign(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	// Setup admin user
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Assign admin role first
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("bench-user-%d-%d", time.Now().UnixNano(), i)
		err := service.Assign(actorCtx, userID, "developer", "organization", orgID)
		if err != nil {
			b.Errorf("Assign failed: %v", err)
		}
	}
}

// BenchmarkAssignDirect benchmarks the AssignDirect method (bypasses checks)
func BenchmarkAssignDirect(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("bench-user-%d-%d", time.Now().UnixNano(), i)
		// AssignDirect may fail due to various reasons, we just measure the call
		_ = service.AssignDirect(actorCtx, userID, "developer", "organization", orgID)
	}
}

// BenchmarkAssignMultiple benchmarks bulk assignment
func BenchmarkAssignMultiple(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		assignments := make([]RoleAssignment, 10)
		for j := 0; j < 10; j++ {
			assignments[j] = RoleAssignment{
				UserID:    fmt.Sprintf("bench-user-%d-%d-%d", time.Now().UnixNano(), i, j),
				Role:      "developer",
				ScopeType: "organization",
				ScopeID:   orgID,
			}
		}
		err := service.AssignMultiple(actorCtx, assignments)
		if err != nil {
			b.Errorf("AssignMultiple failed: %v", err)
		}
	}
}

// BenchmarkRevoke benchmarks the Revoke method
func BenchmarkRevoke(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}

	// Pre-create users with roles
	userIDs := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		userIDs[i] = fmt.Sprintf("bench-user-%d-%d", time.Now().UnixNano(), i)
		if err := service.Assign(actorCtx, userIDs[i], "developer", "organization", orgID); err != nil {
			b.Fatalf("Failed to assign role: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := service.Revoke(actorCtx, userIDs[i], "developer", "organization", orgID)
		if err != nil {
			b.Errorf("Revoke failed: %v", err)
		}
	}
}

// ============================================================================
// Permission Checking Benchmarks
// ============================================================================

// BenchmarkCan benchmarks the Can method
func BenchmarkCan(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	userID := fmt.Sprintf("bench-user-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin and user
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}
	if err := service.Assign(actorCtx, userID, "developer", "organization", orgID); err != nil {
		b.Fatalf("Failed to assign role: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Can(ctx, userID, "developer", "organization", orgID)
	}
}

// BenchmarkHasPermission benchmarks the HasPermission method
func BenchmarkHasPermission(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	userID := fmt.Sprintf("bench-user-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin and user
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}
	if err := service.Assign(actorCtx, userID, "developer", "organization", orgID); err != nil {
		b.Fatalf("Failed to assign role: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.HasPermission(ctx, userID, "task.create", "organization", orgID)
	}
}

// BenchmarkCheckExists benchmarks the CheckExists method
func BenchmarkCheckExists(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	userID := fmt.Sprintf("bench-user-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin and user
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}
	if err := service.Assign(actorCtx, userID, "developer", "organization", orgID); err != nil {
		b.Fatalf("Failed to assign role: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.CheckExists(ctx, userID, "developer", "organization", orgID)
	}
}

// ============================================================================
// Query Benchmarks
// ============================================================================

// BenchmarkGetUserRoles benchmarks the GetUserRoles method
func BenchmarkGetUserRoles(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	userID := fmt.Sprintf("bench-user-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin and user with multiple roles
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}
	if err := service.Assign(actorCtx, userID, "developer", "organization", orgID); err != nil {
		b.Fatalf("Failed to assign role: %v", err)
	}
	if err := service.Assign(actorCtx, userID, "viewer", "organization", orgID); err != nil {
		b.Fatalf("Failed to assign role: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetUserRoles(ctx, userID)
		if err != nil {
			b.Errorf("GetUserRoles failed: %v", err)
		}
	}
}

// BenchmarkCountRoles benchmarks the CountRoles method
func BenchmarkCountRoles(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	userID := fmt.Sprintf("bench-user-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin and user
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}
	if err := service.Assign(actorCtx, userID, "developer", "organization", orgID); err != nil {
		b.Fatalf("Failed to assign role: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.CountRoles(ctx, userID, "organization", orgID)
		if err != nil {
			b.Errorf("CountRoles failed: %v", err)
		}
	}
}

// ============================================================================
// Transaction Benchmarks
// ============================================================================

// BenchmarkTransaction benchmarks transaction overhead
func BenchmarkTransaction(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("bench-user-%d-%d", time.Now().UnixNano(), i)
		err := service.Transaction(actorCtx, func(ctx context.Context) error {
			return service.Assign(ctx, userID, "developer", "organization", orgID)
		})
		if err != nil {
			b.Errorf("Transaction failed: %v", err)
		}
	}
}

// BenchmarkTransactionVsNoTransaction compares transaction vs direct assignment
func BenchmarkTransactionVsNoTransaction(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}

	b.Run("WithTransaction", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			userID := fmt.Sprintf("bench-user-tx-%d-%d", time.Now().UnixNano(), i)
			err := service.Transaction(actorCtx, func(ctx context.Context) error {
				return service.Assign(ctx, userID, "developer", "organization", orgID)
			})
			if err != nil {
				b.Errorf("Transaction failed: %v", err)
			}
		}
	})

	b.Run("WithoutTransaction", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			userID := fmt.Sprintf("bench-user-notx-%d-%d", time.Now().UnixNano(), i)
			err := service.Assign(actorCtx, userID, "developer", "organization", orgID)
			if err != nil {
				b.Errorf("Assign failed: %v", err)
			}
		}
	})
}

// ============================================================================
// Concurrent Access Benchmarks
// ============================================================================

// BenchmarkConcurrentAssign benchmarks concurrent role assignments
func BenchmarkConcurrentAssign(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			userID := fmt.Sprintf("bench-user-%d-%d", time.Now().UnixNano(), counter)
			counter++
			err := service.Assign(actorCtx, userID, "developer", "organization", orgID)
			if err != nil {
				b.Errorf("Assign failed: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentCan benchmarks concurrent permission checks
func BenchmarkConcurrentCan(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	userID := fmt.Sprintf("bench-user-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin and user
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}
	if err := service.Assign(actorCtx, userID, "developer", "organization", orgID); err != nil {
		b.Fatalf("Failed to assign role: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = service.Can(ctx, userID, "developer", "organization", orgID)
		}
	})
}

// BenchmarkConcurrentMixedOperations benchmarks mixed read/write operations
func BenchmarkConcurrentMixedOperations(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}

	// Pre-create some users
	existingUsers := make([]string, 100)
	for i := 0; i < 100; i++ {
		existingUsers[i] = fmt.Sprintf("bench-existing-%d-%d", time.Now().UnixNano(), i)
		if err := service.Assign(actorCtx, existingUsers[i], "developer", "organization", orgID); err != nil {
			b.Fatalf("Failed to assign role: %v", err)
		}
	}

	b.ResetTimer()
	var wg sync.WaitGroup
	errChan := make(chan error, b.N*2)

	// Writers
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			userID := fmt.Sprintf("bench-new-%d-%d", time.Now().UnixNano(), i)
			if err := service.Assign(actorCtx, userID, "developer", "organization", orgID); err != nil {
				errChan <- err
			}
		}
	}()

	// Readers
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			userIdx := i % len(existingUsers)
			_ = service.Can(ctx, existingUsers[userIdx], "developer", "organization", orgID)
		}
	}()

	wg.Wait()
	close(errChan)

	for err := range errChan {
		b.Errorf("Operation failed: %v", err)
	}
}

// ============================================================================
// Health and Pool Benchmarks
// ============================================================================

// BenchmarkPing benchmarks the Ping method
func BenchmarkPing(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := service.Ping(ctx)
		if err != nil {
			b.Errorf("Ping failed: %v", err)
		}
	}
}

// BenchmarkIsHealthy benchmarks the IsHealthy method
func BenchmarkIsHealthy(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.IsHealthy(ctx)
	}
}

// BenchmarkGetPoolStats benchmarks the GetPoolStats method
func BenchmarkGetPoolStats(b *testing.B) {
	service, _ := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.GetPoolStats()
	}
}

// ============================================================================
// Memory Allocation Benchmarks
// ============================================================================

// BenchmarkAssignAllocs measures memory allocations for Assign
func BenchmarkAssignAllocs(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("bench-user-%d-%d", time.Now().UnixNano(), i)
		_ = service.Assign(actorCtx, userID, "developer", "organization", orgID)
	}
}

// BenchmarkCanAllocs measures memory allocations for Can
func BenchmarkCanAllocs(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	userID := fmt.Sprintf("bench-user-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin and user
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}
	if err := service.Assign(actorCtx, userID, "developer", "organization", orgID); err != nil {
		b.Fatalf("Failed to assign role: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Can(ctx, userID, "developer", "organization", orgID)
	}
}

// BenchmarkGetUserRolesAllocs measures memory allocations for GetUserRoles
func BenchmarkGetUserRolesAllocs(b *testing.B) {
	service, ctx := skipBenchmarkIfNoDatabase(b)
	if service == nil {
		return
	}

	orgID := fmt.Sprintf("bench-org-%d", time.Now().UnixNano())
	userID := fmt.Sprintf("bench-user-%d", time.Now().UnixNano())
	adminID := fmt.Sprintf("bench-admin-%d", time.Now().UnixNano())
	actorCtx := WithActorID(ctx, adminID)

	// Setup admin and user with multiple roles
	if err := service.Assign(actorCtx, adminID, "super_admin", "organization", orgID); err != nil {
		b.Fatalf("Failed to setup admin: %v", err)
	}
	if err := service.Assign(actorCtx, userID, "developer", "organization", orgID); err != nil {
		b.Fatalf("Failed to assign role: %v", err)
	}
	if err := service.Assign(actorCtx, userID, "viewer", "organization", orgID); err != nil {
		b.Fatalf("Failed to assign role: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetUserRoles(ctx, userID)
	}
}
