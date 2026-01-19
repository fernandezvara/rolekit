# RoleKit

**Entity-agnostic role and permission management for Go applications.**

RoleKit provides a flexible, hierarchical permission system that works with any entity type. Whether you're building an app with organizations, projects, repositories, teams, or workspaces, RoleKit adapts to your needs.

## Features

- **Entity-Agnostic**: Works with any scope type (organization, project, repository, team, workspace, etc.)
- **Scope-Specific Roles**: "admin" in organization ≠ "admin" in project
- **Full Wildcard Support**: `*`, `resource.*`, `*.action`
- **Multiple Roles per Scope**: Users can have multiple roles, permissions are UNION
- **Hierarchical Scopes**: Parent-child awareness for queries like "get all projects in org where user has role X"
- **Detailed Audit Logging**: Who, what, when, previous state, new state, request metadata
- **Token-Agnostic**: Only needs userID from context
- **DBKit Integration**: Uses your existing database connection via dbkit
- **Middleware + Service**: Both HTTP middleware and service-level checks

## Installation

```bash
go get github.com/fernandezvara/rolekit
```

## Quick Start

### 1. Define Your Roles

```go
package main

import (
    "github.com/fernandezvara/rolekit"
    "github.com/fernandezvara/dbkit"
)

func main() {
    // Create registry and define roles
    registry := rolekit.NewRegistry()

    // Organization roles
    registry.DefineScope("organization").
        Role("owner").
            Permissions("*").
            CanAssign("*").
        Role("admin").
            Permissions("members.*", "settings.*", "billing.read").
            CanAssign("member", "viewer").
        Role("member").
            Permissions("projects.create", "projects.list").
        Role("viewer").
            Permissions("projects.list")

    // Project roles (with parent scope)
    registry.DefineScope("project").
        ParentScope("organization").
        Role("admin").
            Permissions("*").
            CanAssign("*").
        Role("editor").
            Permissions("files.*", "comments.*").
        Role("viewer").
            Permissions("files.read", "comments.read")

    // Create service with your database
    db, _ := dbkit.New(dbkit.Config{URL: "postgres://..."})
    service := rolekit.NewService(registry, db)

    // Run migrations
    _, err = db.Migrate(ctx, service.Migrations())
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2. Assign Roles

```go
// Context needs actor ID for audit logging
ctx = rolekit.WithActorID(ctx, currentUserID)

// Assign a role
err := service.Assign(ctx, targetUserID, "admin", "organization", orgID)

// Revoke a role
err = service.Revoke(ctx, targetUserID, "admin", "organization", orgID)
```

### 3. Check Permissions (Service Level)

```go
// Check if user has a specific role
if service.Can(ctx, userID, "admin", "organization", orgID) {
    // User is admin
}

// Check if user has a specific permission
if service.HasPermission(ctx, userID, "files.upload", "project", projectID) {
    // User can upload files
}

// Get a checker for multiple checks
checker, _ := service.GetChecker(ctx, userID)
if checker.Can("owner", "organization", orgID) {
    // ...
}
if checker.HasPermission("members.invite", "organization", orgID) {
    // ...
}
```

### 4. HTTP Middleware

```go
// Create middleware
mw := rolekit.NewMiddleware(service,
    rolekit.WithUserIDExtractor(func(r *http.Request) string {
        return r.Context().Value("user_id").(string)
    }),
)

// Protect routes with role requirements
router.With(mw.RequireRole("admin", rolekit.ScopeFromParam("organization", "orgID"))).
    Post("/orgs/{orgID}/settings", updateSettingsHandler)

// Protect routes with permission requirements
router.With(mw.RequirePermission("files.upload", rolekit.ScopeFromParam("project", "projectID"))).
    Post("/projects/{projectID}/files", uploadHandler)

// Load checker for in-handler checks
router.With(mw.LoadChecker()).
    Get("/dashboard", dashboardHandler)
```

### 5. In-Handler Checks

```go
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
    checker := rolekit.FromContext(r.Context())

    if checker.Can("admin", "organization", orgID) {
        // Show admin features
    }

    if checker.HasPermission("billing.view", "organization", orgID) {
        // Show billing info
    }

    // Get all projects where user is editor
    projectIDs := checker.GetScopesWithRole("editor", "project")
}
```

## How Roles Work: Registry vs Database

RoleKit separates **role definition** (in-memory registry) from **role assignment** (database persistence). Here's how it works:

### 1. Registry Definition (In Memory)

First, you define what roles exist and what permissions they grant:

```go
registry := rolekit.NewRegistry()

// This lives in memory only - it's configuration
registry.DefineScope("project").
    Role("reader").Permissions("projects.read").
    Role("writer").Permissions("projects.read", "projects.write").
    Role("admin").Permissions("projects.read", "projects.write", "projects.delete").
        CanAssign("reader", "writer")  // Admin can assign reader/writer roles
```

**Registry stores:**

- Which roles exist for each scope type
- What permissions each role grants
- Who can assign which roles
- **This is configuration only** - no database writes

### 2. Role Assignment (Database Persistence)

When you assign a role to a user, it gets written to the database:

```go
// This creates a database record
err := service.Assign(ctx, "user123", "writer", "project", "proj-456")
```

**Database record created:**

```sql
INSERT INTO role_assignments (
    id, user_id, role, scope_type, scope_id, created_at, updated_at
) VALUES (
    'uuid-123', 'user123', 'writer', 'project', 'proj-456', NOW(), NOW()
);
```

### 3. Permission Checking (Runtime)

When checking permissions, RoleKit:

1. **Queries the database** for user's roles in that scope
2. **Looks up permissions** from the in-memory registry
3. **Evaluates if the requested permission is granted**

```go
// This triggers a database query + registry lookup
canWrite := service.HasPermission(ctx, "user123", "projects.write", "project", "proj-456")
```

**Behind the scenes:**

```sql
-- Step 1: Database query - get user's roles for this project
SELECT role FROM role_assignments
WHERE user_id = 'user123' AND scope_type = 'project' AND scope_id = 'proj-456';
-- Returns: ['writer']

-- Step 2: Registry lookup (in memory)
// writer role grants: ["projects.read", "projects.write"]

