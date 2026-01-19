# RoleKit dbkit Integration Enhancement Plan

## Overview

This plan ensures RoleKit properly utilizes the updated `github.com/fernandezvara/dbkit` library, taking advantage of its latest features, improved error handling, and enhanced functionality.

## Feature Changes Summary

| Feature                 | Action                                        | Rationale                                                           |
| ----------------------- | --------------------------------------------- | ------------------------------------------------------------------- |
| **dbkit version**       | Update to latest version                      | Leverage latest bug fixes, features, and improvements               |
| **Error handling**      | Enhance with dbkit's chainable error wrapping | Better error classification, context preservation, and traceability |
| **Transaction support** | Add transaction capabilities                  | Support for atomic operations and rollbacks                         |
| **Health checks**       | Add database health monitoring                | Better observability and monitoring                                 |
| **Connection pooling**  | Optimize connection usage                     | Better performance and resource management                          |
| **Migration system**    | Enhance with dbkit's migration features       | Better migration tracking and status                                |
| **Query optimization**  | Use dbkit's optimized helpers                 | Better performance and type safety                                  |
| **Legacy patterns**     | Remove any remaining direct Bun usage         | Consistent abstraction layer                                        |

## User Stories

### Story 1: Update dbkit Dependency and Verify Integration

**As a** library maintainer  
**I want to** update to the latest dbkit version and verify all integrations work correctly  
**So that** we leverage the latest features and improvements

**Acceptance Criteria:**

1. Update dbkit to the latest version
2. Verify all existing functionality works with the new version
3. Update any deprecated function calls
4. Ensure all tests pass

**Implementation Tasks:**

1. **Implement the user story** - Update go.mod with latest dbkit version
2. **Document all functions** - Document any changes in function signatures or behavior
3. **Update README.md** - Update version requirements and compatibility notes
4. **Add tests** - Verify all existing functionality works with new dbkit version
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 2: Enhance Error Handling with dbkit's Chainable Error Wrapping

**As a** developer using RoleKit  
**I want to** receive detailed, chainable errors with context from database operations  
**So that** I can handle different error scenarios appropriately and trace error origins

**Acceptance Criteria:**

1. Use dbkit's chainable error wrapping with context preservation
2. Wrap dbkit errors with RoleKit context using chainable error methods
3. Provide clear error messages with actionable information and error codes
4. Support error unwrapping to access original dbkit errors
5. Maintain backward compatibility with existing error handling

**Implementation Tasks:**

1. **Implement the user story** - Update all database operations to use dbkit's chainable error wrapping
2. **Document all functions** - Document error types, chainable methods, and handling patterns
3. **Update README.md** - Add error handling examples and best practices with chainable errors
4. **Add tests** - Test error scenarios, error chaining, and error type preservation
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 3: Add Transaction Support for Atomic Operations

**As a** developer using RoleKit  
**I want to** perform multiple role operations within a transaction  
**So that** I can ensure data consistency and rollback on failures

**Acceptance Criteria:**

1. Add transaction methods to Service (Transaction, TransactionWithOptions)
2. Support nested transactions with savepoints
3. Ensure all Service methods can work within transactions
4. Provide transaction examples for common use cases

**Implementation Tasks:**

1. **Implement the user story** - Add transaction support to Service struct
2. **Document all functions** - Document transaction methods and usage patterns
3. **Update README.md** - Add transaction examples and best practices
4. **Add tests** - Test transaction commit, rollback, and nested transactions
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 4: Add Database Health Monitoring

**As a** system operator  
**I want to** monitor the health of the RoleKit database connection  
**So that** I can detect and respond to database issues proactively

**Acceptance Criteria:**

1. Add health check methods to Service (Health, IsHealthy)
2. Include connection pool statistics in health status
3. Support health check endpoints for monitoring systems
4. Provide configurable health check thresholds

**Implementation Tasks:**

1. **Implement the user story** - Add health monitoring capabilities to Service
2. **Document all functions** - Document health check methods and status interpretation
3. **Update README.md** - Add health monitoring examples and integration patterns
4. **Add tests** - Test health check scenarios and status reporting
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 5: Optimize Database Operations with dbkit Helpers

**As a** developer using RoleKit  
**I want to** benefit from optimized database operations  
**So that** my application performs better with type-safe queries

**Acceptance Criteria:**

