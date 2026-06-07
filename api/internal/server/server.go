// Package server wires all modules together and mounts routes.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/mudahurus/api/internal/accesslog"
	"github.com/mudahurus/api/internal/assistant"
	"github.com/mudahurus/api/internal/auth"
	"github.com/mudahurus/api/internal/catalog"
	"github.com/mudahurus/api/internal/config"
	"github.com/mudahurus/api/internal/coupons"
	"github.com/mudahurus/api/internal/customers"
	"github.com/mudahurus/api/internal/dashboard"
	"github.com/mudahurus/api/internal/db"
	"github.com/mudahurus/api/internal/enhancements"
	"github.com/mudahurus/api/internal/events"
	"github.com/mudahurus/api/internal/httpx"
	"github.com/mudahurus/api/internal/notify"
	"github.com/mudahurus/api/internal/orders"
	"github.com/mudahurus/api/internal/storage"
	"github.com/mudahurus/api/internal/storefront"
)

type App struct {
	Cfg    *config.Config
	Log    *slog.Logger
	Pool   *db.Pool
	Tokens *auth.TokenManager
	Mailer notify.Mailer
	AuthMW *auth.Middleware
	Echo   *echo.Echo
}

// buildStorage selects the object-storage backend. Local disk is the default
// (no external deps); S3/MinIO is used when S3_ENDPOINT is reachable.
func (a *App) buildStorage() (storage.Storage, bool) {
	if a.Cfg.Env != "development" || a.Cfg.S3Endpoint != "localhost:9000" {
		if s3, err := storage.NewS3(a.Cfg.S3Endpoint, a.Cfg.S3AccessKey, a.Cfg.S3SecretKey, a.Cfg.S3Bucket, a.Cfg.S3UseSSL); err == nil {
			a.Log.Info("object storage: s3/minio", "endpoint", a.Cfg.S3Endpoint)
			return s3, false
		}
		a.Log.Warn("s3 unavailable, falling back to local disk storage")
	}
	root := filepath.Join(".data", "objects")
	// Relative base so signed URLs resolve against the browser's origin and flow
	// through the SPA dev-proxy / nginx (/api/files/*) in every environment.
	local, err := storage.NewLocal(root, "/api/files", a.Cfg.JWTAccessSecret)
	if err != nil {
		a.Log.Error("local storage init failed", "error", err)
	}
	a.Log.Info("object storage: local disk", "root", root)
	return local, true
}

