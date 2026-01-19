package rolekit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMiddlewareNewMiddleware tests the middleware constructor
func TestMiddlewareNewMiddleware(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	service := &Service{registry: registry}

	// Test with default options
	mw := NewMiddleware(service)
	require.NotNil(t, mw)
	assert.Equal(t, service, mw.service)
	assert.NotNil(t, mw.getUserID)
	assert.NotNil(t, mw.errorHandler)

	// Test with custom options
	customUserID := func(r *http.Request) string { return "custom-user" }
	customErrorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusTeapot)
	}

	mw2 := NewMiddleware(service,
		WithUserIDExtractor(customUserID),
		WithErrorHandler(customErrorHandler),
	)
	// Test that custom functions are set by checking behavior
	req := httptest.NewRequest("GET", "/", nil)
	assert.Equal(t, "custom-user", mw2.getUserID(req))

	w := httptest.NewRecorder()
	mw2.errorHandler(w, req, nil)
	assert.Equal(t, http.StatusTeapot, w.Code)
}

// TestMiddlewareDefaultGetUserID tests the default user ID extractor
func TestMiddlewareDefaultGetUserID(t *testing.T) {
	// Test with user ID in context
	ctx := WithUserID(context.Background(), "test-user")
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(ctx)

	userID := defaultGetUserID(req)
	assert.Equal(t, "test-user", userID)

	// Test without user ID in context
	req = httptest.NewRequest("GET", "/", nil)
	userID = defaultGetUserID(req)
	assert.Empty(t, userID)
}

// TestMiddlewareDefaultErrorHandler tests the default error handler
func TestMiddlewareDefaultErrorHandler(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Unauthorized error",
			err:            NewError(ErrUnauthorized, "access denied"),
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Forbidden\n",
		},
		{
			name:           "Invalid scope error",
			err:            NewError(ErrInvalidScope, "invalid scope"),
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Bad Request\n",
		},
		{
			name:           "Invalid role error",
			err:            NewError(ErrInvalidRole, "invalid role"),
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Bad Request\n",
		},
		{
			name:           "Generic error",
			err:            NewError(ErrDatabaseError, "database error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal Server Error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)

			defaultErrorHandler(w, req, tt.err)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}

// TestMiddlewareScopeExtractors tests all scope extractor functions
func TestMiddlewareScopeExtractors(t *testing.T) {
	t.Run("StaticScope", func(t *testing.T) {
		extractor := StaticScope("organization", "org123")

		req := httptest.NewRequest("GET", "/", nil)
		scopeType, scopeID, err := extractor(req)

		assert.NoError(t, err)
		assert.Equal(t, "organization", scopeType)
		assert.Equal(t, "org123", scopeID)
	})

	t.Run("ScopeFromQuery", func(t *testing.T) {
		tests := []struct {
			name        string
			scopeType   string
			queryParam  string
			url         string
			expectError bool
			expectedID  string
		}{
			{
				name:        "Scope from query parameter",
				scopeType:   "organization",
				queryParam:  "orgID",
				url:         "/?orgID=org123",
				expectError: false,
				expectedID:  "org123",
			},
			{
				name:        "Missing query parameter",
				scopeType:   "organization",
				queryParam:  "orgID",
				url:         "/",
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", tt.url, nil)

				extractor := ScopeFromQuery(tt.scopeType, tt.queryParam)
				scopeType, scopeID, err := extractor(req)

				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.scopeType, scopeType)
					assert.Equal(t, tt.expectedID, scopeID)
				}
			})
		}
	})

	t.Run("ScopeFromHeader", func(t *testing.T) {
		tests := []struct {
			name        string
			scopeType   string
			headerName  string
			setupHeader func(*http.Request)
			expectError bool
			expectedID  string
		}{
			{
				name:       "Scope from header",
				scopeType:  "organization",
				headerName: "X-Org-ID",
				setupHeader: func(req *http.Request) {
					req.Header.Set("X-Org-ID", "org123")
				},
				expectError: false,
				expectedID:  "org123",
			},
			{
				name:        "Missing header",
				scopeType:   "organization",
				headerName:  "X-Org-ID",
				setupHeader: func(req *http.Request) {},
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", "/", nil)
				tt.setupHeader(req)

				extractor := ScopeFromHeader(tt.scopeType, tt.headerName)
				scopeType, scopeID, err := extractor(req)

				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.scopeType, scopeType)
					assert.Equal(t, tt.expectedID, scopeID)
				}
			})
		}
	})

	t.Run("ScopeFromContext", func(t *testing.T) {
		tests := []struct {
			name         string
			scopeType    string
			contextKey   string
			setupContext func(context.Context) context.Context
			expectError  bool
			expectedID   string
		}{
			{
				name:       "Scope from context",
				scopeType:  "organization",
				contextKey: "orgID",
				setupContext: func(ctx context.Context) context.Context {
					//nolint:staticcheck // Using string key for context
					return context.WithValue(ctx, "orgID", "org123")
				},
				expectError: false,
				expectedID:  "org123",
			},
			{
				name:       "Missing context value",
				scopeType:  "organization",
				contextKey: "orgID",
				setupContext: func(ctx context.Context) context.Context {
					return ctx
				},
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", "/", nil)
				req = req.WithContext(tt.setupContext(req.Context()))

				extractor := ScopeFromContext(tt.scopeType, tt.contextKey)
				scopeType, scopeID, err := extractor(req)

				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.scopeType, scopeType)
					assert.Equal(t, tt.expectedID, scopeID)
				}
			})
		}
	})
}

// TestMiddlewareErrorPaths tests error handling paths in middleware
func TestMiddlewareErrorPaths(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	service := &Service{registry: registry}
	mw := NewMiddleware(service)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	t.Run("RequireRole without user ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		// Don't set user ID

		w := httptest.NewRecorder()
		handler := mw.RequireRole("admin", StaticScope("organization", "org123"))(nextHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("RequireRole with invalid scope", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req = req.WithContext(WithUserID(req.Context(), "user123"))

		w := httptest.NewRecorder()
		// Use a scope extractor that returns an error
		errorExtractor := func(r *http.Request) (string, string, error) {
			return "", "", NewError(ErrInvalidScope, "invalid scope")
		}
		handler := mw.RequireRole("admin", errorExtractor)(nextHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("LoadChecker without user ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		// Don't set user ID

		w := httptest.NewRecorder()
		handler := mw.LoadChecker()(nextHandler)
		handler.ServeHTTP(w, req)

		// Should continue without checker
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestMiddlewareInjectAuditContext tests the audit context injection middleware
func TestMiddlewareInjectAuditContext(t *testing.T) {
	registry := NewRegistry()
	registry.DefineScope("organization").Role("admin")

	service := &Service{registry: registry}
	mw := NewMiddleware(service)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if audit context is available
		auditCtx := GetAuditContext(r.Context())
		require.NotNil(t, auditCtx)
		assert.Equal(t, "user123", auditCtx.ActorID)
		assert.Equal(t, "192.168.1.1", auditCtx.IPAddress)
		assert.Equal(t, "test-agent", auditCtx.UserAgent)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(WithUserID(req.Context(), "user123"))
	req = req.WithContext(WithActorID(req.Context(), "user123"))

	// Add IP and User-Agent to request
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	req.Header.Set("User-Agent", "test-agent")

	w := httptest.NewRecorder()

	handler := mw.InjectAuditContext()(nextHandler)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
