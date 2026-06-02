// Package coupons implements coupon CRUD (FR-5.1).
package coupons

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/mudahurus/api/internal/db"
	"github.com/mudahurus/api/internal/httpx"
	"github.com/mudahurus/api/internal/tenancy"
)

var ErrNotFound = errors.New("not found")

type Coupon struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"-"`
	ProductID   *uuid.UUID `json:"product_id"`
	Campaign    string     `json:"campaign"`
	Description string     `json:"description"`
	ExpiredDate *time.Time `json:"expired_date"`
	CreatedAt   time.Time  `json:"created_at"`
	Expired     bool       `json:"expired"`
}

type Repo struct{ pool *db.Pool }

func NewRepo(pool *db.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) Create(ctx context.Context, c *Coupon) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO coupons (tenant_id, product_id, campaign, description, expired_date)
		VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at`,
		c.TenantID, c.ProductID, c.Campaign, c.Description, c.ExpiredDate,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *Repo) Update(ctx context.Context, c *Coupon) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE coupons SET product_id=$3, campaign=$4, description=$5, expired_date=$6, updated_at=now()
		WHERE id=$1 AND tenant_id=$2`,
		c.ID, c.TenantID, c.ProductID, c.Campaign, c.Description, c.ExpiredDate)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM coupons WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) Get(ctx context.Context, tenantID, id uuid.UUID) (*Coupon, error) {
	c := &Coupon{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, product_id, campaign, description, expired_date, created_at FROM coupons WHERE id=$1 AND tenant_id=$2`,
		id, tenantID).Scan(&c.ID, &c.ProductID, &c.Campaign, &c.Description, &c.ExpiredDate, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err == nil {
		c.Expired = c.ExpiredDate != nil && c.ExpiredDate.Before(time.Now())
	}
	return c, err
}

func (r *Repo) List(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]Coupon, int, error) {
	args := []any{tenantID}
	where := "tenant_id=$1"
	if search != "" {
		args = append(args, "%"+search+"%")
		where += " AND (campaign ILIKE $2 OR description ILIKE $2)"
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM coupons WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	q := `SELECT id, product_id, campaign, description, expired_date, created_at FROM coupons WHERE ` + where +
		` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	now := time.Now()
	var out []Coupon
	for rows.Next() {
		var c Coupon
		if err := rows.Scan(&c.ID, &c.ProductID, &c.Campaign, &c.Description, &c.ExpiredDate, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		c.Expired = c.ExpiredDate != nil && c.ExpiredDate.Before(now)
		out = append(out, c)
	}
	return out, total, rows.Err()
}

// ---- HTTP ----

type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) Routes(g *echo.Group) {
	g.GET("/coupons", h.list)
	g.POST("/coupons", h.create)
	g.GET("/coupons/:id", h.get)
	g.PUT("/coupons/:id", h.update)
	g.DELETE("/coupons/:id", h.delete)
}

func tid(c echo.Context) uuid.UUID {
	id, _ := tenancy.From(c.Request().Context())
	return id.TenantID
}

func bind(c echo.Context) (*Coupon, error) {
	var in struct {
		ProductID   string `json:"product_id"`
		Campaign    string `json:"campaign"`
		Description string `json:"description"`
		ExpiredDate string `json:"expired_date"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return nil, err
	}
	if err := httpx.NewValidator().Require("campaign", in.Campaign).Err(); err != nil {
		return nil, err
	}
	cp := &Coupon{Campaign: in.Campaign, Description: in.Description}
	if in.ProductID != "" {
		pid, err := uuid.Parse(in.ProductID)
		if err != nil {
			return nil, httpx.BadRequest("invalid product_id")
		}
		cp.ProductID = &pid
	}
	if in.ExpiredDate != "" {
		for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02 15:04:05"} {
			if t, err := time.Parse(layout, in.ExpiredDate); err == nil {
				cp.ExpiredDate = &t
				break
			}
		}
	}
	return cp, nil
}

func (h *Handler) list(c echo.Context) error {
	p := httpx.ParsePage(c)
	items, total, err := h.repo.List(c.Request().Context(), tid(c), p.Search, p.Limit, p.Offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, httpx.ListResponse{Records: items, QueryRecordCount: total, TotalRecordCount: total})
}

func (h *Handler) create(c echo.Context) error {
	cp, err := bind(c)
	if err != nil {
		return err
	}
	cp.TenantID = tid(c)
	if err := h.repo.Create(c.Request().Context(), cp); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, cp)
}

func (h *Handler) get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httpx.BadRequest("invalid id")
	}
	cp, err := h.repo.Get(c.Request().Context(), tid(c), id)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, cp)
}

func (h *Handler) update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httpx.BadRequest("invalid id")
	}
	cp, err := bind(c)
	if err != nil {
		return err
	}
	cp.ID, cp.TenantID = id, tid(c)
	if err := h.repo.Update(c.Request().Context(), cp); err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, cp)
}

func (h *Handler) delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httpx.BadRequest("invalid id")
	}
	if err := h.repo.Delete(c.Request().Context(), tid(c), id); err != nil {
		return mapErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func mapErr(err error) error {
	if err == ErrNotFound {
		return httpx.NotFound("coupon not found")
	}
	return err
}
