// Package storefront serves the public, read-only customer storefront (FR-6.x)
// plus guest checkout (FR-3.1). The tenant is resolved from /store/{username}
// server-side; only status='active' products are exposed. No auth required.
package storefront

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/mudahurus/api/internal/auth"
	"github.com/mudahurus/api/internal/db"
	"github.com/mudahurus/api/internal/httpx"
	"github.com/mudahurus/api/internal/orders"
	"github.com/mudahurus/api/internal/storage"
	"github.com/mudahurus/api/internal/tenancy"
)

type StoreProduct struct {
	ID          uuid.UUID `json:"id"`
	SKU         string    `json:"sku"`
	ProductName string    `json:"product_name"`
	Description string    `json:"description"`
	UnitPrice   float64   `json:"unit_price"`
	URLSlug     string    `json:"url_slug"`
	ImageURL    string    `json:"image_url,omitempty"`
	Category    string    `json:"category,omitempty"`
}

type Handler struct {
	pool   *db.Pool
	auth   *auth.Store
	orders *orders.Service
	store  storage.Storage
}

func NewHandler(pool *db.Pool, authStore *auth.Store, ord *orders.Service, store storage.Storage) *Handler {
	return &Handler{pool: pool, auth: authStore, orders: ord, store: store}
}

// Routes mounts the public storefront API (rate limiting applied by caller).
func (h *Handler) Routes(g *echo.Group) {
	g.GET("/store/:username", h.landing)
	g.GET("/store/:username/search", h.search)
	g.GET("/store/:username/products/:id", h.productDetail)
	g.POST("/store/:username/checkout", h.checkout)
	g.GET("/store/:username/get_product_by_sku", h.productBySKU) // legacy parity
}

func (h *Handler) resolveTenant(c echo.Context) (*auth.Tenant, error) {
	t, err := h.auth.TenantByUsername(c.Request().Context(), c.Param("username"))
	if err != nil {
		return nil, httpx.NotFound("store not found")
	}
	return t, nil
}

func (h *Handler) signImage(ctx context.Context, key string) string {
	if key == "" {
		return ""
	}
	if url, err := h.store.SignedGetURL(ctx, key, 3600); err == nil {
		return url
	}
	return ""
}

func (h *Handler) landing(c echo.Context) error {
	t, err := h.resolveTenant(c)
	if err != nil {
		return err
	}
	ctx := c.Request().Context()
	rows, err := h.pool.Query(ctx, `
		SELECT p.id, p.sku, p.product_name, p.description, p.unit_price, p.url_slug, p.image_key, coalesce(c.name,'')
		FROM products p LEFT JOIN categories c ON c.id=p.category_id
		WHERE p.tenant_id=$1 AND p.status='active' ORDER BY p.created_at DESC LIMIT 200`, t.ID)
	if err != nil {
		return err
	}
	products, err := scanStoreProducts(ctx, h, rows)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{
		"store": map[string]string{
			"username":   t.Username,
			"store_name": storeName(t),
		},
		"products": products,
	})
}

func (h *Handler) search(c echo.Context) error {
	t, err := h.resolveTenant(c)
	if err != nil {
		return err
	}
	q := c.QueryParam("q")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	ctx := c.Request().Context()
	// Postgres full-text search scoped to tenant + active (MH-402); falls back
	// to ILIKE for very short / non-lexeme queries.
	rows, err := h.pool.Query(ctx, `
		SELECT p.id, p.sku, p.product_name, p.description, p.unit_price, p.url_slug, p.image_key, coalesce(c.name,'')
		FROM products p LEFT JOIN categories c ON c.id=p.category_id
		WHERE p.tenant_id=$1 AND p.status='active'
		  AND ($2 = '' OR p.search_tsv @@ plainto_tsquery('simple',$2) OR p.product_name ILIKE '%'||$2||'%')
		ORDER BY ts_rank(p.search_tsv, plainto_tsquery('simple',$2)) DESC, p.created_at DESC
		LIMIT $3 OFFSET $4`, t.ID, q, limit, offset)
	if err != nil {
		return err
	}
	products, err := scanStoreProducts(ctx, h, rows)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"query": q, "products": products})
}

func (h *Handler) productDetail(c echo.Context) error {
	t, err := h.resolveTenant(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httpx.BadRequest("invalid product id")
	}
	ctx := c.Request().Context()
	p := StoreProduct{}
	var imageKey string
	err = h.pool.QueryRow(ctx, `
		SELECT p.id, p.sku, p.product_name, p.description, p.unit_price, p.url_slug, p.image_key, coalesce(c.name,'')
		FROM products p LEFT JOIN categories c ON c.id=p.category_id
		WHERE p.tenant_id=$1 AND p.id=$2 AND p.status='active'`, t.ID, id).
		Scan(&p.ID, &p.SKU, &p.ProductName, &p.Description, &p.UnitPrice, &p.URLSlug, &imageKey, &p.Category)
	if err == pgx.ErrNoRows {
		return httpx.NotFound("product not found")
	}
	if err != nil {
		return err
	}
	p.ImageURL = h.signImage(ctx, imageKey)
	return c.JSON(http.StatusOK, p)
}

func (h *Handler) productBySKU(c echo.Context) error {
	t, err := h.resolveTenant(c)
	if err != nil {
		return err
	}
	sku := c.QueryParam("sku")
	ctx := c.Request().Context()
	p := StoreProduct{}
	var imageKey string
	err = h.pool.QueryRow(ctx, `
		SELECT p.id, p.sku, p.product_name, p.description, p.unit_price, p.url_slug, p.image_key, coalesce(c.name,'')
		FROM products p LEFT JOIN categories c ON c.id=p.category_id
		WHERE p.tenant_id=$1 AND lower(p.sku)=lower($2) AND p.status='active'`, t.ID, sku).
		Scan(&p.ID, &p.SKU, &p.ProductName, &p.Description, &p.UnitPrice, &p.URLSlug, &imageKey, &p.Category)
	if err == pgx.ErrNoRows {
		return httpx.NotFound("product not found")
	}
	if err != nil {
		return err
	}
	p.ImageURL = h.signImage(ctx, imageKey)
	return c.JSON(http.StatusOK, p)
}

// checkout places a guest order against the resolved store (FR-3.1, MH-403).
// The tenant is injected server-side into the context so orders.Service scopes
// correctly — never trusted from the client body.
func (h *Handler) checkout(c echo.Context) error {
	t, err := h.resolveTenant(c)
	if err != nil {
		return err
	}
	var in orders.AdminInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	ctx := tenancy.With(c.Request().Context(), tenancy.Identity{TenantID: t.ID, Role: "seller", Username: t.Username})
	o, err := h.orders.GuestCheckout(ctx, t.ID, in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, map[string]any{
		"order_id":    o.ID,
		"status":      o.Status,
		"total_price": o.TotalPrice,
		"invoice_url": "/invoice/" + o.ID.String(),
	})
}

func scanStoreProducts(ctx context.Context, h *Handler, rows pgx.Rows) ([]StoreProduct, error) {
	defer rows.Close()
	var out []StoreProduct
	for rows.Next() {
		var p StoreProduct
		var imageKey string
		if err := rows.Scan(&p.ID, &p.SKU, &p.ProductName, &p.Description, &p.UnitPrice, &p.URLSlug, &imageKey, &p.Category); err != nil {
			return nil, err
		}
		p.ImageURL = h.signImage(ctx, imageKey)
		out = append(out, p)
	}
	return out, rows.Err()
}

func storeName(t *auth.Tenant) string {
	if t.StoreName != "" {
		return t.StoreName
	}
	return t.Username
}
