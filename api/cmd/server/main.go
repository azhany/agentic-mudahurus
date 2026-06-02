// Command server is the MUDAHURUS 2.0 transactional API (Go/Echo).
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mudahurus/api/internal/auth"
	"github.com/mudahurus/api/internal/config"
	"github.com/mudahurus/api/internal/db"
	"github.com/mudahurus/api/internal/httpx"
	"github.com/mudahurus/api/internal/logging"
	"github.com/mudahurus/api/internal/notify"
	"github.com/mudahurus/api/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := logging.New(cfg.LogLevel)

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Error("migration failed", "error", err)
		os.Exit(1)
	}
	log.Info("migrations applied")

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = httpx.ErrorHandler
	e.Use(middleware.Recover())
	e.Use(httpx.RequestContext(cfg.LogLevel))
	e.Use(httpx.AccessLog())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{"Authorization", "Content-Type", httpx.RequestIDHeader},
	}))

	// Health & readiness probes (MH-004).
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.GET("/readyz", func(c echo.Context) error {
		cctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(cctx); err != nil {
			return httpx.Internal("database not ready")
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
	})
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Dependencies
	tokens := auth.NewTokenManager(cfg.JWTAccessSecret, cfg.JWTRefreshSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	mailer := notify.New(cfg.SMTPSink, cfg.SMTPFrom, log)
	authMW := auth.NewMiddleware(tokens)

	app := &server.App{
		Cfg:    cfg,
		Log:    log,
		Pool:   pool,
		Tokens: tokens,
		Mailer: mailer,
		AuthMW: authMW,
		Echo:   e,
	}
	app.Mount()

	// Graceful shutdown
	go func() {
		if err := e.Start(cfg.HTTPAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()
	log.Info("server started", "addr", cfg.HTTPAddr, "env", cfg.Env)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = e.Shutdown(shutdownCtx)
	log.Info("server stopped")
}