// Mount constructs and registers all handlers.
func (a *App) Mount() {
	e := a.Echo

	store, isLocal := a.buildStorage()
	emitter := events.New(a.Cfg.EventsSink, a.Cfg.AirflowTriggerURL, a.Log)

	// --- shared stores/services ---
	authStore := auth.NewStore(a.Pool)
	authSvc := auth.NewService(authStore, a.Tokens, a.Mailer)
	authHandler := auth.NewHandler(authSvc)

	catalogRepo := catalog.NewRepo(a.Pool)
	catalogSvc := catalog.NewService(catalogRepo, store, emitter)
	catalogHandler := catalog.NewHandler(catalogSvc, store)

	ordersRepo := orders.NewRepo(a.Pool)
	ordersSvc := orders.NewService(ordersRepo, catalogSvc, store, emitter)
	sellerLookup := func(c echo.Context, tenantID uuid.UUID) orders.SellerInfo {
		if t, err := authStore.TenantByID(c.Request().Context(), tenantID); err == nil {
			return orders.SellerInfo{FullName: t.FullName, StoreName: t.StoreName}
		}
		return orders.SellerInfo{}
	}
	ordersHandler := orders.NewHandler(ordersSvc, store, sellerLookup)

	customersHandler := customers.NewHandler(customers.NewRepo(a.Pool))
	couponsHandler := coupons.NewHandler(coupons.NewRepo(a.Pool))
	dashboardHandler := dashboard.NewHandler(a.Pool)
	storefrontHandler := storefront.NewHandler(a.Pool, authStore, ordersSvc, store)
	assistantHandler := assistant.NewHandler(a.Cfg.RAGBaseURL, authStore)

	accessLogger := accesslog.New(a.Pool, a.Log)

	// All JSON API routes live under /api so the SPA can own the human-facing
	// paths (/store/{username}, /invoice/{id}, /admin/*, /login) without colliding
	// with the API (ARCHITECTURE §4: clean REST API behind the Vue SPA).
	apiRoot := e.Group("/api")

	// --- public (unauthenticated) routes ---
	public := apiRoot.Group("")
	public.Use(accessLogger.Middleware())
	authHandlerPublic := public

	// Auth: register handler needs both public + authed groups; build authed first.
	authed := apiRoot.Group("")
	authed.Use(a.AuthMW.Authenticated())

	authHandler.Routes(authHandlerPublic, authed)

	// Authenticated, tenant-scoped admin API.
	catalogHandler.Routes(authed)
	ordersHandler.AdminRoutes(authed)
	customersHandler.Routes(authed)
	couponsHandler.Routes(authed)
	dashboardHandler.Routes(authed)
	assistantHandler.AdminRoutes(authed)

	// Operator-only example route group (RBAC demo, MH-104).
	operator := apiRoot.Group("/operator")
	operator.Use(a.AuthMW.Authenticated(), a.AuthMW.RequireRole("operator"))
	operator.GET("/tenants/count", func(c echo.Context) error {
		var n int
		_ = a.Pool.QueryRow(c.Request().Context(), `SELECT count(*) FROM tenants`).Scan(&n)
		return c.JSON(http.StatusOK, map[string]int{"tenants": n})
	})

	// --- public storefront (rate limited) ---
	storeGroup := apiRoot.Group("")
	storeGroup.Use(accessLogger.Middleware())
	storeGroup.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))
	storefrontHandler.Routes(storeGroup)
	assistantHandler.PublicRoutes(storeGroup)

	// Public invoice + payment proof upload (FR-3.4, FR-3.5).
	storeGroup.GET("/invoice/:id", ordersHandler.PublicInvoice)
	storeGroup.POST("/orders/:id/payment", ordersHandler.PublicUploadPayment)

	// --- Enhancement Backlog (EH-1..EH-6) ---
	// Seller-invoked endpoints work immediately; autonomous behaviour (EH-2
	// auto-chase) is gated behind MH_EH2_FULFILLMENT (default off).
	publicBase := os.Getenv("API_PUBLIC_BASE_URL")
	if publicBase == "" {
		publicBase = "http://localhost" + a.Cfg.HTTPAddr
	}
	ehModule := enhancements.NewModule(a.Pool, catalogSvc, ordersSvc, ordersRepo, a.Mailer, a.Log, publicBase)
	ehPublic := apiRoot.Group("")
	ehPublic.Use(accessLogger.Middleware())
	ehModule.Routes(authed, ehPublic)
	ehModule.StartBackground(context.Background())

	// --- local object-storage serve route (signed), under /api/files/* ---
	if isLocal {
		if ls, ok := store.(*storage.LocalStorage); ok {
			e.GET("/api/files/*", localFileHandler(ls))
		}
	}
}

func localFileHandler(ls *storage.LocalStorage) echo.HandlerFunc {
	return func(c echo.Context) error {
		key := c.Param("*")
		sig := c.QueryParam("sig")
		exp, _ := strconv.ParseInt(c.QueryParam("exp"), 10, 64)
		if !ls.Verify(key, sig, exp) {
			return httpx.Forbidden("invalid or expired signature")
		}
		rc, ct, err := ls.Open(c.Request().Context(), key)
		if err != nil {
			return httpx.NotFound("file not found")
		}
		defer rc.Close()
		return c.Stream(http.StatusOK, ct, rc)
	}
}
