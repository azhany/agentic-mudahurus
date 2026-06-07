package enhancements

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/mudahurus/api/internal/catalog"
	"github.com/mudahurus/api/internal/db"
	"github.com/mudahurus/api/internal/httpx"
	"github.com/mudahurus/api/internal/notify"
	"github.com/mudahurus/api/internal/orders"
	"github.com/mudahurus/api/internal/tenancy"
)

// Module wires EH-1 … EH-6 together and mounts their routes.
//
// Seller-invoked endpoints (copilot, content, analytics, recommendations,
// charge) work out of the box — they act only on the caller's own tenant data.
// AUTONOMOUS behaviour (the EH-2 background auto-chase loop) stays gated behind
// MH_EH2_FULFILLMENT=true so nothing fires unattended unless explicitly enabled.
type Module struct {
	pool       *db.Pool
	catalog    *catalog.Service
	orders     *orders.Service
	ordersRepo *orders.Repo
	log        *slog.Logger
	publicBase string

	content  ContentGenerator
	notifier Notifier
	gateway  Gateway
	tracking TrackingProvider
	router   *Router
}

func NewModule(pool *db.Pool, cat *catalog.Service, ord *orders.Service, ordRepo *orders.Repo, mailer notify.Mailer, log *slog.Logger, publicBase string) *Module {
	notifier := &MultiChannelNotifier{Senders: map[Channel]Notifier{
		ChannelEmail:    &EmailSender{Mailer: mailer},
		ChannelWhatsApp: &WhatsAppSender{WebhookURL: os.Getenv("MH_WHATSAPP_WEBHOOK"), Log: log},
	}}
	return &Module{
		pool: pool, catalog: cat, orders: ord, ordersRepo: ordRepo, log: log, publicBase: publicBase,
		content:  TemplateContentGenerator{},
		notifier: notifier,
		gateway:  MockGateway{BaseURL: publicBase},
		tracking: NoopTrackingProvider{},
		router:   NewRouter(),
	}
}

// Routes mounts all enhancement endpoints. authed = JWT+tenant group; public =
// unauthenticated group (payment webhook + mock hosted page).
func (m *Module) Routes(authed, public *echo.Group) {
	// EH-1 Seller Copilot
	authed.POST("/copilot/interpret", m.copilotInterpret)
	authed.POST("/copilot/execute", m.copilotExecute)
	// EH-4 AI content (copilot tool)
	authed.POST("/copilot/generate-content", m.generateContent)
	// EH-5 analytics + recommendations
	authed.GET("/analytics/insights", m.analyticsInsights)
	authed.GET("/recommendations", m.recommendations)
	// EH-6 payments + multi-currency
	authed.POST("/orders/:id/charge", m.charge)
	authed.GET("/currencies", m.currencies)
	// EH-2 fulfillment
	authed.GET("/fulfillment/chase-candidates", m.chaseCandidates)
	authed.POST("/fulfillment/track", m.track)
	// EH-3 notifications (manual send)
	authed.POST("/notifications/send", m.sendNotification)

	// public: payment gateway callback + mock hosted page
	public.POST("/payments/webhook", m.paymentWebhook)
	public.GET("/pay/mock/:ref", m.mockHostedPage)

	m.log.Info("enhancements mounted (EH-1..EH-6)",
		"eh2_autochase", Enabled(FlagFulfillment))
}

func tid(c echo.Context) uuid.UUID {
	id, _ := tenancy.From(c.Request().Context())
	return id.TenantID
}
func role(c echo.Context) string {
	id, _ := tenancy.From(c.Request().Context())
	return id.Role
}

// ---------- EH-1: Seller Copilot ----------

func (m *Module) copilotInterpret(c echo.Context) error {
	var in struct{ Message string `json:"message"` }
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	kind := m.router.Route(AgentRequest{Role: role(c), Scope: "admin", Message: in.Message})
	if kind == AgentStorefrontAssistant {
		return c.JSON(http.StatusOK, map[string]any{
			"agent": kind, "actions": []ProposedAction{},
			"note": "This looks like a question — use POST /assistant/search for grounded answers.",
		})
	}
	actions := ParseSellerCommand(in.Message)
	return c.JSON(http.StatusOK, map[string]any{"agent": kind, "actions": actions})
}

