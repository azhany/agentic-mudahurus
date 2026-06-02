// Package customers implements customer CRUD + loyalty lookup (FR-4.x).
// PII fields (ic_no, dob, contact_no) are flagged; the RAG extractor excludes
// them from embeddings by default (PRD §6 Privacy).
package customers

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

type Customer struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"-"`
	FullName      string     `json:"full_name"`
	ICNo          string     `json:"ic_no"`  // PII
	DOB           *time.Time `json:"dob"`     // PII
	Email         string     `json:"email"`
	ContactNo     string     `json:"contact_no"` // PII
	MailingAddr   string     `json:"mailing_addr"`
	City          string     `json:"city"`
	Postcode      string     `json:"postcode"`
	State         string     `json:"state"`
	LoyaltyCode   string     `json:"customer_loyalty_code"`
	Type          string     `json:"type"`
	CreatedAt     time.Time  `json:"created_at"`
}

type Repo struct{ pool *db.Pool }

func NewRepo(pool *db.Pool) *Repo { return &Repo{pool: pool} }

const cols = `id, full_name, ic_no, dob, email, contact_no, mailing_addr, city, postcode, state, customer_loyalty_code, type, created_at`

func scan(row pgx.Row) (*Customer, error) {
	c := &Customer{}
	err := row.Scan(&c.ID, &c.FullName, &c.ICNo, &c.DOB, &c.Email, &c.ContactNo, &c.MailingAddr,
		&c.City, &c.Postcode, &c.State, &c.LoyaltyCode, &c.Type, &c.CreatedAt)
	return c, err
}

func (r *Repo) Create(ctx context.Context, c *Customer) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO customers (tenant_id, full_name, ic_no, dob, email, contact_no, mailing_addr, city, postcode, state, customer_loyalty_code, type)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id, created_at`,
		c.TenantID, c.FullName, c.ICNo, c.DOB, c.Email, c.ContactNo, c.MailingAddr, c.City, c.Postcode, c.State, c.LoyaltyCode, c.Type,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *Repo) Update(ctx context.Context, c *Customer) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE customers SET full_name=$3, ic_no=$4, dob=$5, email=$6, contact_no=$7,
			mailing_addr=$8, city=$9, postcode=$10, state=$11, customer_loyalty_code=$12, type=$13, updated_at=now()
		WHERE id=$1 AND tenant_id=$2`,
		c.ID, c.TenantID, c.FullName, c.ICNo, c.DOB, c.Email, c.ContactNo, c.MailingAddr, c.City, c.Postcode, c.State, c.LoyaltyCode, c.Type)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM customers WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) Get(ctx context.Context, tenantID, id uuid.UUID) (*Customer, error) {
	c, err := scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM customers WHERE id=$1 AND tenant_id=$2`, id, tenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

func (r *Repo) GetByLoyalty(ctx context.Context, tenantID uuid.UUID, code string) (*Customer, error) {
	c, err := scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM customers WHERE tenant_id=$1 AND customer_loyalty_code=$2`, tenantID, code))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

func (r *Repo) List(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]Customer, int, error) {
	args := []any{tenantID}
	where := "tenant_id=$1"
	if search != "" {
		args = append(args, "%"+search+"%")
		where += " AND (full_name ILIKE $2 OR email ILIKE $2 OR customer_loyalty_code ILIKE $2 OR contact_no ILIKE $2)"
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM customers WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	q := `SELECT ` + cols + ` FROM customers WHERE ` + where + ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Customer
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *c)
	}
	return out, total, rows.Err()
}

// ---- HTTP ----

type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) Routes(g *echo.Group) {
	g.GET("/customers", h.list)
	g.POST("/customers", h.create)
	g.GET("/customers/:id", h.get)
	g.PUT("/customers/:id", h.update)
	g.DELETE("/customers/:id", h.delete)
	g.GET("/customers/by-loyalty/:code", h.byLoyalty)
}

func tid(c echo.Context) uuid.UUID {
	id, _ := tenancy.From(c.Request().Context())
	return id.TenantID
}

func (h *Handler) list(c echo.Context) error {
	p := httpx.ParsePage(c)
	items, total, err := h.repo.List(c.Request().Context(), tid(c), p.Search, p.Limit, p.Offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, httpx.ListResponse{Records: items, QueryRecordCount: total, TotalRecordCount: total})
}

func bind(c echo.Context) (*Customer, error) {
	var in struct {
		FullName    string `json:"full_name"`
		ICNo        string `json:"ic_no"`
		DOB         string `json:"dob"`
		Email       string `json:"email"`
		ContactNo   string `json:"contact_no"`
		MailingAddr string `json:"mailing_addr"`
		City        string `json:"city"`
		Postcode    string `json:"postcode"`
		State       string `json:"state"`
		LoyaltyCode string `json:"customer_loyalty_code"`
		Type        string `json:"type"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return nil, err
	}
	if err := httpx.NewValidator().Require("full_name", in.FullName).Err(); err != nil {
		return nil, err
	}
	cust := &Customer{
		FullName: in.FullName, ICNo: in.ICNo, Email: in.Email, ContactNo: in.ContactNo,
		MailingAddr: in.MailingAddr, City: in.City, Postcode: in.Postcode, State: in.State,
		LoyaltyCode: in.LoyaltyCode, Type: in.Type,
	}
	if cust.Type == "" {
		cust.Type = "regular"
	}
	if in.DOB != "" {
		if t, err := time.Parse("2006-01-02", in.DOB); err == nil {
			cust.DOB = &t
		}
	}
	return cust, nil
}

func (h *Handler) create(c echo.Context) error {
	cust, err := bind(c)
	if err != nil {
		return err
	}
	cust.TenantID = tid(c)
	if err := h.repo.Create(c.Request().Context(), cust); err != nil {
		return err
	}
	// Auto-assign loyalty code if absent (legacy behaviour: user-customerid).
	if cust.LoyaltyCode == "" {
		cust.LoyaltyCode = cust.ID.String()[:8]
		cust.TenantID = tid(c)
		_ = h.repo.Update(c.Request().Context(), cust)
	}
	return c.JSON(http.StatusCreated, cust)
}

func (h *Handler) get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httpx.BadRequest("invalid id")
	}
	cust, err := h.repo.Get(c.Request().Context(), tid(c), id)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, cust)
}

func (h *Handler) byLoyalty(c echo.Context) error {
	cust, err := h.repo.GetByLoyalty(c.Request().Context(), tid(c), c.Param("code"))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, cust)
}

func (h *Handler) update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httpx.BadRequest("invalid id")
	}
	cust, err := bind(c)
	if err != nil {
		return err
	}
	cust.ID, cust.TenantID = id, tid(c)
	if err := h.repo.Update(c.Request().Context(), cust); err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, cust)
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
		return httpx.NotFound("customer not found")
	}
	return err
}