-- Step 3: Permission evaluation
// "projects.write" is in ["projects.read", "projects.write"] → TRUE
```

### 4. Multiple Roles Example

A user can have multiple roles in the same scope:

```go
service.Assign(ctx, "user123", "reader", "project", "proj-456")
service.Assign(ctx, "user123", "admin", "project", "proj-456")
```

**Database stores both records:**

```sql
('uuid-1', 'user123', 'reader', 'project', 'proj-456', ...),
('uuid-2', 'user123', 'admin',   'project', 'proj-456', ...);
```

**Permission evaluation:** Permissions are UNION of all roles:

- `reader`: ["projects.read"]
- `admin`: ["projects.read", "projects.write", "projects.delete"]
- **Combined effective permissions**: ["projects.read", "projects.write", "projects.delete"]

### 5. Audit Trail

Every assignment change is automatically logged:

```sql
INSERT INTO role_audit_log (
    timestamp, actor_id, action, target_user_id, role,
    scope_type, scope_id, previous_roles, new_roles
) VALUES (
    NOW(), 'admin-user', 'assigned', 'user123', 'writer',
    'project', 'proj-456', '[]', '["writer"]'
);
```

### Summary Flow

1. **Registry** (memory): Defines what roles exist and their permissions
2. **Assignment** (database): Stores which user has which role for which entity
3. **Check** (runtime): Queries DB → looks up registry → evaluates permission
4. **Audit** (database): Logs all changes for compliance

This separation gives you:

- **Flexibility**: Easy to change role definitions without database migrations
- **Persistence**: User assignments survive application restarts
- **Performance**: Registry lookups are in-memory, only role assignments hit the database
- **Auditability**: Complete history of all role changes

## Core Concepts

### Scopes

A scope is a tuple of `(EntityType, EntityID)` that represents a permission boundary:

```go
// Examples
("organization", "org_123")
("project", "proj_456")
("repository", "repo_789")
("team", "team_012")
```

### Scope-Specific Roles

The same role name can have different meanings in different scopes:

```go
// "admin" in organization has different permissions than "admin" in project
registry.DefineScope("organization").
    Role("admin").Permissions("members.*", "billing.*")

registry.DefineScope("project").
    Role("admin").Permissions("files.*", "settings.*")
```

### Permission Wildcards

RoleKit supports three types of wildcards:

| Pattern            | Matches                                             |
| ------------------ | --------------------------------------------------- |
| `*`                | All permissions                                     |
| `files.*`          | `files.read`, `files.write`, `files.delete`, etc.   |
| `*.read`           | `files.read`, `members.read`, `settings.read`, etc. |
| `files.metadata.*` | `files.metadata.read`, `files.metadata.write`, etc. |

### Multiple Roles per Scope

Users can have multiple roles in the same scope. Permissions are the **UNION** of all roles:

```go
// User has both "editor" and "reviewer" roles in project
// User gets permissions from BOTH roles
service.Assign(ctx, userID, "editor", "project", projectID)
service.Assign(ctx, userID, "reviewer", "project", projectID)
```

### Scope Wildcards (Super Users)

For super-users that need access to all entities of a type:

```go
// Admin on ALL projects
service.Assign(ctx, userID, "admin", "project", "*")

// Now this returns true for any project ID
service.Can(ctx, userID, "admin", "project", "any_project_id")
```

### Hierarchical Scopes

Define parent-child relationships between scopes:

```go
registry.DefineScope("project").
    ParentScope("organization")

// When creating a project, set its parent
service.SetScopeParent(ctx, "project", projectID, "organization", orgID)

// Query all projects in org where user has a role
projectIDs, _ := service.GetChildScopes(ctx, userID, "project", "organization", orgID)
```

### Role Assignment Permissions

Control who can assign which roles:

```go
registry.DefineScope("organization").
    Role("owner").
        Permissions("*").
        CanAssign("*").                    // Can assign any role
    Role("admin").
        Permissions("members.*").
        CanAssign("member", "viewer").     // Can only assign member or viewer
    Role("member").
        Permissions("read")                // Cannot assign any roles
```

## Middleware

### Scope Extractors

RoleKit provides several ways to extract scope from requests:

```go
// From URL parameter (chi, gorilla/mux style)
rolekit.ScopeFromParam("organization", "orgID")
// Route: /orgs/{orgID}/settings

// From query parameter
rolekit.ScopeFromQuery("project", "project_id")
// URL: /api/files?project_id=proj_123

// From header
rolekit.ScopeFromHeader("organization", "X-Organization-ID")

// From context
rolekit.ScopeFromContext("organization", "org_id")

// Static (for global resources)
rolekit.StaticScope("system", "global")
```

### Middleware Functions

```go
// Require specific role
mw.RequireRole("admin", extractor)

// Require any of multiple roles
mw.RequireAnyRole([]string{"admin", "owner"}, extractor)

// Require specific permission
mw.RequirePermission("files.upload", extractor)

// Require any of multiple permissions
mw.RequireAnyPermission([]string{"files.read", "files.write"}, extractor)

// Just load checker (no enforcement)
mw.LoadChecker()

// Inject audit context (IP, user agent, request ID)
mw.InjectAuditContext()
```

### Custom Error Handler

```go
mw := rolekit.NewMiddleware(service,
    rolekit.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
        if rolekit.IsUnauthorized(err) {
            w.WriteHeader(http.StatusForbidden)
            json.NewEncoder(w).Encode(map[string]string{
                "error": "You don't have permission to perform this action",
            })
            return
        }
        // Handle other errors...
    }),
)
```

**Note:** Add `"encoding/json"` to your imports for the error handler example.

## Error Handling

RoleKit uses dbkit's chainable error wrapping to provide detailed context about database operations. All errors include operation names, database context, and preserve original error types for classification.

### Error Types and Classification

```go
import (
    "errors"
    "fmt"
    "log"

    "github.com/fernandezvara/dbkit"
    "github.com/fernandezvara/rolekit"
)

err := service.Assign(ctx, userID, role, scopeType, scopeID)
if err != nil {
    // Quick checks with sentinel errors
    if dbkit.IsDuplicate(err) {
        // Handle duplicate role assignment
        return fmt.Errorf("user already has this role in scope")
    }
    if dbkit.IsNotFound(err) {
        // Handle not found scenarios
        return fmt.Errorf("scope or user not found")
    }
    if dbkit.IsForeignKey(err) {
        // Handle foreign key violations
        return fmt.Errorf("invalid scope reference")
    }
    if dbkit.IsRetryable(err) {
        // Handle retryable errors (serialization, deadlock)
        return fmt.Errorf("temporary conflict, please retry")
    }

    // Access rich error details
    var dbErr *dbkit.Error
    if errors.As(err, &dbErr) {
        fmt.Printf("Operation: %s\n", dbErr.Operation)
        fmt.Printf("Table: %s\n", dbErr.Table)
        fmt.Printf("Column: %s\n", dbErr.Column)
        fmt.Printf("Constraint: %s\n", dbErr.Constraint)
        fmt.Printf("Detail: %s\n", dbErr.Detail)
        fmt.Printf("Hint: %s\n", dbErr.Hint)
    }

    return err
}
```

### Common Error Scenarios

```go
// 1. Duplicate Role Assignment
err := service.Assign(ctx, "user123", "admin", "organization", "org1")
if dbkit.IsDuplicate(err) {
    // User already has admin role in this organization
    log.Printf("User %s already has role %s in organization %s", userID, role, orgID)
}

