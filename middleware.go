package rolekit

import (
	"net/http"
)

// Middleware provides HTTP middleware for role and permission checking.
type Middleware struct {
	service      *Service
	getUserID    func(*http.Request) string
	errorHandler func(http.ResponseWriter, *http.Request, error)
}

// MiddlewareOption configures the Middleware.
type MiddlewareOption func(*Middleware)

// NewMiddleware creates a new Middleware instance.
//
// Example:
//
//	mw := rolekit.NewMiddleware(service,
//	    rolekit.WithUserIDExtractor(func(r *http.Request) string {
//	        return r.Context().Value("user_id").(string)
//	    }),
//	)
func NewMiddleware(service *Service, opts ...MiddlewareOption) *Middleware {
	m := &Middleware{
		service:      service,
		getUserID:    defaultGetUserID,
		errorHandler: defaultErrorHandler,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// WithUserIDExtractor sets a custom function to extract user ID from request.
func WithUserIDExtractor(fn func(*http.Request) string) MiddlewareOption {
	return func(m *Middleware) {
		m.getUserID = fn
	}
}

// WithErrorHandler sets a custom error handler for middleware.
func WithErrorHandler(fn func(http.ResponseWriter, *http.Request, error)) MiddlewareOption {
	return func(m *Middleware) {
		m.errorHandler = fn
	}
}

func defaultGetUserID(r *http.Request) string {
	return GetUserID(r.Context())
}

func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	if IsUnauthorized(err) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if IsInvalidScope(err) || IsInvalidRole(err) {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// ScopeExtractor extracts scope information from an HTTP request.
type ScopeExtractor func(*http.Request) (scopeType, scopeID string, err error)

// ScopeFromParam creates a ScopeExtractor that reads scope ID from URL parameters.
// Compatible with chi, gorilla/mux, and standard library patterns.
//
// Example:
//
//	// For route /orgs/{orgID}/projects/{projectID}
//	mw.RequireRole("admin", rolekit.ScopeFromParam("organization", "orgID"))
func ScopeFromParam(scopeType, paramName string) ScopeExtractor {
	return func(r *http.Request) (string, string, error) {
		// Try chi/go-chi style
		scopeID := r.PathValue(paramName)
		if scopeID == "" {
			// Try context (set by router middleware)
			if v := r.Context().Value(paramName); v != nil {
				if s, ok := v.(string); ok {
					scopeID = s
				}
			}
		}
		if scopeID == "" {
			return "", "", NewError(ErrInvalidScope, "scope ID not found in request").
				WithScope(scopeType, "")
		}
		return scopeType, scopeID, nil
	}
}

// ScopeFromQuery creates a ScopeExtractor that reads scope ID from query parameters.
//
// Example:
//
//	// For route /api/files?project_id=proj_123
//	mw.RequirePermission("files.read", rolekit.ScopeFromQuery("project", "project_id"))
func ScopeFromQuery(scopeType, queryParam string) ScopeExtractor {
	return func(r *http.Request) (string, string, error) {
		scopeID := r.URL.Query().Get(queryParam)
		if scopeID == "" {
			return "", "", NewError(ErrInvalidScope, "scope ID not found in query").
				WithScope(scopeType, "")
		}
		return scopeType, scopeID, nil
	}
}

// ScopeFromHeader creates a ScopeExtractor that reads scope ID from a header.
//
// Example:
//
//	// For header X-Organization-ID: org_123
//	mw.RequireRole("member", rolekit.ScopeFromHeader("organization", "X-Organization-ID"))
func ScopeFromHeader(scopeType, headerName string) ScopeExtractor {
	return func(r *http.Request) (string, string, error) {
		scopeID := r.Header.Get(headerName)
		if scopeID == "" {
			return "", "", NewError(ErrInvalidScope, "scope ID not found in header").
				WithScope(scopeType, "")
		}
		return scopeType, scopeID, nil
	}
}

// ScopeFromContext creates a ScopeExtractor that reads scope from context values.
//
// Example:
//
//	mw.RequireRole("admin", rolekit.ScopeFromContext("organization", "org_id"))
func ScopeFromContext(scopeType, contextKey string) ScopeExtractor {
	return func(r *http.Request) (string, string, error) {
		if v := r.Context().Value(contextKey); v != nil {
			if s, ok := v.(string); ok {
				return scopeType, s, nil
			}
		}
		return "", "", NewError(ErrInvalidScope, "scope ID not found in context").
			WithScope(scopeType, "")
	}
}

// StaticScope creates a ScopeExtractor that always returns the same scope.
// Useful for global resources.
//
// Example:
//
//	mw.RequireRole("super_admin", rolekit.StaticScope("system", "global"))
func StaticScope(scopeType, scopeID string) ScopeExtractor {
	return func(r *http.Request) (string, string, error) {
		return scopeType, scopeID, nil
	}
}

// RequireRole creates middleware that requires a specific role in a scope.
//
// Example:
//
//	router.With(mw.RequireRole("admin", rolekit.ScopeFromParam("organization", "orgID"))).
//	    Post("/orgs/{orgID}/settings", updateSettingsHandler)
func (m *Middleware) RequireRole(role string, extractor ScopeExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userID := m.getUserID(r)
			if userID == "" {
				m.errorHandler(w, r, ErrNoUserID)
				return
			}

			scopeType, scopeID, err := extractor(r)
			if err != nil {
				m.errorHandler(w, r, err)
				return
			}

			if !m.service.Can(ctx, userID, role, scopeType, scopeID) {
				m.errorHandler(w, r, NewError(ErrUnauthorized, "missing required role").
					WithScope(scopeType, scopeID).
					WithRole(role).
					WithUser(userID))
				return
			}

			// Add checker to context for use in handlers
			checker, err := m.service.GetChecker(ctx, userID)
			if err == nil {
				ctx = WithChecker(ctx, checker)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole creates middleware that requires any of the specified roles.
//
// Example:
//
//	router.With(mw.RequireAnyRole([]string{"admin", "owner"}, extractor)).
//	    Delete("/orgs/{orgID}", deleteOrgHandler)
func (m *Middleware) RequireAnyRole(roles []string, extractor ScopeExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userID := m.getUserID(r)
			if userID == "" {
				m.errorHandler(w, r, ErrNoUserID)
				return
			}

			scopeType, scopeID, err := extractor(r)
			if err != nil {
				m.errorHandler(w, r, err)
				return
			}

			if !m.service.HasAnyRole(ctx, userID, roles, scopeType, scopeID) {
				m.errorHandler(w, r, NewError(ErrUnauthorized, "missing required role").
					WithScope(scopeType, scopeID).
					WithUser(userID))
				return
			}

			checker, err := m.service.GetChecker(ctx, userID)
			if err == nil {
				ctx = WithChecker(ctx, checker)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission creates middleware that requires a specific permission.
//
// Example:
//
//	router.With(mw.RequirePermission("files.upload", rolekit.ScopeFromParam("project", "projectID"))).
//	    Post("/projects/{projectID}/files", uploadHandler)
func (m *Middleware) RequirePermission(permission string, extractor ScopeExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userID := m.getUserID(r)
			if userID == "" {
				m.errorHandler(w, r, ErrNoUserID)
				return
			}

			scopeType, scopeID, err := extractor(r)
			if err != nil {
				m.errorHandler(w, r, err)
				return
			}

			if !m.service.HasPermission(ctx, userID, permission, scopeType, scopeID) {
				m.errorHandler(w, r, NewError(ErrUnauthorized, "missing required permission").
					WithScope(scopeType, scopeID).
					WithUser(userID))
				return
			}

			checker, err := m.service.GetChecker(ctx, userID)
			if err == nil {
				ctx = WithChecker(ctx, checker)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission creates middleware that requires any of the specified permissions.
//
// Example:
//
//	router.With(mw.RequireAnyPermission([]string{"files.read", "files.write"}, extractor)).
//	    Get("/projects/{projectID}/files", listFilesHandler)
func (m *Middleware) RequireAnyPermission(permissions []string, extractor ScopeExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userID := m.getUserID(r)
			if userID == "" {
				m.errorHandler(w, r, ErrNoUserID)
				return
			}

			scopeType, scopeID, err := extractor(r)
			if err != nil {
				m.errorHandler(w, r, err)
				return
			}

			// Get checker and check permissions
			checker, err := m.service.GetChecker(ctx, userID)
			if err != nil {
				m.errorHandler(w, r, err)
				return
			}

			if !checker.HasAnyPermission(permissions, scopeType, scopeID) {
				m.errorHandler(w, r, NewError(ErrUnauthorized, "missing required permission").
					WithScope(scopeType, scopeID).
					WithUser(userID))
				return
			}

			ctx = WithChecker(ctx, checker)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// LoadChecker creates middleware that loads the user's Checker into context.
// Use this when you want to do permission checks in the handler rather than middleware.
//
// Example:
//
//	router.With(mw.LoadChecker()).Get("/dashboard", dashboardHandler)
//
//	func dashboardHandler(w http.ResponseWriter, r *http.Request) {
//	    checker := rolekit.FromContext(r.Context())
//	    if checker.Can("admin", "organization", orgID) {
//	        // Show admin features
//	    }
//	}
func (m *Middleware) LoadChecker() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userID := m.getUserID(r)
			if userID == "" {
				// No user, continue without checker
				next.ServeHTTP(w, r)
				return
			}

			checker, err := m.service.GetChecker(ctx, userID)
			if err != nil {
				// Log error but continue
				next.ServeHTTP(w, r)
				return
			}

			ctx = WithChecker(ctx, checker)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// InjectAuditContext creates middleware that extracts audit information from the request
// and adds it to the context for use in role assignment operations.
//
// Example:
//
//	router.Use(mw.InjectAuditContext())
func (m *Middleware) InjectAuditContext() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract IP address
			ip := r.Header.Get("X-Forwarded-For")
			if ip == "" {
				ip = r.Header.Get("X-Real-IP")
			}
			if ip == "" {
				ip = r.RemoteAddr
			}
			ctx = WithIPAddress(ctx, ip)

			// Extract User Agent
			ctx = WithUserAgent(ctx, r.UserAgent())

			// Extract Request ID (commonly set by other middleware)
			requestID := r.Header.Get("X-Request-ID")
			if requestID != "" {
				ctx = WithRequestID(ctx, requestID)
			}

			// Set actor ID from user ID if available
			userID := m.getUserID(r)
			if userID != "" {
				ctx = WithActorID(ctx, userID)
				ctx = WithUserID(ctx, userID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
