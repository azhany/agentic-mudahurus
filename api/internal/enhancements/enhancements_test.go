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

func TestEH1ParseSellerCommand(t *testing.T) {
	a := ParseSellerCommand("add product Kuih Lapis price RM12.50 category Makanan")
	if len(a) != 1 || a[0].Kind != "create_product" {
		t.Fatalf("expected create_product, got %+v", a)
	}
	if a[0].Params["product_name"] != "Kuih Lapis" || a[0].Params["unit_price"] != "12.50" || a[0].Params["category"] != "Makanan" {
		t.Errorf("bad parse params: %+v", a[0].Params)
	}

	b := ParseSellerCommand("set price of KL01 to RM9.90")
	if len(b) != 1 || b[0].Kind != "update_product_price" || b[0].Params["sku"] != "KL01" || b[0].Params["unit_price"] != "9.90" {
		t.Errorf("bad set-price parse: %+v", b)
	}

	c := ParseSellerCommand("ship order 1a2b3c4d")
	if len(c) != 1 || c[0].Kind != "advance_order_status" || c[0].Params["status"] != "shipped" {
		t.Errorf("bad ship parse: %+v", c)
	}

	if got := ParseSellerCommand("what is the meaning of life"); len(got) != 0 {
		t.Errorf("non-command should yield no actions, got %+v", got)
	}
}

func TestEH6Convert(t *testing.T) {
	// RM10.00 = 1000 sen -> USD at 0.22 -> 220 minor units
	got := Convert(Money{Amount: 1000, Currency: "MYR"}, "USD")
	if got.Currency != "USD" || got.Amount != 220 {
		t.Errorf("expected 220 USD minor units, got %+v", got)
	}
	if same := Convert(Money{Amount: 500, Currency: "MYR"}, "MYR"); same.Amount != 500 {
		t.Errorf("identity conversion failed: %+v", same)
	}
}

func TestEH3RenderTemplate(t *testing.T) {
	out := Render("payment_reminder", map[string]string{"order_id": "abc12345", "expiry": "2026-06-05 12:00"})
	if want := "abc12345"; !contains(out, want) {
		t.Errorf("rendered body missing %q: %s", want, out)
	}
}

func TestEH6MockGatewayCharge(t *testing.T) {
	res, err := MockGateway{BaseURL: "http://x"}.CreateCharge(nil, ChargeRequest{OrderID: "o1"})
	if err != nil || res.GatewayRef == "" || res.Status != "pending" || res.RedirectURL == "" {
		t.Errorf("unexpected charge result: %+v err=%v", res, err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
