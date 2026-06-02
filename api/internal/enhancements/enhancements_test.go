package enhancements

import (
	"testing"
	"time"
)

func TestFlagsDefaultOff(t *testing.T) {
	for _, f := range []string{FlagOrchestration, FlagFulfillment, FlagNotifications, FlagAIContent, FlagRecommendations, FlagPayments} {
		if Enabled(f) {
			t.Errorf("flag %s should default OFF in v1", f)
		}
	}
}

func TestEH1RouterRouting(t *testing.T) {
	r := NewRouter()
	if got := r.Route(AgentRequest{Scope: "storefront", Message: "do you sell teh?"}); got != AgentStorefrontAssistant {
		t.Errorf("storefront scope should route to assistant, got %s", got)
	}
	if got := r.Route(AgentRequest{Role: "seller", Scope: "admin", Message: "track my poslaju shipment"}); got != AgentFulfillment {
		t.Errorf("tracking intent should route to fulfillment, got %s", got)
	}
	if got := r.Route(AgentRequest{Role: "seller", Scope: "admin", Message: "update price of SKU123"}); got != AgentSellerCopilot {
		t.Errorf("write intent should route to copilot, got %s", got)
	}
}

func TestEH2DecideChase(t *testing.T) {
	now := time.Now()
	d := DecideChase("o1", "pending", now.Add(12*time.Hour), false, now)
	if !d.ShouldChase {
		t.Error("unpaid pending order expiring in 12h should be chased")
	}
	d2 := DecideChase("o2", "pending", now.Add(12*time.Hour), true, now)
	if d2.ShouldChase {
		t.Error("paid order should not be chased")
	}
	d3 := DecideChase("o3", "shipped", now.Add(12*time.Hour), false, now)
	if d3.ShouldChase {
		t.Error("non-pending order should not be chased")
	}
}

func TestEH4CategorySuggestion(t *testing.T) {
	res, _ := TemplateContentGenerator{}.Generate(nil, ContentRequest{ProductName: "Kuih Lapis", Keywords: []string{"manis"}})
	if res.SuggestedCategory != "Makanan" {
		t.Errorf("expected Makanan, got %s", res.SuggestedCategory)
	}
}

func TestEH5TrendDrop(t *testing.T) {
	insights := AnalyzeTrend([]SalesPoint{{"2026-04", 1000}, {"2026-05", 500}})
	if len(insights) == 0 || insights[0].Headline != "Sales dropped" {
		t.Errorf("expected a sales-drop insight, got %+v", insights)
	}
}

func TestEH6ManualGatewayDisabled(t *testing.T) {
	_, err := ManualGateway{}.CreateCharge(nil, ChargeRequest{})
	if err != ErrGatewayDisabled {
		t.Errorf("manual gateway should report disabled, got %v", err)
	}
}