// 2. Scope Not Found
err := service.Assign(ctx, "user123", "admin", "organization", "nonexistent")
if dbkit.IsForeignKey(err) {
    // Organization doesn't exist
    return fmt.Errorf("organization not found")
}

// 3. Role Not Assigned (when revoking)
err := service.Revoke(ctx, "user123", "admin", "organization", "org1")
if dbkit.IsNotFound(err) {
    // User doesn't have this role to revoke
    return fmt.Errorf("user does not have this role")
}

// 4. Database Connection Issues
err := service.GetUserRoles(ctx, "user123")
if dbkit.IsConnection(err) {
    // Database is unavailable
    return fmt.Errorf("database temporarily unavailable")
}
```

### Error Wrapping Patterns

RoleKit wraps dbkit errors with additional context while preserving the original error information:

```go
// RoleKit adds context about the operation
type Error struct {
    Code       ErrorCode
    Message    string
    Scope      *ScopeInfo
    Role       string
    User       string
    Operation  string
    Err        error  // Original dbkit error
}

// Access all error information
var roleErr *rolekit.Error
if errors.As(err, &roleErr) {
    fmt.Printf("RoleKit Error: %s\n", roleErr.Message)
    fmt.Printf("Scope: %s:%s\n", roleErr.Scope.Type, roleErr.Scope.ID)
    fmt.Printf("Role: %s\n", roleErr.Role)
    fmt.Printf("User: %s\n", roleErr.User)

    // Access original dbkit error
    var dbErr *dbkit.Error
    if errors.As(roleErr.Err, &dbErr) {
        fmt.Printf("Database Error: %s\n", dbErr.Constraint)
    }
}
```

## Transactions

RoleKit provides transaction support for atomic operations. All role operations within a transaction are either committed together or rolled back if any operation fails.

### Basic Transactions

```go
import (
    "context"
    "log"

    "github.com/fernandezvara/rolekit"
)

// Execute multiple role operations atomically
err := service.Transaction(ctx, func(ctx context.Context) error {
    // Assign multiple roles - all succeed or all fail together
    if err := service.Assign(ctx, "user1", "admin", "organization", "org1"); err != nil {
        return err // This will cause a rollback
    }
    if err := service.Assign(ctx, "user1", "member", "project", "proj1"); err != nil {
        return err // This will cause a rollback
    }
    if err := service.Revoke(ctx, "user1", "viewer", "organization", "old_org"); err != nil {
        return err // This will cause a rollback
    }
    return nil // This will cause a commit
})

if err != nil {
    log.Printf("Transaction failed: %v", err)
} else {
    log.Printf("All role operations completed successfully")
}
```

### Read-Only Transactions

For operations that only read data and want to ensure consistency:

```go
err := service.ReadOnlyTransaction(ctx, func(ctx context.Context) error {
    // Read multiple data sources consistently
    roles, err := service.GetUserRoles(ctx, userID)
    if err != nil {
        return err
    }

    members, err := service.GetScopeMembers(ctx, "organization", orgID)
    if err != nil {
        return err
    }

    // Process the consistent data snapshot
    return processRoleData(roles, members)
})
```

### Transaction Options

Control transaction behavior with custom options:

```go
import "github.com/fernandezvara/dbkit"

// High isolation level for critical operations
err := service.TransactionWithOptions(ctx, dbkit.SerializableTxOptions(), func(ctx context.Context) error {
    // Critical role changes that require serializable isolation
    return service.Assign(ctx, userID, "super_admin", "system", "global")
})

// Read-only with specific isolation level
err := service.TransactionWithOptions(ctx, dbkit.ReadOnlyTxOptions(), func(ctx context.Context) error {
    // Consistent read operations
    roles, err := service.GetUserRoles(ctx, userID)
    return err
})
```

### Nested Transactions (Savepoints)

RoleKit supports nested transactions using savepoints:

```go
err := service.Transaction(ctx, func(ctx context.Context) error {
    // Outer transaction
    if err := service.Assign(ctx, "user1", "admin", "organization", "org1"); err != nil {
        return err
    }

    // Nested transaction (uses savepoint)
    err = service.Transaction(ctx, func(ctx context.Context) error {
        if err := service.Assign(ctx, "user2", "member", "organization", "org1"); err != nil {
            return err // Only rolls back the inner transaction
        }
        return nil
    })

    // If inner transaction fails, outer transaction continues
    // The first assignment (user1 as admin) is still committed

    return nil
})
```

### Common Transaction Patterns

#### 1. Bulk Role Assignment

```go
func assignMultipleRoles(service *rolekit.Service, userID string, assignments []roleAssignment) error {
    return service.Transaction(ctx, func(ctx context.Context) error {
        for _, assignment := range assignments {
            if err := service.Assign(ctx, userID, assignment.Role,
                assignment.ScopeType, assignment.ScopeID); err != nil {
                return fmt.Errorf("failed to assign %s: %w", assignment.Role, err)
            }
        }
        return nil
    })
}
```

#### 2. Role Transfer

```go
func transferRoles(service *rolekit.Service, fromUser, toUser, scopeType, scopeID string) error {
    return service.Transaction(ctx, func(ctx context.Context) error {
        // Get all roles of the source user
        roles, err := service.GetUserRoles(ctx, fromUser)
        if err != nil {
            return err
        }

        // Revoke all roles from source user
        if err := service.RevokeAll(ctx, fromUser, scopeType, scopeID); err != nil {
            return err
        }

        // Assign all roles to target user
        for _, assignment := range roles.GetAssignments(scopeType, scopeID) {
            if err := service.Assign(ctx, toUser, assignment.Role, scopeType, scopeID); err != nil {
                return err
            }
        }

        return nil
    })
}
```

#### 3. Conditional Role Assignment

```go
func assignWithConditions(service *rolekit.Service, userID, role, scopeType, scopeID string) error {
    return service.Transaction(ctx, func(ctx context.Context) error {
        // Check if user already has a conflicting role
        roles, err := service.GetUserRoles(ctx, userID)
        if err != nil {
            return err
        }

        if roles.HasRole("super_admin", scopeType, scopeID) {
            return fmt.Errorf("cannot assign %s: user already has super_admin role", role)
        }

        // Assign the new role
        return service.Assign(ctx, userID, role, scopeType, scopeID)
    })
}
```

### Transaction Error Handling

```go
err := service.Transaction(ctx, func(ctx context.Context) error {
    if err := service.Assign(ctx, userID, role, scopeType, scopeID); err != nil {
        // Check for specific error types
        if dbkit.IsDuplicate(err) {
            return fmt.Errorf("user already has this role")
        }
        if dbkit.IsForeignKey(err) {
            return fmt.Errorf("scope does not exist")
        }
        return err // Other errors will cause rollback
    }
    return nil
})

