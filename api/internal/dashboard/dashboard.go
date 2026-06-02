// Package dashboard exposes seller KPI counts (FR-7.1).
package dashboard

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/mudahurus/api/internal/db"
	"github.com/mudahurus/api/internal/tenancy"
)

type Counts struct {
	Orders        int `json:"orders"`
	PendingOrders int `json:"pending_orders"`
	Products      int `json:"products"`
	Customers     int `json:"customers"`
	ShippedOrders int `json:"shipped_orders"`
}

type Handler struct{ pool *db.Pool }

func NewHandler(pool *db.Pool) *Handler { return &Handler{pool: pool} }

func (h *Handler) Routes(g *echo.Group) {
	g.GET("/dashboard/counts", h.counts)
}

func (h *Handler) counts(c echo.Context) error {
	id, _ := tenancy.From(c.Request().Context())
	out, err := h.load(c.Request().Context(), id.TenantID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, out)
}

func (h *Handler) load(ctx context.Context, tenantID uuid.UUID) (*Counts, error) {
	out := &Counts{}
	err := h.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM orders    WHERE tenant_id=$1),
			(SELECT count(*) FROM orders    WHERE tenant_id=$1 AND status='pending'),
			(SELECT count(*) FROM products  WHERE tenant_id=$1),
			(SELECT count(*) FROM customers WHERE tenant_id=$1),
			(SELECT count(*) FROM orders    WHERE tenant_id=$1 AND status='shipped')`,
		tenantID,
	).Scan(&out.Orders, &out.PendingOrders, &out.Products, &out.Customers, &out.ShippedOrders)
	return out, err
}