1. Replace any remaining direct Bun queries with dbkit helpers
2. Use appropriate dbkit functions (FindAll, FindOne, Create, Update, Delete, etc.)
3. Optimize common queries for better performance
4. Ensure type safety in all database operations

**Implementation Tasks:**

1. **Implement the user story** - Optimize all database operations using dbkit helpers
2. **Document all functions** - Document optimized operations and performance benefits
3. **Update README.md** - Add performance optimization examples and benchmarks
4. **Add tests** - Test optimized operations and verify performance improvements
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 6: Enhance Migration System with dbkit Features

**As a** developer deploying RoleKit  
**I want to** have robust migration tracking and status reporting  
**So that** I can safely manage database schema changes

**Acceptance Criteria:**

1. Use dbkit's migration status tracking features
2. Add migration rollback capabilities
3. Provide migration status reporting
4. Support migration checksums for integrity verification

**Implementation Tasks:**

1. **Implement the user story** - Enhance migration system with dbkit features
2. **Document all functions** - Document migration methods and status reporting
3. **Update README.md** - Add migration management examples and best practices
4. **Add tests** - Test migration scenarios, status tracking, and rollback
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 7: Add Connection Pool Configuration and Monitoring

**As a** system administrator  
**I want to** configure and monitor database connection pools  
**So that** I can optimize resource usage and prevent connection exhaustion

**Acceptance Criteria:**

1. Support dbkit connection pool configuration
2. Add connection pool statistics monitoring
3. Provide connection pool tuning recommendations
4. Support dynamic connection pool adjustment

**Implementation Tasks:**

1. **Implement the user story** - Add connection pool configuration and monitoring
2. **Document all functions** - Document connection pool options and monitoring
3. **Update README.md** - Add connection pool configuration examples and tuning guide
4. **Add tests** - Test connection pool configuration and monitoring features
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 8: Sample Application Testing with Real Database Integration

**As a** developer maintaining RoleKit  
**I want to** have a comprehensive sample application that tests all RoleKit features with a real PostgreSQL database  
**So that** I can verify the library works correctly in production-like scenarios and demonstrate its capabilities

**Acceptance Criteria:**

1. Create a sample application that demonstrates all RoleKit features
2. Set up PostgreSQL 18 database configuration using reference files
3. Add comprehensive role assignments and test scenarios
4. Test all major features: roles, permissions, transactions, health monitoring, connection pooling, migrations
5. Include performance testing and load scenarios
6. Provide clear documentation and usage examples

**Implementation Tasks:**

1. **Create sample application structure** - Build a complete test application
2. **Set up database configuration** - Use reference files for PostgreSQL 18 setup
3. **Implement test scenarios** - Create comprehensive role and permission tests
4. **Add performance testing** - Test connection pooling and optimization
5. **Document sample application** - Add usage documentation and examples
6. **Ensure tests pass** - Verify all scenarios work correctly
7. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 9: Fix Transaction Context Propagation in RoleKit

**As a** developer using RoleKit  
**I want to** have consistent transaction handling across all role operations  
**So that** role assignments work reliably within transactions without "transaction already committed" errors

**Acceptance Criteria:**

1. All database operations within a single method use the same transaction context
2. No "transaction already committed" errors during role assignments
3. Role assignments work correctly within nested transactions
4. Transaction context is properly propagated through all internal method calls

**Implementation Tasks:**

1. **Implement the user story** - Fix Assign() method to use single transaction context
2. **Document all functions** - Document transaction context handling patterns
3. **Update README.md** - Add transaction best practices and examples
4. **Add tests** - Test transaction context propagation and nested transactions
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 10: Add Transaction-Safe Role Assignment Methods

**As a** system administrator  
**I want to** have atomic role assignments that never leave the database in an inconsistent state  
**So that** concurrent role assignments don't conflict and failed assignments don't create partial data

**Acceptance Criteria:**

1. Role assignment is fully atomic (all or nothing)
2. Concurrent role assignments don't conflict
3. Failed assignments don't create partial data
4. Audit trail is maintained for all operations
5. High-performance direct assignment method available

**Implementation Tasks:**

1. **Implement the user story** - Add AssignDirect() method for atomic assignments
2. **Document all functions** - Document atomic assignment patterns and performance benefits
3. **Update README.md** - Add atomic assignment examples and concurrency patterns
4. **Add tests** - Test atomic assignments, concurrent access, and failure scenarios
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 11: Implement Proper Bulk Operations with Transaction Safety