func (m *Module) copilotExecute(c echo.Context) error {
	var act ProposedAction
	if err := httpx.Bind(c, &act); err != nil {
		return err
	}
	ctx := c.Request().Context()
	tenantID := tid(c)
	switch act.Kind {
	case "create_product":
		return m.execCreateProduct(c, ctx, tenantID, act)
	case "update_product_price":
		return m.execUpdatePrice(c, ctx, tenantID, act)
	case "advance_order_status":
		return m.execAdvanceOrder(c, ctx, tenantID, act)
	case "generate_content":
		return m.generateContentFromAction(c, ctx, act)
	default:
		return httpx.BadRequest("unknown action kind: " + act.Kind)
	}
}

func (m *Module) execCreateProduct(c echo.Context, ctx context.Context, tenantID uuid.UUID, act ProposedAction) error {
	price, _ := strconv.ParseFloat(act.Params["unit_price"], 64)
	in := catalog.ProductInput{
		SKU:         act.Params["sku"],
		ProductName: act.Params["product_name"],
		UnitPrice:   price,
		Status:      "active",
	}
	if in.SKU == "" {
		in.SKU = deriveSKU(in.ProductName)
	}
	if catName := strings.TrimSpace(act.Params["category"]); catName != "" {
		in.CategoryID = m.resolveOrCreateCategory(ctx, tenantID, catName)
	}
	p, err := m.catalog.CreateProduct(ctx, tenantID, in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, map[string]any{"executed": "create_product", "product": p})
}

func (m *Module) execUpdatePrice(c echo.Context, ctx context.Context, tenantID uuid.UUID, act ProposedAction) error {
	price, err := strconv.ParseFloat(act.Params["unit_price"], 64)
	if err != nil {
		return httpx.BadRequest("invalid unit_price")
	}
	existing, err := m.catalog.GetProductBySKU(ctx, tenantID, act.Params["sku"])
	if err != nil {
		return err
	}
	in := catalog.ProductInput{
		CategoryID:  uuidPtrToStr(existing.CategoryID),
		SKU:         existing.SKU,
		ProductName: existing.ProductName,
		Description: existing.Description,
		UnitPrice:   price,
		URLSlug:     existing.URLSlug,
		Status:      existing.Status,
	}
	p, err := m.catalog.UpdateProduct(ctx, tenantID, existing.ID, in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"executed": "update_product_price", "product": p})
}

func (m *Module) execAdvanceOrder(c echo.Context, ctx context.Context, tenantID uuid.UUID, act ProposedAction) error {
	oid, err := m.resolveOrderID(ctx, tenantID, act.Target)
	if err != nil {
		return err
	}
	status := act.Params["status"]
	if status == "" {
		return httpx.BadRequest("target status is required")
	}
	o, err := m.orders.UpdateStatus(ctx, tenantID, oid, status)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"executed": "advance_order_status", "order": o})
}

// ---------- EH-4: AI content ----------

