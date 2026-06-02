// Package assistant is the Go proxy to the Python RAG plane (MH-603).
// It resolves tenant_id SERVER-SIDE (from the JWT for admin, or from
// /store/{username} for the storefront) and forwards the question to FastAPI.
// The client can NEVER supply or override tenant_id (ARCHITECTURE §8 guardrail).
package assistant

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/mudahurus/api/internal/auth"
	"github.com/mudahurus/api/internal/httpx"
	"github.com/mudahurus/api/internal/tenancy"
)

type Handler struct {
	ragBaseURL string
	auth       *auth.Store
	client     *http.Client
}

func NewHandler(ragBaseURL string, authStore *auth.Store) *Handler {
	return &Handler{ragBaseURL: ragBaseURL, auth: authStore, client: &http.Client{Timeout: 30 * time.Second}}
}

func (h *Handler) AdminRoutes(g *echo.Group) {
	g.POST("/assistant/search", h.adminSearch)
}

func (h *Handler) PublicRoutes(g *echo.Group) {
	g.POST("/store/:username/ask", h.storefrontAsk)
}

type assistantRequest struct {
	TenantID string `json:"tenant_id"`
	Question string `json:"question"`
	Scope    string `json:"scope"` // "admin" | "storefront"
	TopK     int    `json:"top_k,omitempty"`
}

func (h *Handler) adminSearch(c echo.Context) error {
	id, err := tenancy.From(c.Request().Context())
	if err != nil {
		return httpx.Unauthorized("authentication required")
	}
	var in struct {
		Question string `json:"question"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if in.Question == "" {
		return httpx.BadRequest("question is required")
	}
	return h.forward(c, assistantRequest{TenantID: id.TenantID.String(), Question: in.Question, Scope: "admin"})
}

func (h *Handler) storefrontAsk(c echo.Context) error {
	t, err := h.auth.TenantByUsername(c.Request().Context(), c.Param("username"))
	if err != nil {
		return httpx.NotFound("store not found")
	}
	var in struct {
		Question string `json:"question"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if in.Question == "" {
		return httpx.BadRequest("question is required")
	}
	return h.forward(c, assistantRequest{TenantID: t.ID.String(), Question: in.Question, Scope: "storefront"})
}

// forward POSTs to the FastAPI assistant endpoint and relays its JSON response.
func (h *Handler) forward(c echo.Context, req assistantRequest) error {
	body, _ := json.Marshal(req)
	hreq, err := http.NewRequestWithContext(c.Request().Context(), http.MethodPost,
		h.ragBaseURL+"/assistant/ask", bytes.NewReader(body))
	if err != nil {
		return httpx.Internal("assistant request failed")
	}
	hreq.Header.Set("Content-Type", "application/json")
	if rid, ok := c.Get("request_id").(string); ok {
		hreq.Header.Set(httpx.RequestIDHeader, rid)
	}
	resp, err := h.client.Do(hreq)
	if err != nil {
		// Graceful degradation: assistant unavailable -> refusal, not a 500.
		return c.JSON(http.StatusOK, map[string]any{
			"answer":    "The assistant is temporarily unavailable. Please try again shortly.",
			"grounded":  false,
			"citations": []any{},
		})
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	return c.Blob(resp.StatusCode, "application/json", payload)
}
