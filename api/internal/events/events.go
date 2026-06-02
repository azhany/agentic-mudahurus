// Package events emits domain events that trigger RAG ingestion (MH-506).
// On product/order create and payment-proof upload, the API fires an event so
// the index refreshes without waiting for the scheduled DAG run.
//
// Sinks: "log" (dev no-op) and "http" (POST to the Airflow API trigger / queue).
// Emission is async and best-effort — it never blocks or fails the request.
package events

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Type       string    `json:"type"`        // product.created, order.created, payment.uploaded, ...
	TenantID   uuid.UUID `json:"tenant_id"`
	SourceType string    `json:"source_type"` // product | order | payment
	SourceID   string    `json:"source_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type Emitter interface {
	Emit(ctx context.Context, e Event)
}

// LogEmitter records events to the structured log (dev default).
type LogEmitter struct{ Log *slog.Logger }

func (l *LogEmitter) Emit(ctx context.Context, e Event) {
	l.Log.Info("ingestion event", "type", e.Type, "tenant_id", e.TenantID, "source_type", e.SourceType, "source_id", e.SourceID)
}

// HTTPEmitter POSTs the event to a trigger URL (Airflow REST / queue gateway).
type HTTPEmitter struct {
	URL    string
	Client *http.Client
	Log    *slog.Logger
}

func (h *HTTPEmitter) Emit(ctx context.Context, e Event) {
	go func() {
		body, _ := json.Marshal(map[string]any{"conf": e})
		req, err := http.NewRequest(http.MethodPost, h.URL, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		cl := h.Client
		if cl == nil {
			cl = &http.Client{Timeout: 5 * time.Second}
		}
		resp, err := cl.Do(req)
		if err != nil {
			h.Log.Warn("event emit failed", "type", e.Type, "error", err)
			return
		}
		_ = resp.Body.Close()
	}()
}

// New builds an emitter for the configured sink.
func New(sink, url string, log *slog.Logger) Emitter {
	if sink == "http" && url != "" {
		return &HTTPEmitter{URL: url, Log: log}
	}
	return &LogEmitter{Log: log}
}

// Helper constructors for the three v1 trigger points.
func ProductCreated(tenantID, id uuid.UUID) Event {
	return Event{Type: "product.created", TenantID: tenantID, SourceType: "product", SourceID: id.String(), OccurredAt: time.Now()}
}
func OrderCreated(tenantID, id uuid.UUID) Event {
	return Event{Type: "order.created", TenantID: tenantID, SourceType: "order", SourceID: id.String(), OccurredAt: time.Now()}
}
func PaymentUploaded(tenantID, orderID uuid.UUID) Event {
	return Event{Type: "payment.uploaded", TenantID: tenantID, SourceType: "payment", SourceID: orderID.String(), OccurredAt: time.Now()}
}