if err != nil {
    // Transaction was rolled back
    log.Printf("Transaction rolled back: %v", err)
}
```

## Health Monitoring

RoleKit provides database health monitoring capabilities to help you ensure your role management system is functioning properly.

### Basic Health Checks

```go
import (
    "context"
    "log"

    "github.com/fernandezvara/rolekit"
)

// Simple health check
if service.IsHealthy(ctx) {
    log.Println("Database is healthy")
} else {
    log.Println("Database is not healthy")
}

// Detailed health status
status := service.Health(ctx)
if status.Healthy {
    log.Printf("Database is healthy, latency: %v", status.Latency)
} else {
    log.Printf("Database is unhealthy: %s", status.Error)
    log.Printf("Pool stats: InUse=%d, Idle=%d", status.PoolStats.InUse, status.PoolStats.Idle)
}
```

### Connection Pool Monitoring

Monitor database connection pool statistics to detect connection exhaustion or performance issues:

```go
// Get connection pool statistics
stats := service.GetPoolStats()
log.Printf("Connection Pool Statistics:")
log.Printf("  Open Connections: %d", stats.OpenConnections)
log.Printf("  In Use: %d", stats.InUse)
log.Printf("  Idle: %d", stats.Idle)
log.Printf("  Wait Count: %d", stats.WaitCount)
log.Printf("  Wait Duration: %v", stats.WaitDuration)
log.Printf("  Max Idle Closed: %d", stats.MaxIdleClosed)
log.Printf("  Max Lifetime Closed: %d", stats.MaxLifetimeClosed)

// Alert on connection pool issues
if stats.WaitCount > 10 {
    log.Printf("WARNING: High connection wait count: %d", stats.WaitCount)
}
if stats.InUse > stats.OpenConnections*80/100 {
    log.Printf("WARNING: High connection usage: %d/%d", stats.InUse, stats.OpenConnections)
}
```

### Health Check Endpoints

Create HTTP endpoints for monitoring systems like Kubernetes, load balancers, or monitoring services:

```go
import (
    "encoding/json"
    "net/http"
)

// Simple health endpoint (returns 200 or 503)
func healthHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    if service.IsHealthy(ctx) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("Service Unavailable"))
    }
}

// Detailed health endpoint (returns JSON with full status)
func healthDetailedHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    status := service.Health(ctx)

    w.Header().Set("Content-Type", "application/json")
    if status.Healthy {
        w.WriteHeader(http.StatusOK)
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "healthy": status.Healthy,
        "latency": status.Latency.String(),
        "error":   status.Error,
        "pool_stats": map[string]interface{}{
            "open_connections": status.PoolStats.OpenConnections,
            "in_use":          status.PoolStats.InUse,
            "idle":             status.PoolStats.Idle,
            "wait_count":       status.PoolStats.WaitCount,
            "wait_duration":    status.PoolStats.WaitDuration.String(),
        },
    })
}

// Metrics endpoint for Prometheus-style monitoring
func metricsHandler(w http.ResponseWriter, r *http.Request) {
    stats := service.GetPoolStats()

    // Output in Prometheus metrics format
    fmt.Fprintf(w, "# HELP rolekit_db_connections_open Number of open database connections\n")
    fmt.Fprintf(w, "# TYPE rolekit_db_connections_open gauge\n")
    fmt.Fprintf(w, "rolekit_db_connections_open %d\n", stats.OpenConnections)

    fmt.Fprintf(w, "# HELP rolekit_db_connections_in_use Number of connections currently in use\n")
    fmt.Fprintf(w, "# TYPE rolekit_db_connections_in_use gauge\n")
    fmt.Fprintf(w, "rolekit_db_connections_in_use %d\n", stats.InUse)

    fmt.Fprintf(w, "# HELP rolekit_db_connections_idle Number of idle connections\n")
    fmt.Fprintf(w, "# TYPE rolekit_db_connections_idle gauge\n")
    fmt.Fprintf(w, "rolekit_db_connections_idle %d\n", stats.Idle)

    fmt.Fprintf(w, "# HELP rolekit_db_wait_count Number of connections waiting for a connection\n")
    fmt.Fprintf(w, "# TYPE rolekit_db_wait_count gauge\n")
    fmt.Fprintf(w, "rolekit_db_wait_count %d\n", stats.WaitCount)
}
```

### Custom Health Checks

Create custom health checks that include role-specific metrics:

```go
// Custom health check with role metrics
func customHealthCheck(service *rolekit.Service) error {
    ctx := context.Background()

    // Check database health
    if !service.IsHealthy(ctx) {
        return fmt.Errorf("database is not healthy")
    }

    // Check connection pool
    stats := service.GetPoolStats()
    if stats.WaitCount > 5 {
        return fmt.Errorf("high connection wait count: %d", stats.WaitCount)
    }

    // Check role assignment functionality
    testUserID := "health-check-user"
    testRole := "health-check-role"
    testScopeType := "system"
    testScopeID := "health-check"

    // Try to assign and revoke a test role
    if err := service.Assign(ctx, testUserID, testRole, testScopeType, testScopeID); err != nil {
        return fmt.Errorf("role assignment test failed: %w", err)
    }

    if err := service.Revoke(ctx, testUserID, testRole, testScopeType, testScopeID); err != nil {
        return fmt.Errorf("role revocation test failed: %w", err)
    }

    return nil
}

// Usage in health endpoint
func advancedHealthHandler(w http.ResponseWriter, r *http.Request) {
    if err := customHealthCheck(service); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "unhealthy",
            "error":  err.Error(),
        })
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "healthy",
    })
}
```

### Periodic Health Monitoring

Set up periodic health checks for proactive monitoring:

```go
import "time"

func startHealthMonitoring(service *rolekit.Service) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

            if !service.IsHealthy(ctx) {
                log.Printf("ALERT: Database is unhealthy")

                // Get detailed status for logging
                status := service.Health(ctx)
                log.Printf("Health status: %+v", status)

                // Get pool statistics
                stats := service.GetPoolStats()
                log.Printf("Pool stats: %+v", stats)

                // You could trigger alerts here (PagerDuty, Slack, etc.)
                // sendAlert("RoleKit Database Unhealthy", status.Error)
            }

            cancel()
        }
    }
}

