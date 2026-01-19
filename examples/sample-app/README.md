# RoleKit Sample Application

This sample application demonstrates all RoleKit features in a realistic scenario with a real PostgreSQL database.

## Overview

The sample application shows how to:

- Set up role hierarchies and permissions
- Assign and manage user roles
- Use transactions for atomic operations
- Monitor database health
- Optimize connection pools
- Perform bulk operations
- Handle errors gracefully
- Test performance under load

## Prerequisites

- PostgreSQL 18 (or other supported versions)
- Go 1.25.5 or later

## Setup

### 1. Start PostgreSQL

Using Docker:

```bash
docker run --name rolekit-postgres \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=rolekit_test \
  -p 5432:5432 \
  -d postgres:18
```

### 2. Run the Sample Application

```bash
cd examples/sample-app
export DATABASE_URL="postgres://postgres:password@localhost:5432/rolekit_test?sslmode=disable"
go run .
```

## What the Sample Application Does

### 1. Role Hierarchy Setup

The application defines a comprehensive role hierarchy:

```
organization
├── super_admin (all permissions)
├── org_admin (organization, project, team, task permissions)
└── project
    ├── project_manager (project, team, task permissions)
    └── team
        ├── team_lead (team, task permissions)
        ├── developer (team.read, task.*)
        └── task
            └── viewer (read-only permissions)
```

### 2. Test Scenarios

The application runs through these test scenarios:

#### Basic Role Assignment

- Creates sample users and organization
- Assigns roles to users
- Tests permission checking
- Verifies access control

#### Complex Role Hierarchy

- Tests role inheritance across scope levels
- Verifies permission propagation
- Tests parent-child scope relationships

#### Transaction Support

- Demonstrates atomic role assignments
- Tests transaction rollback
- Verifies data consistency

#### Health Monitoring

- Checks database connectivity
- Monitors connection pool statistics
- Tests optimization features

#### Connection Pool Optimization

- Tests different pool configurations
- Demonstrates dynamic adjustment
- Verifies performance tuning

#### Bulk Operations

- Performs bulk role assignments
- Tests bulk revocations
- Measures performance

#### Error Handling

- Tests unauthorized assignments
- Handles non-existent users
- Validates error scenarios

#### Performance Testing

- Creates 100 test users
- Measures bulk assignment performance
- Tests permission check speed
- Benchmarks user role queries

## Sample Output

```
🚀 Starting RoleKit Sample Application
🔄 Running database migrations...
✅ Migrations completed: 1 applied
🔧 Configuring connection pool...
✅ Connection pool configured: MaxOpen=100, MaxIdle=50
📋 Running scenario: Basic Role Assignment
  ✅ Assigned super_admin role to user-1
  ✅ Assigned project_manager role to user-2
  ✅ Assigned developer role to user-3
  ✅ Assigned viewer role to user-4
  ✅ User user-1 can organization.delete: true
  ✅ User user-2 can project.create: true
  ✅ User user-3 can task.create: true
  ✅ User user-4 can task.create: false
  ✅ User user-4 can task.read: true
✅ Scenario completed: Basic Role Assignment
...
🎉 All scenarios completed successfully!
```

## Configuration

You can configure the sample application using environment variables:

- `DATABASE_URL`: PostgreSQL connection string (default: `postgres://postgres:password@localhost:5432/rolekit_test?sslmode=disable`)

## Key Features Demonstrated

### 1. Fluent API

```go
registry.DefineScope("organization").
    Role("super_admin").Permissions("*").CanAssign("*").
    Role("org_admin").Permissions("organization.*", "project.*").
    DefineScope("project").
        Role("project_manager").Permissions("project.*", "team.*")
```

### 2. Context-Based Operations

```go
ctx := rolekit.WithActorID(ctx, "user-1")
err := service.Assign(ctx, userID, role, scope, scopeID)

ctx := rolekit.WithUserID(ctx, userID)
hasPermission := service.Can(ctx, userID, permission, scope, scopeID)
```

### 3. Transaction Support

```go
err := service.Transaction(ctx, func(ctx context.Context) error {
    // Multiple operations here
    return nil
})
```

### 4. Connection Pool Management

```go
config := rolekit.HighPerformancePoolConfig()
err := service.ConfigureConnectionPool(config)

stats := service.GetPoolStats()
err := service.OptimizeConnectionPool()
```

### 5. Bulk Operations

```go
assignments := []rolekit.RoleAssignment{...}
err := service.AssignMultiple(ctx, assignments)

revocations := []rolekit.RoleRevocation{...}
err := service.RevokeMultiple(ctx, revocations)
```

### 6. Health Monitoring

```go
healthy := service.IsHealthy(ctx)
health := service.Health(ctx)
stats := service.GetPoolStats()
```

## Performance Metrics

The sample application measures and reports:

- Bulk assignment speed (100 users)
- Permission check performance
- User role query speed
- Connection pool statistics

## Database Schema

The sample application creates these tables:

- `users`: User information
- `organizations`: Organization data
- `projects`: Project information
- `role_assignments`: RoleKit's role assignment table (created automatically)

## Troubleshooting

### Database Connection Issues

- Ensure PostgreSQL is running
- Check the DATABASE_URL environment variable
- Verify database exists and is accessible

### Permission Denied Errors

- Check that the database user has proper permissions
- Ensure the role assignment rules are correctly configured

### Performance Issues

- Monitor connection pool statistics
- Try different pool configurations
- Check database query performance

## Extending the Sample

You can extend this sample application by:

1. Adding more complex role hierarchies
2. Implementing additional business logic
3. Adding web API endpoints
4. Creating monitoring dashboards
5. Adding more performance tests
6. Implementing caching strategies

## Best Practices Demonstrated

- Use context for all operations
- Handle errors gracefully
- Use transactions for data consistency
- Monitor database health
- Optimize connection pools
- Test bulk operations for performance
- Validate all inputs
- Use structured logging

This sample application serves as both a comprehensive test suite and a reference implementation for using RoleKit in production applications.
