package orders

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		ok       bool
	}{
		{"pending", "payment_received", true},
		{"pending", "shipped", false},
		{"payment_received", "payment_accepted", true},
		{"payment_accepted", "shipped", true},
		{"shipped", "pending", false},
		{"cancelled", "shipped", false},
	}
	for _, c := range cases {
		if got := CanTransition(c.from, c.to); got != c.ok {
			t.Errorf("CanTransition(%s,%s)=%v want %v", c.from, c.to, got, c.ok)
		}
	}
}

func TestRenderPDF(t *testing.T) {
	inv := &Invoice{
		StoreName: "Kedai Ali",
		Order: &Order{
			ID:          uuid.New(),
			Status:      "pending",
			FullName:    "Siti",
			TotalPrice:  59.80,
			CreatedAt:   time.Now(),
			ExpiredDate: time.Now().Add(72 * time.Hour),
			Items: []OrderItem{
				{SKU: "ABC", ProductName: "Kuih", Quantity: 2, UnitPrice: 29.90, LineTotal: 59.80},
			},
		},
	}
	pdf := inv.RenderPDF()
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Fatal("output is not a PDF")
	}
	if !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Fatal("PDF missing EOF marker")
	}
	if !bytes.Contains(pdf, []byte("Kedai Ali")) {
		t.Fatal("PDF missing store name")
	}
}
