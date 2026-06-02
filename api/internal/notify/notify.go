// Package notify provides a minimal mailer abstraction. In dev the "log" sink
// prints messages (and tokens) to stdout; SMTP is pluggable. The EH-3
// notifications enhancement extends this interface (WhatsApp/email).
package notify

import (
	"context"
	"log/slog"
)

type Message struct {
	To      string
	Subject string
	Body    string
}

type Mailer interface {
	Send(ctx context.Context, m Message) error
}

// LogMailer logs messages instead of sending — used in dev/test.
type LogMailer struct{ Log *slog.Logger }

func (l *LogMailer) Send(ctx context.Context, m Message) error {
	l.Log.Info("email (log sink)", "to", m.To, "subject", m.Subject, "body", m.Body)
	return nil
}

// New returns a mailer for the configured sink. Only "log" is wired in v1;
// real SMTP is a deferred enhancement (EH-3).
func New(sink string, from string, log *slog.Logger) Mailer {
	return &LogMailer{Log: log}
}
