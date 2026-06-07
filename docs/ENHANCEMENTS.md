# Enhancement Backlog (EH-1 … EH-6) — Reference

These were **scaffolded** during v1 and have now been **implemented and mounted**.
They build on the v1 foundation (RAG plane, Orders, Catalog, events) exactly as
the PRD §10 / SPRINT_PLAN backlog intended.

> **Scope guard still honoured:** seller-invoked endpoints only ever act on the
> authenticated caller's own tenant. The single piece of *autonomous* behaviour
> (EH-2's background auto-chase) is OFF unless `MH_EH2_FULFILLMENT=true`.

## Feature flags / env
| Env var | Effect | Default |
|---|---|---|
| `MH_EH2_FULFILLMENT` | enable the background auto-chase sweep | off |
| `MH_EH2_CHASE_INTERVAL` | sweep interval (Go duration, e.g. `1h`) | `1h` |
| `MH_WHATSAPP_WEBHOOK` | WhatsApp Business API URL (else logs) | — |
| `MH_GATEWAY_SECRET` | required `X-Gateway-Signature` on payment webhook | — |
| `API_PUBLIC_BASE_URL` | base for payment redirect URLs | `http://localhost<addr>` |

(The `MH_EH1/3/4/5/6_*` flags from `flags.go` remain available for gating any
future destructive extensions; the request-driven endpoints below are always
mounted.)

## Endpoints (all admin endpoints require a seller JWT)

> All paths below are under the **`/api`** namespace (e.g. the full path for the
> first one is `POST /api/copilot/interpret`). Omitted here for brevity.

### EH-1 — Seller Copilot (multi-agent orchestration)
- `POST /copilot/interpret` `{ "message": "add product Kuih Lapis price RM12.50 category Makanan" }`
  → `{ agent, actions:[{Kind,Target,Params}] }` (parses NL; never executes)
- `POST /copilot/execute` `{ "kind":"create_product", "params":{...} }`
  → executes via the domain services (create_product, update_product_price,
  advance_order_status, generate_content). Human-in-the-loop by design.
- UI: **Admin → Copilot** (`web/src/views/Copilot.vue`).
- RAG-plane router + specialist agents: `rag/mudahurus_rag/enhancements/orchestration.py`.

### EH-2 — Autonomous fulfillment
- `GET /fulfillment/chase-candidates?notify=true` — unpaid pending orders within
  24h of the 3-day expiry; optionally sends reminders.
- `POST /fulfillment/track` `{ "tracking_no":"EP..." }` — courier status (POSLAJU
  provider is a stub returning `unknown` until a real integration is added).
- Background sweep (flagged) chases across all tenants on an interval.

### EH-3 — Notifications
- `POST /notifications/send` `{ channel:"email|whatsapp", recipient, template, vars }`.
- Templates (BM): `order_confirmed`, `payment_reminder`, `shipped`.
- Auto-sent: payment confirmation (on webhook) + chase reminders (EH-2).

### EH-4 — AI content (Copilot tool)
- `POST /copilot/generate-content` `{ product_name, keywords[], tone }`
  → `{ Description, SEOTitle, SEOMeta, SuggestedCategory }`.

### EH-5 — Recommendations & analytics
- `GET /analytics/insights` → monthly sales series + auto-detected drop/growth.
- `GET /recommendations?product_id=<uuid>` → same-category upsell suggestions.

### EH-6 — Payments + multi-currency
- `GET /currencies` → supported currencies (MYR base).
- `POST /orders/:id/charge?currency=USD` → creates a gateway charge (mock),
  converts MYR→currency, returns `redirect_url` to a hosted page.
- `GET /pay/mock/:ref` → mock hosted payment page (dev).
- `POST /payments/webhook` (public) → on `status:"paid"`: marks payment
  `verified`, advances the order to `payment_accepted`, sends confirmation.
- Swap `MockGateway` for a real `Gateway` impl (iPay88/Stripe) — same interface;
  the manual payment-proof flow (v1) remains the default.

## Tests
- Go: `api/internal/enhancements/enhancements_test.go` (router, NL parser, FX
  conversion, templates, chase decision, gateways).
- Python: `rag/tests/test_enhancements.py` (router + copilot proposes-not-executes).
