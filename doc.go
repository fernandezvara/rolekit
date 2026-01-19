// Package rolekit provides an entity-agnostic role and permission management system.
//
// RoleKit is designed to work with any entity type (organizations, projects, repositories,
// teams, workspaces, etc.) and provides a flexible, hierarchical permission system with
// full wildcard support.
//
// # Core Concepts
//
// Scope: A tuple of (EntityType, EntityID) representing a permission boundary.
// Examples: ("organization", "org_123"), ("project", "proj_456"), ("repository", "repo_789")
//
// Role: A named set of permissions defined per scope type. The same role name (e.g., "admin")
// can have different permissions in different scope types.
//
// Permission: A dot-separated string like "files.upload", "members.invite", etc.
// Supports wildcards: "*" (all), "files.*" (all file actions), "*.read" (all read actions).
//
// # Key Features
//
//   - Entity-agnostic: Works with any scope type you define
//   - Scope-specific roles: "admin" in organization ≠ "admin" in project
//   - Full wildcard support: *, resource.*, *.action
//   - Multiple roles per scope: User can have multiple roles, permissions are UNION
//   - Hierarchical scopes: Parent-child awareness for queries
//   - Detailed audit logging: Who, what, when, previous state, new state
//   - Token-agnostic: Only needs userID from context
//   - DBKit integration: Uses your existing database connection
//
// # Basic Usage
//
//	// 1. Define your roles (at application startup)
//	registry := rolekit.NewRegistry()
//
//	registry.DefineScope("organization").
//	    Role("owner").
//	        Permissions("*").
//	        CanAssign("*").
//	    Role("admin").
//	        Permissions("members.*", "settings.*", "billing.read").
//	        CanAssign("member", "viewer").
//	    Role("member").
//	        Permissions("projects.create", "projects.list").
//	    Role("viewer").
//	        Permissions("projects.list")
//
//	registry.DefineScope("project").
//	    ParentScope("organization").
//	    Role("admin").
//	        Permissions("*").
//	        CanAssign("*").
//	    Role("editor").
//	        Permissions("files.*", "comments.*").
//	    Role("viewer").
//	        Permissions("files.read", "comments.read")
//
//	// 2. Create the service
//	service := rolekit.NewService(registry, db)
//
//	// 3. Run migrations
//	service.Migrate(ctx)
//
//	// 4. Assign roles
//	service.Assign(ctx, userID, "admin", "organization", orgID)
//	service.Assign(ctx, userID, "editor", "project", projectID)
//
//	// 5. Check permissions (service level)
//	if service.Can(ctx, userID, "admin", "organization", orgID) {
//	    // User has admin role
//	}
//
//	if service.HasPermission(ctx, userID, "files.upload", "project", projectID) {
//	    // User can upload files
//	}
//
// # Middleware Usage
//
//	// Setup middleware
//	mw := rolekit.NewMiddleware(service)
//
//	// Protect routes
//	router.With(mw.RequireRole("admin", rolekit.ScopeFromParam("organization", "orgID"))).
//	    Post("/orgs/{orgID}/members", inviteHandler)
//
//	router.With(mw.RequirePermission("files.upload", rolekit.ScopeFromParam("project", "projectID"))).
//	    Post("/projects/{projectID}/files", uploadHandler)
//
// # Wildcard Permissions
//
// RoleKit supports three types of wildcards:
//
//   - "*" matches all permissions
//   - "resource.*" matches all actions on a resource (e.g., "files.*" matches "files.read", "files.write")
//   - "*.action" matches an action on all resources (e.g., "*.read" matches "files.read", "members.read")
//
// # Scope Wildcards
//
// For super-users, you can assign roles to all entities of a type:
//
//	service.Assign(ctx, userID, "admin", "project", "*")  // Admin on ALL projects
//
// # Audit Log
//
// All role changes are automatically logged with:
//   - Actor (who made the change)
//   - Target user
//   - Action (assigned, revoked)
//   - Role and scope
//   - Previous roles (for context)
//   - Timestamp
//   - Request metadata (IP, user agent, request ID)
package rolekit
