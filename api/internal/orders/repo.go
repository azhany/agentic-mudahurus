// Package orders implements the normalized order money-flow:
// orders + order_items + payments (FR-3.x). All queries tenant-scoped.
package orders

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mudahurus/api/internal/db"
)

var ErrNotFound = errors.New("not found")

// Order statuses and the allowed forward transitions.
var transitions = map[string][]string{
	"pending":           {"payment_received", "cancelled", "expired"},
	"payment_received":  {"payment_accepted", "rejected", "cancelled"},
	"payment_accepted":  {"shipped", "cancelled"},
	"shipped":           {},
	"expired":           {},
	"cancelled":         {},
	"rejected":          {"payment_received"},
}

func CanTransition(from, to string) bool {
	for _, t := range transitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

type ShippingAddress struct {
	MailingAddr  string `json:"mailing_addr"`
	MailingAddr2 string `json:"mailing_addr2"`
	City         string `json:"city"`
	Postcode     string `json:"postcode"`
	State        string `json:"state"`
}

type OrderItem struct {
	ID          uuid.UUID  `json:"id"`
	ProductID   *uuid.UUID `json:"product_id"`
	SKU         string     `json:"sku"`
	ProductName string     `json:"product_name"`
	Quantity    int        `json:"quantity"`
	UnitPrice   float64    `json:"unit_price"`
	LineTotal   float64    `json:"line_total"`
}

type Payment struct {
	ID        uuid.UUID `json:"id"`
	ProofKey  string    `json:"proof_key"`
	ProofURL  string    `json:"proof_url,omitempty"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Order struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"-"`
	CustomerID      *uuid.UUID      `json:"customer_id"`
	Status          string          `json:"status"`
	FullName        string          `json:"full_name"`
	Email           string          `json:"email"`
	ContactNo       string          `json:"contact_no"`
	ShippingAddress ShippingAddress `json:"shipping_address"`
	AdditionalNotes string          `json:"additional_notes"`
	TotalPrice      float64         `json:"total_price"`
	ExpiredDate     time.Time       `json:"expired_date"`
	CreatedAt       time.Time       `json:"created_at"`
	Items           []OrderItem     `json:"items,omitempty"`
	Payments        []Payment       `json:"payments,omitempty"`
}

type Repo struct{ pool *db.Pool }

func NewRepo(pool *db.Pool) *Repo { return &Repo{pool: pool} }

// Create inserts an order + its items atomically and sets the 3-day expiry.
func (r *Repo) Create(ctx context.Context, o *Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if o.ExpiredDate.IsZero() {
		o.ExpiredDate = time.Now().Add(72 * time.Hour) // FR-3.2: now + 3 days
	}
	var total float64
	for i := range o.Items {
		o.Items[i].LineTotal = float64(o.Items[i].Quantity) * o.Items[i].UnitPrice
		total += o.Items[i].LineTotal
	}
	if o.TotalPrice == 0 {
		o.TotalPrice = total
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (tenant_id, customer_id, status, full_name, email, contact_no,
			shipping_address, additional_notes, total_price, expired_date)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at`,
		o.TenantID, o.CustomerID, o.Status, o.FullName, o.Email, o.ContactNo,
		o.ShippingAddress, o.AdditionalNotes, o.TotalPrice, o.ExpiredDate,
	).Scan(&o.ID, &o.CreatedAt)
	if err != nil {
		return err
	}
	for i := range o.Items {
		it := &o.Items[i]
		if err := tx.QueryRow(ctx, `
			INSERT INTO order_items (order_id, product_id, sku, product_name, quantity, unit_price, line_total)
			VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
			o.ID, it.ProductID, it.SKU, it.ProductName, it.Quantity, it.UnitPrice, it.LineTotal,
		).Scan(&it.ID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repo) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE orders SET status=$3, updated_at=now() WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, status)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) UpdateDetails(ctx context.Context, o *Order) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE orders SET full_name=$3, email=$4, contact_no=$5, shipping_address=$6,
			additional_notes=$7, total_price=$8, updated_at=now()
		WHERE id=$1 AND tenant_id=$2`,
		o.ID, o.TenantID, o.FullName, o.Email, o.ContactNo, o.ShippingAddress,
		o.AdditionalNotes, o.TotalPrice)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM orders WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) Get(ctx context.Context, tenantID, id uuid.UUID) (*Order, error) {
	o := &Order{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, customer_id, status, full_name, email, contact_no,
			shipping_address, additional_notes, total_price, expired_date, created_at
		FROM orders WHERE id=$1 AND tenant_id=$2`, id, tenantID).
		Scan(&o.ID, &o.TenantID, &o.CustomerID, &o.Status, &o.FullName, &o.Email, &o.ContactNo,
			&o.ShippingAddress, &o.AdditionalNotes, &o.TotalPrice, &o.ExpiredDate, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.loadChildren(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

// GetByID fetches an order without tenant scoping — used by the public invoice
// route which is authorized by an unguessable order id (UUID) instead.
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	o := &Order{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, customer_id, status, full_name, email, contact_no,
			shipping_address, additional_notes, total_price, expired_date, created_at
		FROM orders WHERE id=$1`, id).
		Scan(&o.ID, &o.TenantID, &o.CustomerID, &o.Status, &o.FullName, &o.Email, &o.ContactNo,
			&o.ShippingAddress, &o.AdditionalNotes, &o.TotalPrice, &o.ExpiredDate, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.loadChildren(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

func (r *Repo) loadChildren(ctx context.Context, o *Order) error {
	rows, err := r.pool.Query(ctx, `
		SELECT id, product_id, sku, product_name, quantity, unit_price, line_total
		FROM order_items WHERE order_id=$1 ORDER BY id`, o.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.ID, &it.ProductID, &it.SKU, &it.ProductName, &it.Quantity, &it.UnitPrice, &it.LineTotal); err != nil {
			rows.Close()
			return err
		}
		o.Items = append(o.Items, it)
	}
	rows.Close()

	prows, err := r.pool.Query(ctx, `
		SELECT id, proof_key, amount, status, created_at FROM payments
		WHERE order_id=$1 ORDER BY created_at DESC`, o.ID)
	if err != nil {
		return err
	}
	defer prows.Close()
	for prows.Next() {
		var p Payment
		if err := prows.Scan(&p.ID, &p.ProofKey, &p.Amount, &p.Status, &p.CreatedAt); err != nil {
			return err
		}
		o.Payments = append(o.Payments, p)
	}
	return prows.Err()
}

// List returns tenant orders with optional status filter (statuses) + search.
func (r *Repo) List(ctx context.Context, tenantID uuid.UUID, statuses []string, search string, limit, offset int) ([]Order, int, error) {
	args := []any{tenantID}
	where := "tenant_id=$1"
	if len(statuses) > 0 {
		args = append(args, statuses)
		where += " AND status = ANY($2)"
	}
	if search != "" {
		args = append(args, "%"+search+"%")
		where += " AND (full_name ILIKE $" + itoa(len(args)) + " OR email ILIKE $" + itoa(len(args)) + " OR contact_no ILIKE $" + itoa(len(args)) + ")"
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	q := `SELECT id, tenant_id, customer_id, status, full_name, email, contact_no,
			shipping_address, additional_notes, total_price, expired_date, created_at
		FROM orders WHERE ` + where + ` ORDER BY created_at DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.TenantID, &o.CustomerID, &o.Status, &o.FullName, &o.Email, &o.ContactNo,
			&o.ShippingAddress, &o.AdditionalNotes, &o.TotalPrice, &o.ExpiredDate, &o.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

// ExpireStale marks pending orders past their expiry as expired (background job
// foundation for EH-2 auto-chase). Returns count expired.
func (r *Repo) ExpireStale(ctx context.Context) (int64, error) {
	ct, err := r.pool.Exec(ctx,
		`UPDATE orders SET status='expired', updated_at=now()
		 WHERE status='pending' AND expired_date < now()`)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// ---- payments ----

func (r *Repo) AddPayment(ctx context.Context, orderID uuid.UUID, proofKey string, amount float64) (*Payment, error) {
	p := &Payment{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO payments (order_id, proof_key, amount, status)
		VALUES ($1,$2,$3,'submitted') RETURNING id, proof_key, amount, status, created_at`,
		orderID, proofKey, amount).Scan(&p.ID, &p.ProofKey, &p.Amount, &p.Status, &p.CreatedAt)
	return p, err
}

func (r *Repo) SetPaymentStatus(ctx context.Context, tenantID, paymentID uuid.UUID, status string) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE payments p SET status=$3, updated_at=now()
		FROM orders o WHERE p.id=$1 AND p.order_id=o.id AND o.tenant_id=$2`,
		paymentID, tenantID, status)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
