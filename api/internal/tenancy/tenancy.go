// Package tenancy carries the request-scoped tenant identity.
//
// Multi-tenancy is a first-class concern (ARCHITECTURE §5): the tenant_id is
// extracted from the JWT (admin) or resolved from /store/{username} (storefront)
// and injected here server-side. The repository layer REQUIRES a tenant_id on
// every query — see repo helpers — so no handler can accidentally skip scoping.
package tenancy

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type ctxKey struct{}

// Identity is the resolved caller principal.
type Identity struct {
	TenantID uuid.UUID
	Role     string // "seller" | "operator"
	Username string
}

var ErrNoTenant = errors.New("tenancy: no tenant in context")

func With(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

func From(ctx context.Context) (Identity, error) {
	id, ok := ctx.Value(ctxKey{}).(Identity)
	if !ok || id.TenantID == uuid.Nil {
		return Identity{}, ErrNoTenant
	}
	return id, nil
}

// MustTenant returns the tenant UUID or an error if absent — repositories call this.
func MustTenant(ctx context.Context) (uuid.UUID, error) {
	id, err := From(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	return id.TenantID, nil
}
