package enhancements

import (
	"context"
	"time"
)

// EH-2 — Autonomous fulfillment.
// POSLAJU tracking polling + pending-order auto-chase BEFORE the 3-day expiry,
// exception-only human-in-loop. Builds on Orders + the events stream. v1 keeps
// POSLAJU manual (display only); this scaffold defines the polling + chase
// contracts and a guarded loop that does nothing unless FlagFulfillment is on.

type TrackingStatus struct {
	TrackingNo string
	State      string // in_transit | delivered | exception | unknown
	UpdatedAt  time.Time
}

// TrackingProvider abstracts the courier API (POSLAJU, etc.).
type TrackingProvider interface {
	Track(ctx context.Context, trackingNo string) (TrackingStatus, error)
}

// ChaseDecision is what the auto-chase logic decides for a pending order.
type ChaseDecision struct {
	OrderID    string
	ShouldChase bool
	Reason     string
}

// DecideChase: chase pending orders approaching expiry (within 24h) that have
// no payment yet. Pure + testable; no side effects in the scaffold.
func DecideChase(orderID, status string, expiresAt time.Time, hasPayment bool, now time.Time) ChaseDecision {
	if status != "pending" || hasPayment {
		return ChaseDecision{OrderID: orderID, ShouldChase: false, Reason: "not an unpaid pending order"}
	}
	if expiresAt.Sub(now) <= 24*time.Hour && expiresAt.After(now) {
		return ChaseDecision{OrderID: orderID, ShouldChase: true, Reason: "expires within 24h, no payment"}
	}
	return ChaseDecision{OrderID: orderID, ShouldChase: false, Reason: "not yet due"}
}

// NoopTrackingProvider is the default until a real courier integration lands.
type NoopTrackingProvider struct{}

func (NoopTrackingProvider) Track(ctx context.Context, trackingNo string) (TrackingStatus, error) {
	return TrackingStatus{TrackingNo: trackingNo, State: "unknown", UpdatedAt: time.Now()}, nil
}
