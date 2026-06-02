package auth

import (
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/mudahurus/api/internal/httpx"
	"github.com/mudahurus/api/internal/logging"
	"github.com/mudahurus/api/internal/tenancy"
)

// Middleware bundles the authentication, tenancy and RBAC guards (MH-104).
// Authenticated requests get a tenancy.Identity injected into the request
// context; every repository call derives tenant_id from it, so a handler
// physically cannot skip tenant scoping.
type Middleware struct{ tokens *TokenManager }

func NewMiddleware(tokens *TokenManager) *Middleware { return &Middleware{tokens: tokens} }

// Authenticated verifies the bearer access token and injects the identity.
func (m *Middleware) Authenticated() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authz := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(authz, "Bearer ") {
				return httpx.Unauthorized("missing bearer token")
			}
			claims, err := m.tokens.ParseAccess(strings.TrimPrefix(authz, "Bearer "))
			if err != nil {
				return httpx.Unauthorized("invalid or expired token")
			}
			tid, err := uuid.Parse(claims.TenantID)
			if err != nil {
				return httpx.Unauthorized("invalid token subject")
			}
			id := tenancy.Identity{TenantID: tid, Role: claims.Role, Username: claims.Username}
			ctx := tenancy.With(c.Request().Context(), id)
			// enrich the request logger with tenant_id (NFR observability)
			ctx = logging.Into(ctx, logging.From(ctx).With("tenant_id", tid.String()))
			c.SetRequest(c.Request().WithContext(ctx))
			c.Set("tenant_id", tid.String())
			return next(c)
		}
	}
}

// RequireRole enforces an RBAC role (e.g. "operator" for platform routes).
func (m *Middleware) RequireRole(role string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id, err := tenancy.From(c.Request().Context())
			if err != nil {
				return httpx.Unauthorized("authentication required")
			}
			if id.Role != role {
				return httpx.Forbidden("insufficient role")
			}
			return next(c)
		}
	}
}
