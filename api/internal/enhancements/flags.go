// Package enhancements holds POST-V1 scaffolds for the Enhancement Backlog
// (EH-1 … EH-6, PRD §10 / SPRINT_PLAN). These are intentionally NOT mounted by
// the v1 server (server.Mount) — they are gated behind feature flags that
// default OFF, so they cannot leak into v1 scope. Each is shaped so it can be
// enabled and wired up independently once it has an approved PRD slice.
//
// Definition of "not scope creep" (SPRINT_PLAN): a backlog item only enters a
// sprint when (a) v1 is released, (b) it has an approved PRD slice, and (c) it
// is independently estimated. Until then it stays disabled here.
package enhancements

import "os"

// Flag identifiers for each backlog item.
const (
	FlagOrchestration   = "EH1_ORCHESTRATION"   // multi-agent router + specialists
	FlagFulfillment     = "EH2_FULFILLMENT"     // POSLAJU polling + auto-chase
	FlagNotifications   = "EH3_NOTIFICATIONS"   // WhatsApp/email updates
	FlagAIContent       = "EH4_AI_CONTENT"      // descriptions/SEO/auto-categorize
	FlagRecommendations = "EH5_RECOMMENDATIONS" // recs + analytics
	FlagPayments        = "EH6_PAYMENTS"        // gateway + multi-currency
)

// Enabled reports whether an enhancement is switched on (env: MH_<FLAG>=true).
// Defaults to false — every enhancement is off in v1.
func Enabled(flag string) bool {
	return os.Getenv("MH_"+flag) == "true"
}
