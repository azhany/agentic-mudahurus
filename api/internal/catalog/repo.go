// Package catalog implements products & categories (FR-2.x). All queries are
// tenant-scoped and parameterized.
package catalog

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mudahurus/api/internal/db"
)

var ErrNotFound = errors.New("not found")

type Category struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"-"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Product struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"-"`
	CategoryID   *uuid.UUID `json:"category_id"`
	CategoryName string     `json:"category,omitempty"`
	SKU          string     `json:"sku"`
	ProductName  string     `json:"product_name"`
	Description  string     `json:"description"`
	UnitPrice    float64    `json:"unit_price"`
	URLSlug      string     `json:"url_slug"`
	ImageKey     string     `json:"image_key"`
	ImageURL     string     `json:"image_url,omitempty"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Repo struct{ pool *db.Pool }

func NewRepo(pool *db.Pool) *Repo { return &Repo{pool: pool} }

// ---- Categories ----

func (r *Repo) CreateCategory(ctx context.Context, tenantID uuid.UUID, name, desc string) (*Category, error) {
	c := &Category{TenantID: tenantID, Name: name, Description: desc}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO categories (tenant_id, name, description) VALUES ($1,$2,$3)
		 RETURNING id, created_at`, tenantID, name, desc).Scan(&c.ID, &c.CreatedAt)
	return c, err
}

func (r *Repo) UpdateCategory(ctx context.Context, tenantID, id uuid.UUID, name, desc string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE categories SET name=$3, description=$4, updated_at=now()
		 WHERE id=$1 AND tenant_id=$2`, id, tenantID, name, desc)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) DeleteCategory(ctx context.Context, tenantID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM categories WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]Category, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, created_at FROM categories
		 WHERE tenant_id=$1 ORDER BY name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repo) GetCategory(ctx context.Context, tenantID, id uuid.UUID) (*Category, error) {
	c := &Category{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, created_at FROM categories WHERE id=$1 AND tenant_id=$2`,
		id, tenantID).Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

// ---- Products ----

const productCols = `p.id, p.category_id, c.name, p.sku, p.product_name, p.description,
	p.unit_price, p.url_slug, p.image_key, p.status, p.created_at`

func scanProduct(row pgx.Row) (*Product, error) {
	p := &Product{}
	var catName *string
	err := row.Scan(&p.ID, &p.CategoryID, &catName, &p.SKU, &p.ProductName, &p.Description,
		&p.UnitPrice, &p.URLSlug, &p.ImageKey, &p.Status, &p.CreatedAt)
	if catName != nil {
		p.CategoryName = *catName
	}
	return p, err
}

func (r *Repo) CreateProduct(ctx context.Context, p *Product) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO products (tenant_id, category_id, sku, product_name, description, unit_price, url_slug, image_key, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at`,
		p.TenantID, p.CategoryID, p.SKU, p.ProductName, p.Description, p.UnitPrice, p.URLSlug, p.ImageKey, p.Status,
	).Scan(&p.ID, &p.CreatedAt)
}

func (r *Repo) UpdateProduct(ctx context.Context, p *Product) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE products SET category_id=$3, sku=$4, product_name=$5, description=$6,
			unit_price=$7, url_slug=$8, status=$9, updated_at=now()
		WHERE id=$1 AND tenant_id=$2`,
		p.ID, p.TenantID, p.CategoryID, p.SKU, p.ProductName, p.Description, p.UnitPrice, p.URLSlug, p.Status)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) SetProductImage(ctx context.Context, tenantID, id uuid.UUID, imageKey string) (oldKey string, err error) {
	err = r.pool.QueryRow(ctx, `
		WITH old AS (SELECT image_key FROM products WHERE id=$1 AND tenant_id=$2)
		UPDATE products SET image_key=$3, updated_at=now() WHERE id=$1 AND tenant_id=$2
		RETURNING (SELECT image_key FROM old)`, id, tenantID, imageKey).Scan(&oldKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return oldKey, err
}

func (r *Repo) DeleteProduct(ctx context.Context, tenantID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM products WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) GetProduct(ctx context.Context, tenantID, id uuid.UUID) (*Product, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+productCols+` FROM products p LEFT JOIN categories c ON c.id=p.category_id
		 WHERE p.id=$1 AND p.tenant_id=$2`, id, tenantID)
	p, err := scanProduct(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r *Repo) GetProductBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*Product, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+productCols+` FROM products p LEFT JOIN categories c ON c.id=p.category_id
		 WHERE p.tenant_id=$1 AND lower(p.sku)=lower($2)`, tenantID, sku)
	p, err := scanProduct(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

// ListProducts is tenant-scoped with optional search + pagination.
func (r *Repo) ListProducts(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]Product, int, error) {
	args := []any{tenantID}
	where := "p.tenant_id=$1"
	if search != "" {
		args = append(args, "%"+search+"%")
		where += " AND (p.product_name ILIKE $2 OR p.sku ILIKE $2 OR p.description ILIKE $2)"
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM products p WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	q := `SELECT ` + productCols + ` FROM products p LEFT JOIN categories c ON c.id=p.category_id
		WHERE ` + where + ` ORDER BY p.created_at DESC LIMIT $` +
		strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *p)
	}
	return out, total, rows.Err()
}
