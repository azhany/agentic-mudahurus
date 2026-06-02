package orders

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/mudahurus/api/internal/httpx"
	"github.com/mudahurus/api/internal/storage"
	"github.com/mudahurus/api/internal/tenancy"
)

// SellerInfo describes the seller for invoice rendering, looked up by tenant id.
type SellerInfo struct {
	FullName  string
	StoreName string
}

// SellerLookup resolves seller display fields for the invoice (set by server wiring).
type SellerLookup func(ctx echo.Context, tenantID uuid.UUID) SellerInfo

type Handler struct {
	svc    *Service
	store  storage.Storage
	seller SellerLookup
}

func NewHandler(svc *Service, store storage.Storage, seller SellerLookup) *Handler {
	return &Handler{svc: svc, store: store, seller: seller}
}

// AdminRoutes are authenticated, tenant-scoped (FR-3.3).
func (h *Handler) AdminRoutes(g *echo.Group) {
	g.GET("/orders", h.list)
	g.GET("/pending_orders", h.listPending) // legacy api/pending_orders parity
	g.POST("/orders", h.create)
	g.GET("/orders/:id", h.get)
	g.PUT("/orders/:id", h.update)
	g.PATCH("/orders/:id/status", h.setStatus)
	g.DELETE("/orders/:id", h.delete)
	g.GET("/orders/:id/invoice.pdf", h.invoicePDF)
	g.PATCH("/payments/:pid/verify", h.verifyPayment)
}

func tid(c echo.Context) uuid.UUID {
	id, _ := tenancy.From(c.Request().Context())
	return id.TenantID
}

func parseUUID(c echo.Context, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		return uuid.Nil, httpx.BadRequest("invalid " + name)
	}
	return id, nil
}

func (h *Handler) list(c echo.Context) error {
	p := httpx.ParsePage(c)
	var statuses []string
	if st := c.QueryParam("status"); st != "" {
		statuses = []string{st}
	}
	items, total, err := h.svc.List(c.Request().Context(), tid(c), statuses, p.Search, p.Limit, p.Offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, httpx.ListResponse{Records: items, QueryRecordCount: total, TotalRecordCount: total})
}

func (h *Handler) listPending(c echo.Context) error {
	p := httpx.ParsePage(c)
	statuses := []string{"pending", "payment_received", "payment_accepted"}
	items, total, err := h.svc.List(c.Request().Context(), tid(c), statuses, p.Search, p.Limit, p.Offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, httpx.ListResponse{Records: items, QueryRecordCount: total, TotalRecordCount: total})
}

func (h *Handler) create(c echo.Context) error {
	var in AdminInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	o, err := h.svc.CreateAdmin(c.Request().Context(), tid(c), in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, o)
}

func (h *Handler) get(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	o, err := h.svc.Get(c.Request().Context(), tid(c), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, o)
}

func (h *Handler) update(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	var in AdminInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	o, err := h.svc.UpdateDetails(c.Request().Context(), tid(c), id, in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, o)
}

func (h *Handler) setStatus(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	o, err := h.svc.UpdateStatus(c.Request().Context(), tid(c), id, in.Status)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, o)
}

func (h *Handler) delete(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.Request().Context(), tid(c), id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) verifyPayment(c echo.Context) error {
	pid, err := parseUUID(c, "pid")
	if err != nil {
		return err
	}
	var in struct {
		Accept bool `json:"accept"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := h.svc.VerifyPayment(c.Request().Context(), tid(c), pid, in.Accept); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]bool{"accepted": in.Accept})
}

func (h *Handler) invoicePDF(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	o, err := h.svc.Get(c.Request().Context(), tid(c), id)
	if err != nil {
		return err
	}
	inv := &Invoice{Order: o}
	if h.seller != nil {
		si := h.seller(c, o.TenantID)
		inv.SellerName, inv.StoreName = si.FullName, si.StoreName
	}
	return c.Blob(http.StatusOK, "application/pdf", inv.RenderPDF())
}

// ---- public (storefront) handlers, mounted by the storefront module ----

// PublicInvoice returns invoice JSON for a given order id (FR-3.4). The order id
// (UUID) acts as the unguessable access token.
func (h *Handler) PublicInvoice(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	o, err := h.svc.repo.GetByID(c.Request().Context(), id)
	if err != nil {
		return mapErr(err)
	}
	h.svc.attachProofURLs(c.Request().Context(), o)
	inv := &Invoice{Order: o}
	if h.seller != nil {
		si := h.seller(c, o.TenantID)
		inv.SellerName, inv.StoreName = si.FullName, si.StoreName
	}
	if c.QueryParam("format") == "pdf" {
		return c.Blob(http.StatusOK, "application/pdf", inv.RenderPDF())
	}
	return c.JSON(http.StatusOK, inv)
}

// PublicUploadPayment accepts a payment-proof upload tied to an order (FR-3.5).
func (h *Handler) PublicUploadPayment(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	o, err := h.svc.repo.GetByID(c.Request().Context(), id)
	if err != nil {
		return mapErr(err)
	}
	file, err := c.FormFile("proof")
	if err != nil {
		return httpx.BadRequest("missing 'proof' file")
	}
	if file.Size > storage.MaxUploadBytes {
		return httpx.BadRequest("file exceeds maximum size")
	}
	ct := file.Header.Get("Content-Type")
	if !storage.AllowedProofMIME[ct] {
		return httpx.BadRequest("unsupported file type: " + ct)
	}
	src, err := file.Open()
	if err != nil {
		return httpx.BadRequest("cannot read upload")
	}
	defer src.Close()
	res, err := h.store.Put(c.Request().Context(), o.TenantID, "payments", file.Filename, ct, src, file.Size)
	if err != nil {
		return httpx.BadRequest(err.Error())
	}
	p, err := h.svc.AddPaymentProof(c.Request().Context(), id, res.Key, o.TotalPrice)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, p)
}
