package enhancements

import (
	"context"
	"sort"
)

// EH-5 — Recommendations & analytics ("why did sales drop", upsell). Builds on
// Orders + the RAG plane. Disabled in v1.

type SalesPoint struct {
	Period string // e.g. "2026-05"
	Total  float64
}

type Insight struct {
	Headline string
	Detail   string
}

// AnalyzeTrend is a pure helper: flags a material month-over-month sales drop.
func AnalyzeTrend(points []SalesPoint) []Insight {
	if len(points) < 2 {
		return nil
	}
	sorted := append([]SalesPoint(nil), points...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Period < sorted[j].Period })
	prev := sorted[len(sorted)-2]
	cur := sorted[len(sorted)-1]
	var insights []Insight
	if prev.Total > 0 {
		change := (cur.Total - prev.Total) / prev.Total
		if change <= -0.2 {
			insights = append(insights, Insight{
				Headline: "Sales dropped",
				Detail:   "Revenue fell materially vs the previous period; investigate top SKUs and pending orders.",
			})
		} else if change >= 0.2 {
			insights = append(insights, Insight{Headline: "Sales up", Detail: "Revenue grew vs the previous period."})
		}
	}
	return insights
}

// Recommender suggests upsell products for a cart/customer.
type Recommender interface {
	Upsell(ctx context.Context, tenantID string, productIDs []string) ([]string, error)
}