func (m *Module) generateContent(c echo.Context) error {
	var in struct {
		ProductName string   `json:"product_name"`
		Keywords    []string `json:"keywords"`
		Tone        string   `json:"tone"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if in.ProductName == "" {
		return httpx.BadRequest("product_name is required")
	}
	res, err := m.content.Generate(c.Request().Context(), ContentRequest{ProductName: in.ProductName, Keywords: in.Keywords, Tone: in.Tone})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func (m *Module) generateContentFromAction(c echo.Context, ctx context.Context, act ProposedAction) error {
	res, err := m.content.Generate(ctx, ContentRequest{ProductName: act.Params["product_name"]})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"executed": "generate_content", "content": res})
}

// ---------- EH-5: analytics + recommendations ----------

func (m *Module) analyticsInsights(c echo.Context) error {
	ctx := c.Request().Context()
	rows, err := m.pool.Query(ctx, `
		SELECT to_char(date_trunc('month', created_at), 'YYYY-MM') AS period,
		       COALESCE(SUM(total_price), 0)::float8 AS total
		FROM orders
		WHERE tenant_id = $1 AND status IN ('payment_accepted','shipped')
		GROUP BY 1 ORDER BY 1`, tid(c))
	if err != nil {
		return err
	}
	defer rows.Close()
	var series []SalesPoint
	for rows.Next() {
		var sp SalesPoint
		if err := rows.Scan(&sp.Period, &sp.Total); err != nil {
			return err
		}
		series = append(series, sp)
	}
	return c.JSON(http.StatusOK, map[string]any{"series": series, "insights": AnalyzeTrend(series)})
}

func (m *Module) recommendations(c echo.Context) error {
	ctx := c.Request().Context()
	tenantID := tid(c)
	productID := c.QueryParam("product_id")
	var rows interface {
		Next() bool
		Scan(...any) error
		Close()
		Err() error
	}
	var err error
	if pid, perr := uuid.Parse(productID); perr == nil {
		rows, err = m.pool.Query(ctx, `
			SELECT p.id, p.sku, p.product_name, p.unit_price::float8
			FROM products p
			WHERE p.tenant_id = $1 AND p.status = 'active' AND p.id <> $2
			  AND (p.category_id IS NOT DISTINCT FROM (SELECT category_id FROM products WHERE id = $2))
			ORDER BY p.created_at DESC LIMIT 5`, tenantID, pid)
	} else {
		rows, err = m.pool.Query(ctx, `
			SELECT p.id, p.sku, p.product_name, p.unit_price::float8
			FROM products p WHERE p.tenant_id = $1 AND p.status = 'active'
			ORDER BY p.created_at DESC LIMIT 5`, tenantID)
	}
	if err != nil {
		return err
	}
	defer rows.Close()
	type rec struct {
		ID          uuid.UUID `json:"id"`
		SKU         string    `json:"sku"`
		ProductName string    `json:"product_name"`
		UnitPrice   float64   `json:"unit_price"`
	}
	var out []rec
	for rows.Next() {
		var r rec
		if err := rows.Scan(&r.ID, &r.SKU, &r.ProductName, &r.UnitPrice); err != nil {
			return err
		}
		out = append(out, r)
	}
	return c.JSON(http.StatusOK, map[string]any{"recommendations": out})
}

// ---------- EH-6: payments + multi-currency ----------

func (m *Module) currencies(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{"supported": SupportedCurrencies, "base": "MYR"})
}

func (m *Module) charge(c echo.Context) error {
	ctx := c.Request().Context()
	tenantID := tid(c)
	oid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httpx.BadRequest("invalid order id")
	}
	o, err := m.orders.Get(ctx, tenantID, oid)
	if err != nil {
		return err
	}
	currency := strings.ToUpper(c.QueryParam("currency"))
	if currency == "" {
		currency = "MYR"
	}
	amount := Convert(Money{Amount: int64(o.TotalPrice*100 + 0.5), Currency: "MYR"}, currency)
	res, err := m.gateway.CreateCharge(ctx, ChargeRequest{OrderID: oid.String(), Amount: amount, Customer: o.Email})
	if err != nil {
		return err
	}
	pay, err := m.ordersRepo.AddPayment(ctx, oid, "", o.TotalPrice)
	if err != nil {
		return err
	}
	redirect := fmt.Sprintf("%s/api/pay/mock/%s?order=%s&payment=%s", m.publicBase, res.GatewayRef, oid, pay.ID)
	return c.JSON(http.StatusOK, map[string]any{
		"gateway_ref":  res.GatewayRef,
		"status":       res.Status,
		"payment_id":   pay.ID,
		"amount_minor": amount.Amount,
		"currency":     amount.Currency,
		"redirect_url": redirect,
	})
}

func (m *Module) mockHostedPage(c echo.Context) error {
	ref := c.Param("ref")
	order := c.QueryParam("order")
	payment := c.QueryParam("payment")
	html := fmt.Sprintf(`<!doctype html><html><body style="font-family:sans-serif;max-width:480px;margin:60px auto;text-align:center">
<h2>MUDAHURUS — Mock Payment</h2>
<p>Gateway ref: <code>%s</code></p>
<button id="pay" style="padding:12px 24px;background:#0e7490;color:#fff;border:0;border-radius:8px;cursor:pointer">Pay now</button>
<p id="msg"></p>
<script>
document.getElementById('pay').onclick=async()=>{
  const r=await fetch('/api/payments/webhook',{method:'POST',headers:{'Content-Type':'application/json'},
    body:JSON.stringify({order_id:'%s',payment_id:'%s',gateway_ref:'%s',status:'paid'})});
  document.getElementById('msg').textContent = r.ok ? 'Payment confirmed ✓' : 'Failed';
};
</script></body></html>`, ref, order, payment, ref)
	return c.HTML(http.StatusOK, html)
}

func (m *Module) paymentWebhook(c echo.Context) error {
	// Verify a shared secret if configured (real gateways sign callbacks).
	if secret := os.Getenv("MH_GATEWAY_SECRET"); secret != "" {
		if c.Request().Header.Get("X-Gateway-Signature") != secret {
			return httpx.Unauthorized("invalid gateway signature")
		}
	}
	var in struct {
		OrderID    string `json:"order_id"`
		PaymentID  string `json:"payment_id"`
		GatewayRef string `json:"gateway_ref"`
		Status     string `json:"status"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	oid, err := uuid.Parse(in.OrderID)
	if err != nil {
		return httpx.BadRequest("invalid order_id")
	}
	pid, err := uuid.Parse(in.PaymentID)
	if err != nil {
		return httpx.BadRequest("invalid payment_id")
	}
	ctx := c.Request().Context()
	o, err := m.ordersRepo.GetByID(ctx, oid)
	if err != nil {
		return httpx.NotFound("order not found")
	}
	if in.Status == "paid" {
		_ = m.ordersRepo.SetPaymentStatus(ctx, o.TenantID, pid, "verified")
		_ = m.ordersRepo.UpdateStatus(ctx, o.TenantID, oid, "payment_accepted")
		// EH-3: confirmation notification (best-effort).
		_ = m.notifier.Notify(ctx, Notification{
			Channel: ChannelEmail, Recipient: o.Email, Template: "order_confirmed",
			Vars: map[string]string{"order_id": oid.String()[:8], "total": fmt.Sprintf("%.2f", o.TotalPrice)},
		})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// ---------- EH-2: fulfillment ----------

func (m *Module) chaseCandidates(c echo.Context) error {
	ctx := c.Request().Context()
	tenantID := tid(c)
	rows, err := m.pool.Query(ctx, `
		SELECT o.id, o.full_name, o.email, o.contact_no, o.expired_date, o.total_price::float8
		FROM orders o
		WHERE o.tenant_id = $1 AND o.status = 'pending'
		  AND o.expired_date > now() AND o.expired_date <= now() + interval '24 hours'
		  AND NOT EXISTS (SELECT 1 FROM payments p WHERE p.order_id = o.id)
		ORDER BY o.expired_date ASC`, tenantID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type cand struct {
		ID        uuid.UUID `json:"id"`
		FullName  string    `json:"full_name"`
		Email     string    `json:"email"`
		ContactNo string    `json:"contact_no"`
		ExpiredAt time.Time `json:"expired_date"`
		Total     float64   `json:"total_price"`
		Reason    string    `json:"reason"`
	}
	notifyFlag := c.QueryParam("notify") == "true"
	var out []cand
	for rows.Next() {
		var x cand
		if err := rows.Scan(&x.ID, &x.FullName, &x.Email, &x.ContactNo, &x.ExpiredAt, &x.Total); err != nil {
			return err
		}
		x.Reason = DecideChase(x.ID.String(), "pending", x.ExpiredAt, false, time.Now()).Reason
		out = append(out, x)
		if notifyFlag {
			_ = m.notifier.Notify(ctx, Notification{
				Channel: ChannelEmail, Recipient: x.Email, Template: "payment_reminder",
				Vars: map[string]string{"order_id": x.ID.String()[:8], "expiry": x.ExpiredAt.Format("2006-01-02 15:04")},
			})
		}
	}
	return c.JSON(http.StatusOK, map[string]any{"candidates": out, "notified": notifyFlag})
}

func (m *Module) track(c echo.Context) error {
	var in struct{ TrackingNo string `json:"tracking_no"` }
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	st, err := m.tracking.Track(c.Request().Context(), in.TrackingNo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, st)
}

// ---------- EH-3: manual notification ----------

func (m *Module) sendNotification(c echo.Context) error {
	var in struct {
		Channel   string            `json:"channel"`
		Recipient string            `json:"recipient"`
		Template  string            `json:"template"`
		Vars      map[string]string `json:"vars"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := m.notifier.Notify(c.Request().Context(), Notification{
		Channel: Channel(in.Channel), Recipient: in.Recipient, Template: in.Template, Vars: in.Vars,
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "sent"})
}

// StartBackground launches the EH-2 auto-chase loop when MH_EH2_FULFILLMENT=true.
// It scans ALL tenants for unpaid pending orders nearing expiry and sends
// reminders. Returns immediately; the loop stops when ctx is cancelled.
func (m *Module) StartBackground(ctx context.Context) {
	if !Enabled(FlagFulfillment) {
		return
	}
	interval := time.Hour
	if v := os.Getenv("MH_EH2_CHASE_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}
	m.log.Info("EH-2 auto-chase loop started", "interval", interval.String())
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				m.runChaseSweep(ctx)
			}
		}
	}()
}

func (m *Module) runChaseSweep(ctx context.Context) {
	rows, err := m.pool.Query(ctx, `
		SELECT o.id, o.tenant_id, o.email, o.expired_date
		FROM orders o
		WHERE o.status = 'pending' AND o.expired_date > now()
		  AND o.expired_date <= now() + interval '24 hours'
		  AND NOT EXISTS (SELECT 1 FROM payments p WHERE p.order_id = o.id)`)
	if err != nil {
		m.log.Warn("chase sweep query failed", "error", err)
		return
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, tenant uuid.UUID
		var email string
		var exp time.Time
		if err := rows.Scan(&id, &tenant, &email, &exp); err != nil {
			continue
		}
		_ = m.notifier.Notify(ctx, Notification{
			Channel: ChannelEmail, Recipient: email, Template: "payment_reminder",
			Vars: map[string]string{"order_id": id.String()[:8], "expiry": exp.Format("2006-01-02 15:04")},
		})
		count++
	}
	if count > 0 {
		m.log.Info("EH-2 auto-chase sweep sent reminders", "count", count)
	}
}

// ---------- helpers ----------

func (m *Module) resolveOrCreateCategory(ctx context.Context, tenantID uuid.UUID, name string) string {
	cats, _ := m.catalog.ListCategories(ctx, tenantID)
	for _, ct := range cats {
		if strings.EqualFold(ct.Name, name) {
			return ct.ID.String()
		}
	}
	if cat, err := m.catalog.CreateCategory(ctx, tenantID, name, ""); err == nil {
		return cat.ID.String()
	}
	return ""
}

func (m *Module) resolveOrderID(ctx context.Context, tenantID uuid.UUID, target string) (uuid.UUID, error) {
	if id, err := uuid.Parse(target); err == nil {
		return id, nil
	}
	// allow an id prefix (e.g. from "ship order 1a2b3c")
	var id uuid.UUID
	err := m.pool.QueryRow(ctx,
		`SELECT id FROM orders WHERE tenant_id=$1 AND id::text LIKE $2 LIMIT 1`,
		tenantID, target+"%").Scan(&id)
	if err != nil {
		return uuid.Nil, httpx.NotFound("order not found for: " + target)
	}
	return id, nil
}

func deriveSKU(name string) string {
	up := strings.ToUpper(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range up {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
		if b.Len() >= 8 {
			break
		}
	}
	if b.Len() == 0 {
		return "SKU" + uuid.NewString()[:6]
	}
	return b.String() + "-" + uuid.NewString()[:4]
}

func uuidPtrToStr(p *uuid.UUID) string {
	if p == nil {
		return ""
	}
	return p.String()
}