**As a** developer using RoleKit  
**I want to** efficiently assign multiple roles in a single operation  
**So that** I can optimize performance for large-scale role management operations

**Acceptance Criteria:**

1. Bulk assignments work within a single transaction
2. Performance is optimized for large numbers of assignments
3. Partial failures don't corrupt data
4. Proper error reporting for failed assignments
5. Batch insert with proper error handling

**Implementation Tasks:**

1. **Implement the user story** - Fix AssignMultiple() to use proper transaction handling
2. **Document all functions** - Document bulk operation patterns and performance considerations
3. **Update README.md** - Add bulk operation examples and best practices
4. **Add tests** - Test bulk operations, partial failures, and performance benchmarks
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 12: Add Transaction Error Recovery and Retry Logic

**As a** system administrator  
**I want to** have automatic recovery from transient transaction errors  
**So that** the system is resilient to temporary database issues

**Acceptance Criteria:**

1. Automatic retry for transient transaction errors
2. Exponential backoff for failed transactions
3. Clear error categorization (transient vs permanent)
4. Monitoring and alerting for transaction failures
5. Configurable retry policies

**Implementation Tasks:**

1. **Implement the user story** - Add error categorization and retry logic to transaction operations
2. **Document all functions** - Document error types, retry patterns, and configuration options
3. **Update README.md** - Add error handling examples and resilience patterns
4. **Add tests** - Test retry logic, error categorization, and failure scenarios
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 13: Fix GetUserRoles Transaction Handling

**As a** developer using RoleKit  
**I want to** GetUserRoles to work correctly within transaction contexts  
**So that** I can reliably check user permissions within transactions

**Acceptance Criteria:**

1. GetUserRoles respects transaction context
2. No transaction state conflicts
3. Consistent read behavior within transactions
4. Proper isolation level handling
5. Performance optimized for transaction contexts

**Implementation Tasks:**

1. **Implement the user story** - Fix GetUserRoles() to handle transaction context properly
2. **Document all functions** - Document transaction-aware query patterns
3. **Update README.md** - Add transaction-aware query examples
4. **Add tests** - Test GetUserRoles within various transaction scenarios
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 14: Add Transaction Monitoring and Observability

**As a** system administrator  
**I want to** have visibility into transaction performance and failures  
**So that** I can monitor system health and identify performance bottlenecks

**Acceptance Criteria:**

1. Transaction success/failure metrics
2. Transaction duration tracking
3. Deadlock detection and reporting
4. Performance alerts for slow transactions
5. Comprehensive transaction logging

**Implementation Tasks:**

1. **Implement the user story** - Add transaction metrics collection and reporting
2. **Document all functions** - Document monitoring capabilities and metric interpretation
3. **Update README.md** - Add monitoring setup examples and alerting patterns
4. **Add tests** - Test metric collection, reporting, and alerting scenarios
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 15: Comprehensive Transaction Testing and Validation

**As a** developer maintaining RoleKit  
**I want to** have comprehensive test coverage for all transaction scenarios  
**So that** I can ensure transaction reliability and prevent regressions

**Acceptance Criteria:**

1. Unit tests for all transaction methods
2. Integration tests with real PostgreSQL transactions
3. Performance tests for transaction overhead
4. Concurrent access pattern testing
5. Deadlock and failure scenario testing

**Implementation Tasks:**

1. **Implement the user story** - Create comprehensive transaction test suite
2. **Document all functions** - Document test scenarios and validation patterns
3. **Update README.md** - Add testing examples and validation guidelines
4. **Add tests** - Implement all transaction test scenarios
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 16: Convert Sample Application to Comprehensive Integration Tests

**As a** developer maintaining RoleKit  
**I want to** convert the sample application into comprehensive integration tests with a real database  
**So that** all RoleKit features are properly tested with database interactions, improving test coverage and reliability

**Acceptance Criteria:**

1. Convert all sample-app demo functions into proper test functions
2. Set up test database infrastructure using `make start` command
3. Create conditional test execution - run database-dependent tests only when database is available
4. Create integration tests for all major RoleKit features
5. Test role assignments, permissions, transactions, health monitoring, connection pooling, migrations
6. Include performance benchmarks and load testing scenarios
7. Ensure all tests can run in CI/CD pipeline with proper test database setup
8. Maintain backward compatibility - tests without database should still run

