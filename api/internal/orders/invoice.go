package orders

import (
	"bytes"
	"fmt"
	"strings"
)

// Invoice is the data projection rendered for /invoice/{id} (FR-3.4).
type Invoice struct {
	Order      *Order `json:"order"`
	SellerName string `json:"seller_name"`
	StoreName  string `json:"store_name"`
}

// RenderPDF produces a minimal, valid PDF (PDF 1.4, Helvetica) for the invoice.
// Self-contained — no external PDF dependency.
func (inv *Invoice) RenderPDF() []byte {
	var lines []string
	add := func(f string, a ...any) { lines = append(lines, fmt.Sprintf(f, a...)) }

	store := inv.StoreName
	if store == "" {
		store = inv.SellerName
	}
	add("INVOICE")
	add("%s", store)
	add("")
	add("Invoice ID: %s", inv.Order.ID)
	add("Status: %s", inv.Order.Status)
	add("Date: %s", inv.Order.CreatedAt.Format("2006-01-02 15:04"))
	add("Due (expires): %s", inv.Order.ExpiredDate.Format("2006-01-02 15:04"))
	add("")
	add("Bill to: %s", inv.Order.FullName)
	add("Email: %s   Phone: %s", inv.Order.Email, inv.Order.ContactNo)
	a := inv.Order.ShippingAddress
	add("Ship to: %s %s, %s %s %s", a.MailingAddr, a.MailingAddr2, a.Postcode, a.City, a.State)
	add("")
	add("%-30s %5s %12s %12s", "Item", "Qty", "Unit", "Total")
	add(strings.Repeat("-", 62))
	for _, it := range inv.Order.Items {
		name := it.ProductName
		if name == "" {
			name = it.SKU
		}
		add("%-30s %5d %12.2f %12.2f", truncate(name, 30), it.Quantity, it.UnitPrice, it.LineTotal)
	}
	add(strings.Repeat("-", 62))
	add("%-48s %12.2f", "TOTAL (MYR)", inv.Order.TotalPrice)
	if inv.Order.AdditionalNotes != "" {
		add("")
		add("Notes: %s", truncate(inv.Order.AdditionalNotes, 80))
	}

	return buildPDF(lines)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}

// buildPDF lays the given text lines onto a single A4 page.
func buildPDF(lines []string) []byte {
	var content bytes.Buffer
	content.WriteString("BT\n/F1 11 Tf\n12 TL\n50 800 Td\n")
	for i, ln := range lines {
		if i > 0 {
			content.WriteString("T*\n")
		}
		content.WriteString("(" + escapePDF(ln) + ") Tj\n")
	}
	content.WriteString("ET")

	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Courier >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", content.Len(), content.String()),
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs)+1)
	for i, o := range objs {
		offsets[i+1] = buf.Len()
		buf.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", i+1, o))
	}
	xrefPos := buf.Len()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n", len(objs)+1))
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objs); i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	buf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF",
		len(objs)+1, xrefPos))
	return buf.Bytes()
}

func escapePDF(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)")
	// drop non-ASCII to keep the Courier base font happy
	var b strings.Builder
	for _, c := range s {
		if c < 128 {
			b.WriteRune(c)
		} else {
			b.WriteByte('?')
		}
	}
	return r.Replace(b.String())
}
