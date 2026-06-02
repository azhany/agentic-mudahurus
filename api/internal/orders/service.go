package orders

import (
	"context"

	"github.com/google/uuid"

	"github.com/mudahurus/api/internal/catalog"
	"github.com/mudahurus/api/internal/events"
	"github.com/mudahurus/api/internal/httpx"
	"github.com/mudahurus/api/internal/storage"
)

type Service struct {
	repo    *Repo
	catalog *catalog.Service
	store   storage.Storage
	emitter events.Emitter
}

func NewService(repo *Repo, cat *catalog.Service, store storage.Storage, emitter events.Emitter) *Service {
	return &Service{repo: repo, catalog: cat, store: store, emitter: emitter}
}

// AdminInput is the admin create/update order payload.
type AdminInput struct {
	FullName        string          `json:"full_name"`
	Email           string          `json:"email"`
	ContactNo       string          `json:"contact_no"`
	ShippingAddress ShippingAddress `json:"shipping_address"`
	AdditionalNotes string          `json:"additional_notes"`
	Items           []ItemInput     `json:"items"`
}

type ItemInput struct {
	SKU       string  `json:"sku"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

func (s *Service) buildItems(ctx context.Context, tenantID uuid.UUID, inputs []ItemInput) ([]OrderItem, error) {
	var items []OrderItem
	for _, in := range inputs {
		if in.Quantity <= 0 {
			in.Quantity = 1
		}
		it := OrderItem{SKU: in.SKU, Quantity: in.Quantity, UnitPrice: in.UnitPrice}
		// Enrich from catalog (name, price) when the SKU resolves.
		if p, err := s.catalog.GetProductBySKU(ctx, tenantID, in.SKU); err == nil {
			it.ProductID = &p.ID
			it.ProductName = p.ProductName
			if it.UnitPrice == 0 {
				it.UnitPrice = p.UnitPrice
			}
		}
		items = append(items, it)
	}
	return items, nil
}

func (s *Service) CreateAdmin(ctx context.Context, tenantID uuid.UUID, in AdminInput) (*Order, error) {
	if err := httpx.NewValidator().Require("full_name", in.FullName).Err(); err != nil {
		return nil, err
	}
	if len(in.Items) == 0 {
		return nil, httpx.BadRequest("at least one item is required")
	}
	items, err := s.buildItems(ctx, tenantID, in.Items)
	if err != nil {
		return nil, err
	}
	o := &Order{
		TenantID: tenantID, Status: "pending", FullName: in.FullName, Email: in.Email,
		ContactNo: in.ContactNo, ShippingAddress: in.ShippingAddress,
		AdditionalNotes: in.AdditionalNotes, Items: items,
	}
	if err := s.repo.Create(ctx, o); err != nil {
		return nil, err
	}
	s.emitter.Emit(ctx, events.OrderCreated(tenantID, o.ID))
	return o, nil
}

// GuestCheckout creates a pending order from the public storefront (FR-3.1).
func (s *Service) GuestCheckout(ctx context.Context, tenantID uuid.UUID, in AdminInput) (*Order, error) {
	if err := httpx.NewValidator().
		Require("full_name", in.FullName).
		Require("contact_no", in.ContactNo).
		Err(); err != nil {
		return nil, err
	}
	if len(in.Items) == 0 {
		return nil, httpx.BadRequest("cart is empty")
	}
	return s.CreateAdmin(ctx, tenantID, in)
}

func (s *Service) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, to string) (*Order, error) {
	o, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if o.Status == to {
		return o, nil
	}
	if !CanTransition(o.Status, to) {
		return nil, httpx.BadRequest("invalid status transition: " + o.Status + " -> " + to)
	}
	if err := s.repo.UpdateStatus(ctx, tenantID, id, to); err != nil {
		return nil, mapErr(err)
	}
	return s.repo.Get(ctx, tenantID, id)
}

func (s *Service) UpdateDetails(ctx context.Context, tenantID, id uuid.UUID, in AdminInput) (*Order, error) {
	o, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, mapErr(err)
	}
	o.FullName, o.Email, o.ContactNo = in.FullName, in.Email, in.ContactNo
	o.ShippingAddress, o.AdditionalNotes = in.ShippingAddress, in.AdditionalNotes
	if err := s.repo.UpdateDetails(ctx, o); err != nil {
		return nil, mapErr(err)
	}
	return s.repo.Get(ctx, tenantID, id)
}

func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return mapErr(s.repo.Delete(ctx, tenantID, id))
}

func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*Order, error) {
	o, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, mapErr(err)
	}
	s.attachProofURLs(ctx, o)
	return o, nil
}

func (s *Service) List(ctx context.Context, tenantID uuid.UUID, statuses []string, search string, limit, offset int) ([]Order, int, error) {
	return s.repo.List(ctx, tenantID, statuses, search, limit, offset)
}

// AddPaymentProof attaches an uploaded payment proof to an order and moves it
// to payment_received (FR-3.5). proofKey is from object storage.
func (s *Service) AddPaymentProof(ctx context.Context, orderID uuid.UUID, proofKey string, amount float64) (*Payment, error) {
	o, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, mapErr(err)
	}
	p, err := s.repo.AddPayment(ctx, orderID, proofKey, amount)
	if err != nil {
		return nil, err
	}
	if o.Status == "pending" {
		_ = s.repo.UpdateStatus(ctx, o.TenantID, orderID, "payment_received")
	}
	s.emitter.Emit(ctx, events.PaymentUploaded(o.TenantID, orderID)) // MH-506 trigger (OCR ingestion)
	return p, nil
}

// VerifyPayment marks a payment verified/rejected and syncs the order status.
func (s *Service) VerifyPayment(ctx context.Context, tenantID, paymentID uuid.UUID, accept bool) error {
	status := "verified"
	if !accept {
		status = "rejected"
	}
	return mapErr(s.repo.SetPaymentStatus(ctx, tenantID, paymentID, status))
}

func (s *Service) attachProofURLs(ctx context.Context, o *Order) {
	for i := range o.Payments {
		if o.Payments[i].ProofKey != "" {
			if url, err := s.store.SignedGetURL(ctx, o.Payments[i].ProofKey, 3600); err == nil {
				o.Payments[i].ProofURL = url
			}
		}
	}
}

func mapErr(err error) error {
	if err == ErrNotFound {
		return httpx.NotFound("order not found")
	}
	return err
}