**Implementation Tasks:**

1. **Create test database setup** - Use `make start` to set up PostgreSQL test database with proper configuration
2. **Add database availability check** - Create helper function to detect if database is running
3. **Convert demo functions to tests** - Transform all test\* functions in sample-app into proper Go tests
4. **Implement conditional test execution** - Skip database-dependent tests if database not available
5. **Add test utilities** - Create helper functions for test database setup and teardown
6. **Create integration test suite** - Organize tests by feature area (roles, transactions, health, etc.)
7. **Add performance benchmarks** - Convert performance demos into Go benchmark tests
8. **Ensure CI/CD compatibility** - Make tests runnable in automated environments with database setup
9. **Update Makefile** - Ensure `make start` properly configures test database
10. \*\*Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 17: Add Database-Backed Unit Tests for Core Functionality

**As a** developer maintaining RoleKit  
**I want to** have unit tests that use a real database for all core functionality  
**So that** we can verify database interactions work correctly and catch regressions early

**Acceptance Criteria:**

1. Unit tests for all Service methods with real database
2. Test all role assignment and revocation scenarios
3. Test permission checking and validation
4. Test transaction commit/rollback scenarios
5. Test error handling and edge cases
6. Test concurrent access patterns
7. Achieve >90% test coverage for all database operations
8. Use `make start` for database setup with conditional test execution

**Implementation Tasks:**

1. **Create database test utilities** - Helper functions for setting up test data and scenarios
2. **Add database availability check** - Reuse helper function from Story 16 to detect database status
3. **Write role assignment tests** - Test all Assign, Revoke, and bulk operation methods
4. **Write permission tests** - Test all permission checking and validation methods
5. **Write transaction tests** - Test transaction scenarios and error handling
6. **Write query tests** - Test all data retrieval and counting methods
7. **Write health and monitoring tests** - Test health checks and pool management
8. **Implement conditional test execution** - Skip tests if database not available
9. \*\*Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 18: Add Performance and Load Testing Suite

**As a** developer maintaining RoleKit  
**I want to** have performance and load tests to verify RoleKit performs well under stress  
**So that** we can identify performance bottlenecks and ensure scalability

**Acceptance Criteria:**

1. Performance benchmarks for all major operations
2. Load testing for concurrent role assignments
3. Connection pool performance testing
4. Transaction overhead measurement
5. Memory usage profiling
6. Performance regression detection
7. Use `make start` for database setup with conditional test execution

**Implementation Tasks:**

1. **Create benchmark suite** - Go benchmark tests for all major operations
2. **Add load testing** - Concurrent access patterns and stress testing
3. **Profile memory usage** - Memory allocation and garbage collection testing
4. **Measure transaction overhead** - Benchmark transaction vs non-transaction operations
5. **Test connection pooling** - Benchmark different pool configurations
6. **Add performance regression tests** - Automated performance monitoring
7. **Add database availability check** - Skip benchmarks if database not available
8. \*\*Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 19: Add Edge Case and Error Scenario Testing

**As a** developer maintaining RoleKit  
**I want to** comprehensive tests for edge cases and error scenarios  
**So that** the library handles unexpected situations gracefully

**Acceptance Criteria:**

1. Test all error paths and error handling
2. Test database connection failures and recovery
3. Test transaction deadlocks and timeouts
4. Test invalid input handling
5. Test boundary conditions and limits
6. Test concurrent modification scenarios
7. Use `make start` for database setup with conditional test execution

**Implementation Tasks:**

1. **Create error scenario tests** - Test all error conditions and recovery paths
2. **Add failure injection** - Simulate database failures and test recovery
3. **Test edge cases** - Boundary conditions, empty inputs, maximum values
4. **Test concurrency issues** - Race conditions, deadlocks, lost updates
5. **Test data integrity** - Verify data consistency under various conditions
6. **Add chaos testing** - Random failures and recovery scenarios
7. **Add database availability check** - Skip error scenario tests if database not available
8. \*\*Ensure 'make test' and 'make lint' passes without errors before start with other task

---

### Story 20: Set Up GitHub Actions for Database-Backed Testing

**As a** developer maintaining RoleKit  
**I want to** have GitHub Actions that automatically run tests with a real database  
**So that** all database-dependent tests are executed in CI/CD pipeline and regressions are caught early

**Acceptance Criteria:**