// Start health monitoring in a goroutine
go startHealthMonitoring(service)
```

### Integration with Monitoring Systems

#### Kubernetes Readiness/Liveness Probes

```go
// Kubernetes liveness probe (restarts container if unhealthy)
func livenessProbeHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Simple check - if database is down, restart the container
    if service.IsHealthy(ctx) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("Unhealthy"))
    }
}

// Kubernetes readiness probe (stops traffic if not ready)
func readinessProbeHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // More comprehensive check - if not ready, stop sending traffic
    status := service.Health(ctx)

    if status.Healthy && status.PoolStats.WaitCount < 5 {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Ready"))
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("Not Ready"))
    }
}
```

#### Prometheus Integration

```go
// Prometheus metrics collector
type RoleKitCollector struct {
    service *rolekit.Service
}

func (c *RoleKitCollector) Describe(ch chan<- *prometheus.Desc) {
    // Describe your metrics here
}

func (c *RoleKitCollector) Collect(ch chan<- prometheus.Metric) {
    stats := c.service.GetPoolStats()

    ch <- prometheus.MustNewConstMetric(
        prometheus.NewDesc("rolekit_db_connections_open", "Number of open database connections", nil, nil),
        float64(stats.OpenConnections),
    )

    ch <- prometheus.MustNewConstMetric(
        prometheus.NewDesc("rolekit_db_connections_in_use", "Number of connections currently in use", nil, nil),
        float64(stats.InUse),
    )

    ch <- prometheus.MustNewConstMetric(
        prometheus.NewDesc("rolekit_db_connections_idle", "Number of idle connections", nil, nil),
        float64(stats.Idle),
    )

    ch <- prometheus.MustNewConstMetric(
        prometheus.NewDesc("rolekit_db_wait_count", "Number of connections waiting", nil, nil),
        float64(stats.WaitCount),
    )
}
```

### Health Check Best Practices

1. **Use appropriate timeouts** - Health checks should be fast (1-5 seconds max)
2. **Implement graceful degradation** - Continue serving if non-critical health checks fail
3. **Monitor connection pool metrics** - Watch for connection exhaustion
4. **Set up alerts** - Get notified before issues become critical
5. **Test health checks** - Ensure they work when the database is actually down
6. **Log health status** - Maintain a history of health check results

```go
// Example of a comprehensive health check with timeout
func comprehensiveHealthCheck(service *rolekit.Service) error {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    // Check database connectivity
    if err := service.Ping(ctx); err != nil {
        return fmt.Errorf("database ping failed: %w", err)
    }

    // Check connection pool health
    stats := service.GetPoolStats()
    if stats.WaitCount > 10 {
        return fmt.Errorf("connection pool overloaded: %d waiting", stats.WaitCount)
    }

    // Check basic functionality
    if !service.IsHealthy(ctx) {
        return fmt.Errorf("database health check failed")
    }

    return nil
}
```

## Performance Optimization

RoleKit provides optimized database operations using dbkit's helper functions for better performance and type safety.

### Bulk Operations

For multiple role assignments or revocations, use bulk operations to reduce database round trips:

```go
import (
    "github.com/fernandezvara/rolekit"
)

