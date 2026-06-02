package httpx

import (
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/mudahurus/api/internal/logging"
)

const RequestIDHeader = "X-Request-ID"

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mudahurus_http_requests_total",
		Help: "Total HTTP requests by method, route and status.",
	}, []string{"method", "route", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mudahurus_http_request_duration_seconds",
		Help:    "HTTP request latency.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
	}, []string{"method", "route"})
)

// RequestContext wires request-id + structured logging into the context.
func RequestContext(logLevel string) echo.MiddlewareFunc {
	base := logging.New(logLevel)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rid := c.Request().Header.Get(RequestIDHeader)
			if rid == "" {
				rid = uuid.NewString()
			}
			c.Response().Header().Set(RequestIDHeader, rid)
			l := base.With("request_id", rid)
			ctx := logging.Into(c.Request().Context(), l)
			c.SetRequest(c.Request().WithContext(ctx))
			c.Set("request_id", rid)
			return next(c)
		}
	}
}

// AccessLog logs each request and records metrics.
func AccessLog() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			if err != nil {
				c.Error(err)
			}
			route := c.Path()
			if route == "" {
				route = "unmatched"
			}
			status := c.Response().Status
			dur := time.Since(start)
			httpRequests.WithLabelValues(c.Request().Method, route, statusClass(status)).Inc()
			httpDuration.WithLabelValues(c.Request().Method, route).Observe(dur.Seconds())

			l := logging.From(c.Request().Context())
			attrs := []any{
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", status,
				"duration_ms", dur.Milliseconds(),
				"remote_ip", c.RealIP(),
			}
			if tid, ok := c.Get("tenant_id").(string); ok && tid != "" {
				attrs = append(attrs, "tenant_id", tid)
			}
			l.Info("request", attrs...)
			return nil
		}
	}
}

func statusClass(status int) string {
	switch {
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 300:
		return "3xx"
	default:
		return "2xx"
	}
}
