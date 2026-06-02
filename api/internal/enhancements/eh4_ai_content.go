package enhancements

import (
	"context"
	"strings"
)

// EH-4 — AI content: product descriptions, SEO, auto-categorization, exposed as
// a Seller Copilot tool (EH-1). Builds on Catalog + an LLM. Disabled in v1.

type ContentRequest struct {
	ProductName string
	Keywords    []string
	Tone        string // "friendly" | "formal"
}

type ContentResult struct {
	Description     string
	SEOTitle        string
	SEOMeta         string
	SuggestedCategory string
}

// ContentGenerator produces marketing copy. The scaffold returns a deterministic
// template; production swaps in an LLM behind the same interface.
type ContentGenerator interface {
	Generate(ctx context.Context, req ContentRequest) (ContentResult, error)
}

type TemplateContentGenerator struct{}

func (TemplateContentGenerator) Generate(ctx context.Context, req ContentRequest) (ContentResult, error) {
	kw := strings.Join(req.Keywords, ", ")
	return ContentResult{
		Description:       "Produk " + req.ProductName + " berkualiti. " + kw,
		SEOTitle:          req.ProductName + " | Beli dalam talian",
		SEOMeta:           "Beli " + req.ProductName + " — " + kw,
		SuggestedCategory: suggestCategory(req.ProductName),
	}, nil
}

func suggestCategory(name string) string {
	n := strings.ToLower(name)
	switch {
	case containsAny(n, "kuih", "kek", "cake", "biskut"):
		return "Makanan"
	case containsAny(n, "teh", "kopi", "air", "drink"):
		return "Minuman"
	case containsAny(n, "baju", "shirt", "tudung"):
		return "Pakaian"
	default:
		return "Lain-lain"
	}
}