// Bulk assign multiple roles
assignments := []rolekit.RoleAssignment{
    {UserID: "user1", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
    {UserID: "user2", Role: "member", ScopeType: "organization", ScopeID: "org1"},
    {UserID: "user3", Role: "viewer", ScopeType: "project", ScopeID: "proj1"},
}
err := service.AssignMultiple(ctx, assignments)

// Bulk revoke multiple roles
revocations := []rolekit.RoleRevocation{
    {UserID: "user1", Role: "admin", ScopeType: "organization", ScopeID: "org1"},
    {UserID: "user2", Role: "member", ScopeType: "organization", ScopeID: "org1"},
}
err := service.RevokeMultiple(ctx, revocations)
```

### Efficient Existence Checks

Use `CheckExists` for simple existence checks instead of fetching full role data:

```go
// Efficient: Only checks if role exists
hasAdmin := service.CheckExists(ctx, "user1", "admin", "organization", "org1")

// Less efficient: Fetches all user roles
roles, _ := service.GetUserRoles(ctx, "user1")
hasAdmin := roles.HasRole("admin", "organization", "org1")
```

### Count Operations

Use count operations when you only need the number of records:

```go
// Count user's roles in a scope
count, err := service.CountRoles(ctx, "user1", "organization", "org1")
log.Printf("User has %d roles in org1", count)

// Count all role assignments in the system
total, err := service.CountAllRoles(ctx)
log.Printf("Total role assignments: %d", total)
```

### Performance Benefits

1. **Reduced Database Round Trips**: Bulk operations combine multiple operations into single database calls
2. **Type Safety**: All operations use Go's type system to prevent runtime errors
3. **Optimized Queries**: dbkit's helper functions generate optimized SQL queries
4. **Connection Pool Efficiency**: Fewer database connections are used for bulk operations
5. **Memory Efficiency**: Batch operations process data in chunks to avoid memory issues

### Performance Comparison

```go
// Before: Multiple individual operations (N database calls)
for _, assignment := range assignments {
    err := service.Assign(ctx, assignment.UserID, assignment.Role,
        assignment.ScopeType, assignment.ScopeID)
    if err != nil {
        return err
    }
}

// After: Single bulk operation (1 database call)
err := service.AssignMultiple(ctx, assignments)
```

### Batch Size Configuration

RoleKit uses dbkit's default batch size (100 records) for bulk operations. This can be adjusted for your specific use case:

```go
// The default batch size is optimized for most workloads
// For very large datasets, you might want to process in smaller chunks
const customBatchSize = 50

// RoleKit automatically handles batching internally
// You don't need to manually manage batch sizes
```

### Monitoring Performance

Use the health monitoring features to track performance:

```go
// Monitor connection pool usage
stats := service.GetPoolStats()
if stats.WaitCount > 0 {
    log.Printf("Database connection pressure detected: %d waiting", stats.WaitCount)
}

// Monitor operation performance
start := time.Now()
err := service.AssignMultiple(ctx, assignments)
duration := time.Since(start)
log.Printf("Bulk assignment of %d roles took %v", len(assignments), duration)
```

## Migration System

RoleKit provides a robust migration system with status tracking, checksum verification, and rollback capabilities using dbkit's migration features.

### Running Migrations

The recommended way to run migrations in production:

```go
import (
    "context"
    "log"

    "github.com/fernandezvara/dbkit"
    "github.com/fernandezvara/rolekit"
)

// Initialize database and service
db, _ := dbkit.New(dbkit.Config{URL: "postgres://..."})
registry := rolekit.NewRegistry()
service := rolekit.NewService(registry, db)

// Run all pending migrations
status, err := service.RunMigrations(ctx)
if err != nil {
    log.Fatalf("Migration failed: %v", err)
}

log.Printf("Migration status: %d applied, %d pending", status.Applied, status.Pending)
```

### Migration Status Tracking

Get detailed information about migration status:

```go
// Get current migration status
status, err := service.GetMigrationStatus(ctx)
if err != nil {
    log.Printf("Failed to get migration status: %v", err)
    return
}

// Display migration status
for _, migration := range status.Migrations {
    status := "Applied"
    if !migration.Applied {
        status = "Pending"
    }
    log.Printf("Migration %s: %s", migration.ID, status)
}

// Check if migrations are up to date
if status.Pending == 0 {
    log.Println("All migrations are up to date")
} else {
    log.Printf("There are %d pending migrations", status.Pending)
}
```

### Migration Checksum Verification

Verify migration integrity to detect unauthorized changes:

```go
// Verify all applied migrations have matching checksums
valid, err := service.VerifyMigrationChecksums(ctx)
if err != nil {
    log.Printf("Checksum verification failed: %v", err)
    return
}

if !valid {
    log.Printf("WARNING: Migration checksums do not match - potential tampering detected")
    // Take appropriate action (alert, rollback, etc.)
} else {
    log.Println("All migration checksums are valid")
}
```

### Migration Validation

Validate migrations before deployment:

```go
// Validate all migrations are properly formatted
err := service.ValidateMigrations()
if err != nil {
    log.Printf("Migration validation failed: %v", err)
    return
}

log.Println("All migrations are valid and ready for deployment")
```

### Migration Rollback

Rollback to a specific migration (requires manual rollback SQL):

```go
// Note: This requires manual rollback SQL to be defined
err := service.RollbackToMigration(ctx, "rolekit-010")
if err != nil {
    log.Printf("Rollback failed: %v", err)
    return
}

log.Printf("Successfully rolled back to migration rolekit-010")
```

### Migration Best Practices

#### 1. Use Descriptive Migration IDs

```go
// Good: Descriptive and chronological
{
    ID:          "rolekit-001",
    Description: "Create role_assignments table",
    SQL:         `CREATE TABLE IF NOT EXISTS role_assignments (...)`,
}

// Bad: Non-descriptive
{
    ID:          "migration1",
    Description: "Create table",
    SQL:         `CREATE TABLE role_assignments (...)`,
}
```

#### 2. Include IF EXISTS Statements

```go
// Good: Safe for repeated execution
SQL: `CREATE TABLE IF NOT EXISTS role_assignments (...)`
SQL: `CREATE INDEX IF NOT EXISTS idx_role_assignments_user ON role_assignments(user_id)`

// Bad: May fail on repeated execution
SQL: `CREATE TABLE role_assignments (...)`
SQL: `CREATE INDEX idx_role_assignments_user ON role_assignments(user_id)`
```

#### 3. Use Transactions for Complex Migrations

```go
// Complex migration with multiple steps
{
    ID:          "rolekit-050",
    Description: "Add audit logging and migrate existing data",
    SQL: `
        BEGIN;

        -- Create audit log table
        CREATE TABLE IF NOT EXISTS role_audit_log (...);

        -- Create indexes
        CREATE INDEX IF NOT EXISTS idx_role_audit_log_timestamp ON role_audit_log(timestamp DESC);

        -- Migrate existing data
        INSERT INTO role_audit_log (actor_id, action, target_user_id, role, scope_type, scope_id, timestamp)
        SELECT user_id, 'initial_import', user_id, role, scope_type, scope_id, NOW()
        FROM role_assignments;

        COMMIT;
    `,
}
```

#### 4. Test Migrations in Development

```go
// Test migration before production deployment
func TestMigration(t *testing.T) {
    // Use test database
    testDB, _ := dbkit.New(dbkit.Config{
        URL: "postgres://localhost:5432/rolekit_test",
    })

    // Create service with test database
    service := rolekit.NewService(registry, testDB)

    // Run migrations
    status, err := service.RunMigrations(context.Background())
    if err != nil {
        t.Fatalf("Migration failed: %v", err)
    }

    // Verify migrations applied
    if status.Pending > 0 {
        t.Errorf("Expected all migrations to be applied, but %d are pending", status.Pending)
    }

    // Verify checksums
    valid, err := service.VerifyMigrationChecksums(context.Background())
    if err != nil {
        t.Fatalf("Checksum verification failed: %v", err)
    }
    if !valid {
        t.Error("Migration checksums are invalid")
    }
}
```

### Migration Status Endpoints

Create HTTP endpoints for monitoring migration status:

```go
// Migration status endpoint
func migrationStatusHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    status, err := service.GetMigrationStatus(ctx)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]string{
            "error": err.Error(),
        })
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(status)
}

// Migration health endpoint
func migrationHealthHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Check if migrations are up to date
    status, err := service.GetMigrationStatus(ctx)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte("unhealthy"))
        return
    }

    // Verify checksums
    valid, err := service.VerifyMigrationChecksums(ctx)
    if err != nil || !valid {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("unhealthy"))
        return
    }

    if status.Pending > 0 {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("pending"))
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("healthy"))
}
```

### Production Deployment Checklist

Before deploying to production:

```go
func preDeploymentChecks(ctx context.Context, service *rolekit.Service) error {
    // 1. Validate migrations
    if err := service.ValidateMigrations(); err != nil {
        return fmt.Errorf("migration validation failed: %w", err)
    }

    // 2. Check current status
    status, err := service.GetMigrationStatus(ctx)
    if err != nil {
        return fmt.Errorf("failed to get migration status: %w", err)
    }

    // 3. Verify checksums of applied migrations
    if status.Applied > 0 {
        valid, err := service.VerifyMigrationChecksums(ctx)
        if err != nil {
            return fmt.Errorf("checksum verification failed: %w", err)
        }
        if !valid {
            return fmt.Errorf("migration checksums do not match - potential tampering")
        }
    }

    // 4. Dry run migrations (if supported)
    // This would be a custom implementation
    log.Printf("Pre-deployment checks passed. Ready to apply %d migrations.", status.Pending)

    return nil
}
```

## Connection Pool Management

RoleKit provides dynamic connection pool configuration and monitoring to optimize database performance based on workload requirements.

### Connection Pool Configuration

Configure connection pool settings for different workload patterns:

```go
import (
    "time"
    "github.com/fernandezvara/rolekit"
)

