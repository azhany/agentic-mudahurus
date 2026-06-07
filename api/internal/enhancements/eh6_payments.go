package enhancements

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// EH-6 — Payment gateway integrations + multi-currency. Builds on Payments. v1
// retains the manual payment-proof flow; this scaffold defines the gateway
// contract so a real provider (e.g. iPay88, Stripe) slots in later. Disabled.

type Money struct {
	Amount   int64  // minor units (e.g. sen)
	Currency string // ISO 4217, e.g. "MYR"
}

type ChargeRequest struct {
	OrderID  string
	Amount   Money
	Customer string
}

type ChargeResult struct {
	GatewayRef string
	Status     string // pending | paid | failed
	RedirectURL string
}

var ErrGatewayDisabled = errors.New("payment gateway disabled in v1 (manual proof flow)")

// Gateway abstracts a payment provider.
type Gateway interface {
	CreateCharge(ctx context.Context, req ChargeRequest) (ChargeResult, error)
}

// ManualGateway is the v1 behaviour: no real charge, customers upload proof.
type ManualGateway struct{}

func (ManualGateway) CreateCharge(ctx context.Context, req ChargeRequest) (ChargeResult, error) {
	return ChargeResult{}, ErrGatewayDisabled
}

// MockGateway simulates a hosted-payment-page provider (e.g. iPay88/Stripe).
// CreateCharge returns a gateway ref + a redirect URL to a stub hosted page; a
// webhook later confirms the charge. Demonstrates the integration shape without
// a real provider — swap in a concrete client behind the Gateway interface.
type MockGateway struct {
	BaseURL string // public base for the hosted-page redirect
}

func (g MockGateway) CreateCharge(ctx context.Context, req ChargeRequest) (ChargeResult, error) {
	ref := "mock_" + uuid.NewString()
	return ChargeResult{
		GatewayRef:  ref,
		Status:      "pending",
		RedirectURL: g.BaseURL + "/pay/mock/" + ref + "?order=" + req.OrderID,
	}, nil
}

// SupportedCurrencies for the multi-currency extension.
var SupportedCurrencies = []string{"MYR", "SGD", "USD"}

// fxFromMYR holds static indicative rates (1 MYR -> X). A production system
// would refresh these from an FX provider; the contract stays the same.
var fxFromMYR = map[string]float64{
	"MYR": 1.0,
	"SGD": 0.30,
	"USD": 0.22,
}

// Convert changes the currency of a Money amount (minor units) using the static
// rate table, going via the MYR base. Unknown currencies pass through unchanged.
func Convert(m Money, to string) Money {
	to = strings.ToUpper(to)
	from := strings.ToUpper(m.Currency)
	if from == to {
		return m
	}
	rFrom, okF := fxFromMYR[from]
	rTo, okT := fxFromMYR[to]
	if !okF || !okT || rFrom == 0 {
		return Money{Amount: m.Amount, Currency: to}
	}
	myr := float64(m.Amount) / rFrom // back to MYR base
	converted := myr * rTo
	return Money{Amount: int64(converted + 0.5), Currency: to}
}
