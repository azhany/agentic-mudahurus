package enhancements

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mudahurus/api/internal/notify"
)

// EH-3 — Notifications: WhatsApp/email order updates & reminders.
// Extends the v1 notify.Mailer concept to multiple channels behind one
// interface. Builds on Orders + events. Disabled in v1.

type Channel string

const (
	ChannelEmail    Channel = "email"
	ChannelWhatsApp Channel = "whatsapp"
)

type Notification struct {
	Channel   Channel
	Recipient string // email or phone (E.164)
	Template  string // e.g. "order_confirmed", "payment_reminder", "shipped"
	Vars      map[string]string
}

// Notifier sends a templated notification on a channel.
type Notifier interface {
	Notify(ctx context.Context, n Notification) error
}

// Templates: BM-first message bodies (parity with the BM UI). A real impl would
// localize and render with the vars.
var Templates = map[string]string{
	"order_confirmed":  "Pesanan anda {order_id} telah diterima. Jumlah RM{total}.",
	"payment_reminder": "Peringatan: sila buat pembayaran untuk pesanan {order_id} sebelum {expiry}.",
	"shipped":          "Pesanan {order_id} telah dihantar. No. penjejakan: {tracking_no}.",
}

// MultiChannelNotifier dispatches to a per-channel sender. The scaffold has no
// real senders; production wires WhatsApp Business API + SMTP.
type MultiChannelNotifier struct {
	Senders map[Channel]Notifier
}

func (m *MultiChannelNotifier) Notify(ctx context.Context, n Notification) error {
	if s, ok := m.Senders[n.Channel]; ok {
		return s.Notify(ctx, n)
	}
	return nil // no-op when channel not configured (v1)
}

// Render fills a template body with the notification vars.
func Render(template string, vars map[string]string) string {
	body, ok := Templates[template]
	if !ok {
		body = template // allow raw bodies
	}
	for k, v := range vars {
		body = strings.ReplaceAll(body, "{"+k+"}", v)
	}
	return body
}

// EmailSender bridges EH-3 to the v1 notify.Mailer (log sink in dev, SMTP later).
type EmailSender struct{ Mailer notify.Mailer }

func (e *EmailSender) Notify(ctx context.Context, n Notification) error {
	return e.Mailer.Send(ctx, notify.Message{
		To:      n.Recipient,
		Subject: "MUDAHURUS: " + n.Template,
		Body:    Render(n.Template, n.Vars),
	})
}

// WhatsAppSender POSTs to a configured WhatsApp Business API webhook. When no
// URL is configured it logs (dev default) instead of failing.
type WhatsAppSender struct {
	WebhookURL string
	Client     *http.Client
	Log        *slog.Logger
}

func (w *WhatsAppSender) Notify(ctx context.Context, n Notification) error {
	body := Render(n.Template, n.Vars)
	if w.WebhookURL == "" {
		w.Log.Info("whatsapp (log sink)", "to", n.Recipient, "body", body)
		return nil
	}
	payload, _ := json.Marshal(map[string]string{"to": n.Recipient, "message": body})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	cl := w.Client
	if cl == nil {
		cl = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := cl.Do(req)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