// Use default configuration (balanced performance)
config := rolekit.DefaultPoolConfig()
err := service.ConfigureConnectionPool(config)
if err != nil {
    log.Printf("Failed to configure connection pool: %v", err)
}

// High-performance configuration for busy applications
highPerfConfig := rolekit.HighPerformancePoolConfig()
err = service.ConfigureConnectionPool(highPerfConfig)
if err != nil {
    log.Printf("Failed to configure high-performance pool: %v", err)
}

// Low-resource configuration for constrained environments
lowResConfig := rolekit.LowResourcePoolConfig()
err = service.ConfigureConnectionPool(lowResConfig)
if err != nil {
    log.Printf("Failed to configure low-resource pool: %v", err)
}
```

### Custom Pool Configuration

Create custom connection pool settings for your specific needs:

```go
// Custom configuration
config := rolekit.PoolConfig{
    MaxOpenConnections:    50,    // Maximum concurrent connections
    MaxIdleConnections:    25,    // Idle connections to keep ready
    ConnectionMaxLifetime: time.Hour,    // Maximum connection lifetime
    ConnectionMaxIdleTime: 10 * time.Minute, // Maximum idle time before closing
}

err := service.ConfigureConnectionPool(config)
if err != nil {
    log.Printf("Failed to configure custom pool: %v", err)
}
```

### Dynamic Pool Optimization

Let RoleKit automatically optimize connection pool settings based on current usage patterns:

```go
// Automatically adjust pool settings based on usage
err := service.OptimizeConnectionPool()
if err != nil {
    log.Printf("Failed to optimize connection pool: %v", err)
}

// The optimization analyzes:
// - Connection wait times
// - Idle connection ratios
// - Wait count patterns
// - Usage trends
```

### Pool Monitoring

Monitor connection pool health and performance:

```go
// Get current pool statistics
stats := service.GetPoolStats()
log.Printf("Pool Statistics:")
log.Printf("  Open Connections: %d", stats.OpenConnections)
log.Printf("  In Use: %d", stats.InUse)
log.Printf("  Idle: %d", stats.Idle)
log.Printf("  Wait Count: %d", stats.WaitCount)
log.Printf("  Wait Duration: %v", stats.WaitDuration)

// Get current configuration
currentConfig, err := service.GetConnectionPoolConfig()
if err != nil {
    log.Printf("Failed to get pool config: %v", err)
    return
}

log.Printf("Current Configuration:")
log.Printf("  Max Open: %d", currentConfig.MaxOpenConnections)
log.Printf("  Max Idle: %d", currentConfig.MaxIdleConnections)
log.Printf("  Max Lifetime: %v", currentConfig.ConnectionMaxLifetime)
log.Printf("  Max Idle Time: %v", currentConfig.ConnectionMaxIdleTime)
```

### Pool Health Monitoring

Set up automated monitoring for connection pool health:

```go
func monitorConnectionPool(service *rolekit.Service) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            stats := service.GetPoolStats()

            // Alert on connection pressure
            if stats.WaitCount > 10 {
                log.Printf("ALERT: High connection wait count: %d", stats.WaitCount)
            }

            // Alert on long wait times
            if stats.WaitDuration > 500*time.Millisecond {
                log.Printf("ALERT: Long connection wait duration: %v", stats.WaitDuration)
            }

            // Alert on connection exhaustion
            if stats.InUse >= stats.OpenConnections*90/100 {
                log.Printf("ALERT: Connection pool nearly exhausted: %d/%d",
                    stats.InUse, stats.OpenConnections)
            }

            // Auto-optimize if needed
            if stats.WaitCount > 5 || stats.WaitDuration > 100*time.Millisecond {
                log.Printf("Auto-optimizing connection pool...")
                err := service.OptimizeConnectionPool()
                if err != nil {
                    log.Printf("Auto-optimization failed: %v", err)
                }
            }
        }
    }
}

// Start monitoring in a goroutine
go monitorConnectionPool(service)
```

### Environment-Specific Configurations

#### Development Environment

```go
// Development: Relaxed settings for debugging
devConfig := rolekit.PoolConfig{
    MaxOpenConnections:    10,
    MaxIdleConnections:    5,
    ConnectionMaxLifetime: 2 * time.Hour,
    ConnectionMaxIdleTime: 15 * time.Minute,
}
```

#### Production Environment

```go
// Production: Optimized for performance and stability
prodConfig := rolekit.PoolConfig{
    MaxOpenConnections:    100,
    MaxIdleConnections:    50,
    ConnectionMaxLifetime: 30 * time.Minute,
    ConnectionMaxIdleTime: 5 * time.Minute,
}
```

#### Testing Environment

```go
// Testing: Minimal resources
testConfig := rolekit.PoolConfig{
    MaxOpenConnections:    5,
    MaxIdleConnections:    2,
    ConnectionMaxLifetime: time.Hour,
    ConnectionMaxIdleTime: 10 * time.Minute,
}
```

### Connection Pool Endpoints

Create HTTP endpoints for monitoring connection pool status:

```go
// Connection pool status endpoint
func poolStatusHandler(w http.ResponseWriter, r *http.Request) {
    stats := service.GetPoolStats()
    config, err := service.GetConnectionPoolConfig()
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]string{
            "error": err.Error(),
        })
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "stats": stats,
        "config": config,
        "healthy": stats.WaitCount < 5 && stats.WaitDuration < 100*time.Millisecond,
    })
}

// Pool optimization endpoint
func optimizePoolHandler(w http.ResponseWriter, r *http.Request) {
    err := service.OptimizeConnectionPool()
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]string{
            "error": err.Error(),
        })
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "optimized",
    })
}
```

### Performance Tuning Guidelines

#### 1. Connection Pool Size

```go
// Rule of thumb: Set MaxOpenConnections to:
// - CPU cores * 2 for CPU-bound workloads
// - CPU cores * 4 for I/O-bound workloads
// - Consider your database server's max_connections limit

// Example for 8-core server with I/O-bound workload
config := rolekit.PoolConfig{
    MaxOpenConnections:    32, // 8 cores * 4
    MaxIdleConnections:    16, // Half of max open
}
```

#### 2. Connection Lifetime

```go
// Shorter lifetimes for high-traffic applications
highTrafficConfig := rolekit.PoolConfig{
    ConnectionMaxLifetime: 15 * time.Minute,
    ConnectionMaxIdleTime: 2 * time.Minute,
}

