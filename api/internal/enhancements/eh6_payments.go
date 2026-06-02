package enhancements

import (
	"context"
	"errors"
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

// SupportedCurrencies for the multi-currency extension (display/convert only
// until a gateway is integrated).
var SupportedCurrencies = []string{"MYR", "SGD", "USD"}

// Convert is a placeholder FX conversion (identity until rates wired).
func Convert(m Money, to string) Money {
	if m.Currency == to {
		return m
	}
	return Money{Amount: m.Amount, Currency: to} // TODO: real FX rates (EH-6)
}
