package enhancements

import "context"

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