// Longer lifetimes for low-traffic applications
lowTrafficConfig := rolekit.PoolConfig{
    ConnectionMaxLifetime: 2 * time.Hour,
    ConnectionMaxIdleTime: 30 * time.Minute,
}
```

#### 3. Monitoring Thresholds

```go
// Recommended alert thresholds
const (
    MaxWaitCount        = 10    // Alert if > 10 connections waiting
    MaxWaitDuration     = 500 * time.Millisecond  // Alert if wait > 500ms
    MaxUtilization     = 0.9   // Alert if > 90% utilization
    MinIdleRatio       = 0.2   // Alert if idle < 20% of max
)
```

### Troubleshooting Connection Pool Issues

#### Common Problems and Solutions

```go
// Problem: Too many connection timeouts
// Solution: Increase pool size or optimize queries
if stats.WaitCount > 20 {
    config, _ := service.GetConnectionPoolConfig()
    config.MaxOpenConnections *= 2
    service.ConfigureConnectionPool(config)
}

// Problem: High idle connections
// Solution: Reduce max idle connections
if stats.Idle > stats.InUse*4 {
    config, _ := service.GetConnectionPoolConfig()
    config.MaxIdleConnections = stats.InUse * 2
    service.ConfigureConnectionPool(config)
}

// Problem: Connection exhaustion
// Solution: Implement connection pooling or circuit breaker
if stats.InUse >= stats.OpenConnections {
    log.Printf("Connection pool exhausted! Consider implementing rate limiting")
    // Implement circuit breaker or rate limiting
}
```

## Audit Log

All role changes are automatically logged:

```go
// Query audit log
filter := rolekit.NewAuditLogFilter().
    WithTargetUser(userID).
    WithScopeType("organization").
    WithSince(time.Now().AddDate(0, -1, 0)).  // Last month
    WithLimit(50)

logs, _ := service.GetAuditLog(ctx, filter)

for _, log := range logs {
    fmt.Printf("%s: %s %s role '%s' to user %s in %s:%s\n",
        log.Timestamp,
        log.ActorID,
        log.Action,        // "assigned" or "revoked"
        log.Role,
        log.TargetUserID,
        log.ScopeType,
        log.ScopeID,
    )
    fmt.Printf("  Previous roles: %v\n", log.PreviousRoles)
    fmt.Printf("  New roles: %v\n", log.NewRoles)
}
```

**Note:** Add `"fmt"` and `"time"` to your imports for the audit log example.

### Audit Entry Contents

| Field           | Description                                   |
| --------------- | --------------------------------------------- |
| `ActorID`       | Who performed the action                      |
| `Action`        | "assigned" or "revoked"                       |
| `TargetUserID`  | User whose role changed                       |
| `Role`          | Role that was assigned/revoked                |
| `ScopeType`     | Type of scope                                 |
| `ScopeID`       | ID of scope                                   |
| `ActorRoles`    | Actor's roles in this scope at time of action |
| `PreviousRoles` | Target's roles before change                  |
| `NewRoles`      | Target's roles after change                   |
| `IPAddress`     | Client IP address                             |
| `UserAgent`     | Client user agent                             |
| `RequestID`     | Request correlation ID                        |
| `Timestamp`     | When the action occurred                      |

## Database Schema

RoleKit creates these tables:

```sql
-- Role assignments
CREATE TABLE role_assignments (
    id UUID PRIMARY KEY,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    scope_type TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    parent_scope_type TEXT,
    parent_scope_id TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    UNIQUE(user_id, role, scope_type, scope_id)
);

-- Audit log
CREATE TABLE role_audit_log (
    id UUID PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    actor_id TEXT NOT NULL,
    action TEXT NOT NULL,
    target_user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    scope_type TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    actor_roles TEXT[],
    previous_roles TEXT[],
    new_roles TEXT[],
    ip_address TEXT,
    user_agent TEXT,
    request_id TEXT,
    metadata JSONB
);

-- Scope hierarchy
CREATE TABLE scope_hierarchy (
    id UUID PRIMARY KEY,
    scope_type TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    parent_scope_type TEXT NOT NULL,
    parent_scope_id TEXT NOT NULL,
    created_at TIMESTAMPTZ,
    UNIQUE(scope_type, scope_id, parent_scope_type, parent_scope_id)
);
```

## Integration with DBKit

RoleKit is designed to work seamlessly with DBKit:

```go
import (
    "context"
    "log"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/fernandezvara/dbkit"
    "github.com/fernandezvara/rolekit"
)

func main() {
    ctx := context.Background()

    // Setup database
    db, _ := dbkit.New(dbkit.Config{URL: "postgres://..."})
    defer db.Close()

    // Define roles
    registry := rolekit.NewRegistry()

    registry.DefineScope("organization").
        Role("owner").Permissions("*").CanAssign("*").
        Role("admin").Permissions("members.*", "settings.*").CanAssign("member").
        Role("member").Permissions("projects.*")

    registry.DefineScope("project").
        ParentScope("organization").
        Role("admin").Permissions("*").CanAssign("*").
        Role("editor").Permissions("files.*", "comments.*").
        Role("viewer").Permissions("files.read", "comments.read")

    // Create service
    roleService := rolekit.NewService(registry, db)
    _, err = db.Migrate(ctx, roleService.Migrations())
    if err != nil {
        log.Fatal(err)
    }

    // Create middleware
    roleMW := rolekit.NewMiddleware(roleService,
        rolekit.WithUserIDExtractor(getUserIDFromToken),
    )

    // Setup router
    r := chi.NewRouter()
    r.Use(roleMW.InjectAuditContext())

    // Organization routes
    r.Route("/orgs/{orgID}", func(r chi.Router) {
        orgScope := rolekit.ScopeFromParam("organization", "orgID")

        r.With(roleMW.RequireRole("admin", orgScope)).
            Put("/settings", updateOrgSettings)

        r.With(roleMW.RequirePermission("members.invite", orgScope)).
            Post("/members", inviteMember)
    })

    // Project routes
    r.Route("/projects/{projectID}", func(r chi.Router) {
        projScope := rolekit.ScopeFromParam("project", "projectID")

        r.With(roleMW.RequirePermission("files.read", projScope)).
            Get("/files", listFiles)

        r.With(roleMW.RequirePermission("files.upload", projScope)).
            Post("/files", uploadFile)
    })

    log.Fatal(http.ListenAndServe(":8080", r))
}

func getUserIDFromToken(r *http.Request) string {
    // Your JWT/session extraction logic
    return r.Context().Value("user_id").(string)
}
```

## License

MIT