1. Create GitHub Actions workflow that sets up PostgreSQL 18 database
2. Run `make start` to initialize test database before running tests
3. Execute full test suite including database-dependent tests
4. Run performance benchmarks in CI pipeline
5. Generate test coverage reports for database tests
6. Handle database setup failures gracefully
7. Support both quick tests (without database) and full tests (with database)

**Implementation Tasks:**

1. **Create GitHub Actions workflow** - Set up PostgreSQL service and test environment
2. **Configure database service** - Use Docker to run PostgreSQL 18 in Actions
3. **Add database initialization** - Run `make start` before test execution
4. **Implement test matrix** - Support different PostgreSQL versions if needed
5. **Add test artifacts** - Save test results and coverage reports
6. **Configure test caching** - Cache dependencies for faster builds
7. **Add conditional workflows** - Support quick PR checks vs full branch tests
8. **Test workflow locally** - Use act or similar tool to validate workflow
9. **Ensure workflow passes** - Verify all tests run successfully in CI
10. \*\*Ensure 'make test' and 'make lint' passes without errors before start with other task

---

## Implementation Rules

For each user story, the following rules must be followed:

1. **Implement the user story** - Write the actual code functionality
2. **Document all functions** - Add comprehensive documentation for all implemented functions to ensure correct library usage
3. **Update README.md** - Add the required documentation to README.md with examples and usage patterns
4. **Add tests** - Write comprehensive tests for all implemented files
5. Ensure 'make test' and 'make lint' passes without errors before start with other task

## Success Criteria

- All dbkit integrations use the latest version and best practices
- Error handling is comprehensive and follows dbkit patterns
- Transaction support is available for atomic operations
- Transaction context propagation works correctly without errors
- Health monitoring provides actionable insights
- Database operations are optimized and type-safe
- Migration system is robust and provides good visibility
- Connection pooling is configurable and monitorable
- Test coverage is comprehensive and reliable (>90% for database operations)
- Transaction operations are atomic, reliable, and performant
- Error recovery and retry logic handles transient failures
- Transaction monitoring provides operational visibility
- Integration tests cover all major features with real database
- Performance benchmarks validate scalability
- Edge cases and error scenarios are thoroughly tested
- All tests can run in CI/CD pipeline with automated database setup

## Timeline Estimate

- **Story 1**: 1-2 days (dependency update and verification)
- **Story 2**: 2-3 days (error handling enhancement)
- **Story 3**: 3-4 days (transaction support)
- **Story 4**: 2-3 days (health monitoring)
- **Story 5**: 3-4 days (operation optimization)
- **Story 6**: 2-3 days (migration enhancement)
- **Story 7**: 2-3 days (connection pooling)
- **Story 8**: 3-4 days (comprehensive testing)
- **Story 9**: 2-3 days (transaction context propagation)
- **Story 10**: 2-3 days (atomic role assignments)
- **Story 11**: 2-3 days (bulk operations)
- **Story 12**: 2-3 days (error recovery and retry)
- **Story 13**: 1-2 days (GetUserRoles transaction handling)
- **Story 14**: 2-3 days (transaction monitoring)
- **Story 15**: 2-3 days (comprehensive transaction testing)
- **Story 16**: 4-5 days (convert sample-app to integration tests)
- **Story 17**: 3-4 days (database-backed unit tests)
- **Story 18**: 2-3 days (performance and load testing)
- **Story 19**: 2-3 days (edge case and error testing)
- **Story 20**: 2-3 days (GitHub Actions database testing setup)

**Total Estimated Time**: 43-62 days

## Risk Assessment

- **Low Risk**: Stories 1, 5, 8, 13, 20 (updates, optimizations, and CI/CD setup)
- **Medium Risk**: Stories 2, 4, 6, 7, 9, 10, 11, 12, 14, 16, 18, 19 (enhancements and new features)
- **High Risk**: Stories 3, 15, 17 (transaction support and comprehensive testing require careful design)

## Dependencies

- All stories depend on Story 1 (dbkit update)
- Stories 9-15 depend on Stories 3 (basic transaction support)
- Story 8 (testing) should be done throughout the process
- Stories 9-15 should be implemented in order as they build on each other
- Stories 16-19 should be done after Stories 1-15 to ensure all functionality is tested
- Story 16 (integration tests) provides the foundation for Stories 17-19
- Story 20 should be implemented after Stories 16-19 to test the complete test suite in CI/CD
