// Package accesslog provides async, parameterized visit logging (FR-7.2, MH-405).
// Replaces the legacy general_log() which used string-interpolated SQL.
package accesslog

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/mudahurus/api/internal/db"
	"github.com/mudahurus/api/internal/tenancy"
)

type entry struct {
	tenantID  *uuid.UUID
	ip        string
	referrer  string
	url       string
	uri       string
	userAgent string
}

// Logger buffers entries and flushes them to Postgres on a background worker,
// so logging never blocks the request path.
type Logger struct {
	pool *db.Pool
	ch   chan entry
	log  *slog.Logger
}

func New(pool *db.Pool, log *slog.Logger) *Logger {
	l := &Logger{pool: pool, ch: make(chan entry, 1024), log: log}
	go l.worker()
	return l
}

func (l *Logger) worker() {
	for e := range l.ch {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := l.pool.Exec(ctx, `
			INSERT INTO access_logs (tenant_id, ip, referrer, url, uri, user_agent)
			VALUES ($1, NULLIF($2,'')::inet, $3, $4, $5, $6)`,
			e.tenantID, e.ip, e.referrer, e.url, e.uri, e.userAgent)
		cancel()
		if err != nil {
			l.log.Warn("access log insert failed", "error", err)
		}
	}
}

// Middleware records each request asynchronously (drops on buffer overflow).
func (l *Logger) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			req := c.Request()
			e := entry{
				ip:        c.RealIP(),
				referrer:  req.Referer(),
				url:       req.URL.String(),
				uri:       req.URL.Path,
				userAgent: req.UserAgent(),
			}
			if id, terr := tenancy.From(req.Context()); terr == nil {
				tid := id.TenantID
				e.tenantID = &tid
			}
			select {
			case l.ch <- e:
			default: // buffer full — drop rather than block
			}
			return err
		}
	}
}
