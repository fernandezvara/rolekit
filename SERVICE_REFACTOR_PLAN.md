# Service.go Refactoring Plan

## Current State Analysis

**File Size**: 1,736 lines (too large)
**Functions**: 30+ methods in single file
**Test Coverage**: No dedicated tests for service.go
**Responsibilities**: Mixed concerns (transactions, migrations, health, pool, roles, etc.)

## Proposed File Structure

### 1. Core Service (`service.go`)

**Lines**: ~200
**Responsibilities**: Core service definition and basic operations

- `Service` struct definition
- `NewService()` constructor
- `Registry()` method
- Basic role operations (`Assign`, `Revoke`, `GetUserRoles`)

### 2. Transaction Management (`service_transactions.go`)

**Lines**: ~300
**Responsibilities**: All transaction-related functionality

- `TransactionMetrics` struct
- `transactionMonitor` struct
- `Transaction()`, `TransactionWithOptions()`, `ReadOnlyTransaction()`
- `AssignDirect()`, `AssignWithRetry()`, `AssignMultipleWithRetry()`
- Transaction monitoring methods (`GetTransactionMetrics()`, `ResetTransactionMetrics()`, `IsTransactionHealthy()`)
- Error recovery functions (`isTransientTransactionError()`)

### 3. Migration Management (`service_migrations.go`)

**Lines**: ~400
**Responsibilities**: Database migration functionality

- `MigrationStatus` struct
- `Migrations()` method
- `RunMigrations()`, `GetMigrationStatus()`, `VerifyMigrationChecksums()`
- `RollbackToMigration()`, `ValidateMigrations()`
- All migration definitions

### 4. Health Monitoring (`service_health.go`)

**Lines**: ~200
**Responsibilities**: Database health and monitoring

- `Health()`, `IsHealthy()`, `Ping()`
- `GetPoolStats()`
- Health check utilities

### 5. Connection Pool Management (`service_pool.go`)

**Lines**: ~300
**Responsibilities**: Connection pool configuration and optimization

- `PoolConfig` struct
- `ConfigureConnectionPool()`, `GetConnectionPoolConfig()`
- `OptimizeConnectionPool()`, `ResetConnectionPool()`
- Pool monitoring and optimization logic

### 6. Bulk Operations (`service_bulk.go`)

**Lines**: ~200
**Responsibilities**: Bulk role operations

- `RoleRevocation` struct
- `AssignMultiple()`, `RevokeMultiple()`
- Bulk operation utilities

### 7. Query Utilities (`service_queries.go`)

**Lines**: ~150
**Responsibilities**: Helper query methods

- `CheckExists()`, `CountRoles()`, `CountAllRoles()`
- Other utility query functions

### 8. Audit Logging (`service_audit.go`)

**Lines**: ~100
**Responsibilities**: Audit trail functionality

- `AuditEntry` struct
- `logAudit()` method
- Audit utilities

## Revised Implementation Strategy

### Phase 2: Extension-Based Approach (Updated)

Instead of creating duplicate types and interfaces, I'll use an extension-based approach:

1. **Create extension files** that work with existing Service struct
2. **Use composition** to add functionality without breaking existing code
3. **Gradual migration** by moving methods one by one
4. **Maintain backward compatibility** throughout the process

### Implementation Approach

1. **Create extension structs** that embed the Service
2. **Add methods to Service** that delegate to extensions
3. **Gradually move functionality** from service.go to extension files
4. **Maintain all public APIs** without breaking changes
5. **Add comprehensive tests** for each extension

### Benefits of This Approach

- ✅ No type conflicts
- ✅ Backward compatibility maintained
- ✅ Gradual migration possible
- ✅ Clear separation of concerns
- ✅ Easy to test individual components

## Testing Strategy

### 1. Service Tests (`service_test.go`)

- Core service functionality
- Basic role assignments and revocations
- User role retrieval
- Error handling

### 2. Transaction Tests (`transaction_test.go`) ✅ EXISTS

- Transaction context propagation
- Rollback scenarios
- Direct assignment
- Retry logic
- Monitoring and metrics
- Concurrent transactions

### 3. Migration Tests (`migration_test.go`)

- Migration execution
- Status tracking
- Checksum verification
- Rollback scenarios

### 4. Health Tests (`health_test.go`)

- Health checks
- Pool statistics
- Connectivity tests

### 5. Pool Tests (`pool_test.go`)

- Pool configuration
- Optimization logic
- Monitoring

### 6. Bulk Operations Tests (`bulk_test.go`)

- Bulk assignments
- Bulk revocations
- Error handling

### 7. Query Tests (`query_test.go`)

- Existence checks
- Count operations
- Query utilities

### 8. Audit Tests (`audit_test.go`)

- Audit logging
- Entry creation

## Implementation Steps

### Phase 1: Analysis and Preparation

1. **Review current service.go structure**
2. **Identify dependencies between functions**
3. **Create test files for each new module**
4. **Set up proper interfaces for decoupling**

### Phase 2: Extract Core Components

1. **Create `transactions.go`** - Extract all transaction-related code
2. **Create `migrations.go`** - Extract migration functionality
3. **Create `health.go`** - Extract health monitoring
4. **Create `pool.go`** - Extract connection pool management

### Phase 3: Extract Supporting Components

1. **Create `bulk.go`** - Extract bulk operations
2. **Create `queries.go`** - Extract query utilities
3. **Create `audit.go`** - Extract audit functionality

### Phase 4: Refactor Core Service

1. **Slim down `service.go`** to core functionality only
2. **Add proper interfaces** for dependency injection
3. **Ensure all imports are correct**

### Phase 5: Comprehensive Testing

1. **Write tests for each new module**
2. **Ensure 100% test coverage**
3. **Add integration tests**
4. **Performance tests for critical paths**

### Phase 6: Documentation and Validation

1. **Update README.md** with new structure
2. **Add godoc comments** to all public functions
3. **Validate all functionality works**
4. **Run full test suite**

## Success Criteria

- ✅ Each file under 300 lines
- ✅ Clear separation of concerns
- ✅ 100% test coverage
- ✅ All existing functionality preserved
- ✅ Better maintainability and readability
- ✅ Proper interfaces for testing
- ✅ Comprehensive documentation

## Risk Mitigation

### High Risk Areas

1. **Transaction context propagation** - Ensure no breaking changes
2. **Migration dependencies** - Maintain order and relationships
3. **Service method signatures** - Keep public API stable

### Mitigation Strategies

1. **Incremental refactoring** - One module at a time
2. **Comprehensive testing** - After each extraction
3. **Backward compatibility** - Maintain all public APIs
4. **Integration testing** - Ensure modules work together

## Timeline Estimate

- **Phase 1**: 1 day (analysis and preparation)
- **Phase 2**: 2 days (core components)
- **Phase 3**: 1 day (supporting components)
- **Phase 4**: 1 day (core service refactor)
- **Phase 5**: 3 days (comprehensive testing)
- **Phase 6**: 1 day (documentation and validation)

**Total Estimated Time**: 9 days

## Next Steps

1. **Review and approve this plan**
2. **Start with Phase 1** - Analysis and preparation
3. **Create test infrastructure** before refactoring
4. **Proceed with incremental extraction** starting with transactions.go
