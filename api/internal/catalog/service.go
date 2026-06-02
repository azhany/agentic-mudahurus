package catalog

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/mudahurus/api/internal/events"
	"github.com/mudahurus/api/internal/httpx"
	"github.com/mudahurus/api/internal/storage"
)

type Service struct {
	repo    *Repo
	store   storage.Storage
	emitter events.Emitter
}

func NewService(repo *Repo, store storage.Storage, emitter events.Emitter) *Service {
	return &Service{repo: repo, store: store, emitter: emitter}
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func toUUID(s string) (*uuid.UUID, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, httpx.BadRequest("invalid id: " + s)
	}
	return &id, nil
}

// ProductInput is the create/update payload.
type ProductInput struct {
	CategoryID  string  `json:"category_id"`
	SKU         string  `json:"sku"`
	ProductName string  `json:"product_name"`
	Description string  `json:"description"`
	UnitPrice   float64 `json:"unit_price"`
	URLSlug     string  `json:"url_slug"`
	Status      string  `json:"status"`
}

func (s *Service) validateProduct(in ProductInput) error {
	v := httpx.NewValidator().
		Require("sku", in.SKU).
		Require("product_name", in.ProductName).
		Check(in.UnitPrice >= 0, "unit_price", "must be >= 0")
	if in.Status != "" && in.Status != "active" && in.Status != "inactive" {
		v.Check(false, "status", "must be active or inactive")
	}
	return v.Err()
}

func (s *Service) CreateProduct(ctx context.Context, tenantID uuid.UUID, in ProductInput) (*Product, error) {
	if err := s.validateProduct(in); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetProductBySKU(ctx, tenantID, in.SKU); err == nil {
		return nil, httpx.Conflict("a product with this SKU already exists")
	}
	catID, err := toUUID(in.CategoryID)
	if err != nil {
		return nil, err
	}
	status := in.Status
	if status == "" {
		status = "active"
	}
	slug := in.URLSlug
	if slug == "" {
		slug = slugify(in.ProductName)
	}
	p := &Product{
		TenantID: tenantID, CategoryID: catID, SKU: in.SKU, ProductName: in.ProductName,
		Description: in.Description, UnitPrice: in.UnitPrice, URLSlug: slug, Status: status,
	}
	if err := s.repo.CreateProduct(ctx, p); err != nil {
		return nil, err
	}
	s.emitter.Emit(ctx, events.ProductCreated(tenantID, p.ID)) // MH-506 trigger
	return p, nil
}

func (s *Service) UpdateProduct(ctx context.Context, tenantID, id uuid.UUID, in ProductInput) (*Product, error) {
	if err := s.validateProduct(in); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetProduct(ctx, tenantID, id)
	if err != nil {
		return nil, mapErr(err)
	}
	// SKU uniqueness if changed
	if !strings.EqualFold(existing.SKU, in.SKU) {
		if _, err := s.repo.GetProductBySKU(ctx, tenantID, in.SKU); err == nil {
			return nil, httpx.Conflict("a product with this SKU already exists")
		}
	}
	catID, err := toUUID(in.CategoryID)
	if err != nil {
		return nil, err
	}
	status := in.Status
	if status == "" {
		status = existing.Status
	}
	slug := in.URLSlug
	if slug == "" {
		slug = existing.URLSlug
	}
	p := &Product{
		ID: id, TenantID: tenantID, CategoryID: catID, SKU: in.SKU, ProductName: in.ProductName,
		Description: in.Description, UnitPrice: in.UnitPrice, URLSlug: slug, Status: status,
	}
	if err := s.repo.UpdateProduct(ctx, p); err != nil {
		return nil, mapErr(err)
	}
	return s.repo.GetProduct(ctx, tenantID, id)
}

func (s *Service) DeleteProduct(ctx context.Context, tenantID, id uuid.UUID) error {
	return mapErr(s.repo.DeleteProduct(ctx, tenantID, id))
}

func (s *Service) GetProduct(ctx context.Context, tenantID, id uuid.UUID) (*Product, error) {
	p, err := s.repo.GetProduct(ctx, tenantID, id)
	if err != nil {
		return nil, mapErr(err)
	}
	s.attachImageURL(ctx, p)
	return p, nil
}

func (s *Service) GetProductBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*Product, error) {
	p, err := s.repo.GetProductBySKU(ctx, tenantID, sku)
	if err != nil {
		return nil, mapErr(err)
	}
	s.attachImageURL(ctx, p)
	return p, nil
}

func (s *Service) ListProducts(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]Product, int, error) {
	products, total, err := s.repo.ListProducts(ctx, tenantID, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	for i := range products {
		s.attachImageURL(ctx, &products[i])
	}
	return products, total, nil
}

func (s *Service) attachImageURL(ctx context.Context, p *Product) {
	if p.ImageKey != "" {
		if url, err := s.store.SignedGetURL(ctx, p.ImageKey, 3600); err == nil {
			p.ImageURL = url
		}
	}
}

// ---- categories ----

func (s *Service) CreateCategory(ctx context.Context, tenantID uuid.UUID, name, desc string) (*Category, error) {
	if err := httpx.NewValidator().Require("name", name).Err(); err != nil {
		return nil, err
	}
	return s.repo.CreateCategory(ctx, tenantID, name, desc)
}

func (s *Service) UpdateCategory(ctx context.Context, tenantID, id uuid.UUID, name, desc string) error {
	if err := httpx.NewValidator().Require("name", name).Err(); err != nil {
		return err
	}
	return mapErr(s.repo.UpdateCategory(ctx, tenantID, id, name, desc))
}

func (s *Service) DeleteCategory(ctx context.Context, tenantID, id uuid.UUID) error {
	return mapErr(s.repo.DeleteCategory(ctx, tenantID, id))
}

func (s *Service) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]Category, error) {
	return s.repo.ListCategories(ctx, tenantID)
}

func mapErr(err error) error {
	if err == ErrNotFound {
		return httpx.NotFound("resource not found")
	}
	return err
}
