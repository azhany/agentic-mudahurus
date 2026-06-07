// Command seed populates the database with demo accounts and data for testing
// the three personas (PRD §4): Platform Operator (super admin), Seller (store
// owner/admin), and Customer (public storefront buyer — unauthenticated).
//
// Idempotent: re-running resets the demo account passwords and only seeds a
// store's catalog/orders when that store is empty. Safe to run repeatedly.
//
//	cd api && go run ./cmd/seed                 # uses DATABASE_URL / .env defaults
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/mudahurus/api/internal/auth"
	"github.com/mudahurus/api/internal/catalog"
	"github.com/mudahurus/api/internal/config"
	"github.com/mudahurus/api/internal/coupons"
	"github.com/mudahurus/api/internal/customers"
	"github.com/mudahurus/api/internal/db"
	"github.com/mudahurus/api/internal/orders"
)

func main() {
	cfg, err := config.Load()
	must(err)

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	must(err)
	defer pool.Close()

	// Ensure schema exists (no-op if the API already migrated).
	must(db.Migrate(ctx, pool))

	s := &seeder{
		pool:      pool,
		catalog:   catalog.NewRepo(pool),
		customers: customers.NewRepo(pool),
		coupons:   coupons.NewRepo(pool),
		orders:    orders.NewRepo(pool),
	}

	fmt.Println("Seeding MUDAHURUS demo data…")

	// 1) Super Admin (platform operator)
	s.upsertTenant("superadmin", "superadmin@mudahurus.my", "superadmin123", "operator", "MUDAHURUS HQ")

	// 2) Store owner #1 — full demo store
	ali := s.upsertTenant("kedaiali", "ali@mudahurus.my", "kedaiali123", "seller", "Kedai Ali Online")
	s.seedKedaiAli(ctx, ali)

	// 3) Store owner #2 — proves multi-tenant isolation
	siti := s.upsertTenant("butiksiti", "siti@mudahurus.my", "butiksiti123", "seller", "Butik Siti")
	s.seedButikSiti(ctx, siti)

	fmt.Println("\n✅ Seed complete. Test accounts:")
	fmt.Println("  Super Admin (operator): superadmin / superadmin123")
	fmt.Println("  Store Owner (seller):   kedaiali   / kedaiali123   → store: /store/kedaiali")
	fmt.Println("  Store Owner (seller):   butiksiti  / butiksiti123  → store: /store/butiksiti")
	fmt.Println("  Customer (no login):    shop the public storefront at /store/kedaiali and checkout as guest")
}

type seeder struct {
	pool      *db.Pool
	catalog   *catalog.Repo
	customers *customers.Repo
	coupons   *coupons.Repo
	orders    *orders.Repo
}

// upsertTenant inserts or updates a tenant by username, (re)setting its password
// so demo credentials are always known. Returns the tenant id.
func (s *seeder) upsertTenant(username, email, password, role, storeName string) uuid.UUID {
	hash, err := auth.HashPassword(password)
	must(err)
	var id uuid.UUID
	err = s.pool.QueryRow(context.Background(), `
		INSERT INTO tenants (username, email, password_hash, role, store_name, email_verified)
		VALUES ($1,$2,$3,$4,$5,true)
		ON CONFLICT (username) DO UPDATE
		  SET email=EXCLUDED.email, password_hash=EXCLUDED.password_hash,
		      role=EXCLUDED.role, store_name=EXCLUDED.store_name, email_verified=true
		RETURNING id`,
		username, email, hash, role, storeName).Scan(&id)
	must(err)
	fmt.Printf("  • tenant %-10s (%s)\n", username, role)
	return id
}

func (s *seeder) hasProducts(ctx context.Context, tenantID uuid.UUID) bool {
	var n int
	_ = s.pool.QueryRow(ctx, `SELECT count(*) FROM products WHERE tenant_id=$1`, tenantID).Scan(&n)
	return n > 0
}

func (s *seeder) seedKedaiAli(ctx context.Context, tenantID uuid.UUID) {
	if s.hasProducts(ctx, tenantID) {
		fmt.Println("    (kedaiali already has catalog — skipping data seed)")
		return
	}
	makanan, err := s.catalog.CreateCategory(ctx, tenantID, "Makanan", "Makanan & snek")
	must(err)
	minuman, err := s.catalog.CreateCategory(ctx, tenantID, "Minuman", "Minuman panas & sejuk")
	must(err)

	prods := []catalog.Product{
		{SKU: "KL01", ProductName: "Kuih Lapis", Description: "Kuih lapis tradisional berlapis-lapis", UnitPrice: 12.50, CategoryID: &makanan.ID, Status: "active", URLSlug: "kuih-lapis"},
		{SKU: "KR01", ProductName: "Karipap Pusing", Description: "Karipap rangup inti kentang", UnitPrice: 0.80, CategoryID: &makanan.ID, Status: "active", URLSlug: "karipap-pusing"},
		{SKU: "TT01", ProductName: "Teh Tarik", Description: "Teh tarik panas", UnitPrice: 3.00, CategoryID: &minuman.ID, Status: "active", URLSlug: "teh-tarik"},
		{SKU: "KO01", ProductName: "Kopi O Panas", Description: "Kopi hitam pekat", UnitPrice: 2.50, CategoryID: &minuman.ID, Status: "active", URLSlug: "kopi-o-panas"},
		{SKU: "RAYA01", ProductName: "Hamper Raya Premium", Description: "Set hamper raya (draf, belum aktif)", UnitPrice: 159.00, CategoryID: &makanan.ID, Status: "inactive", URLSlug: "hamper-raya-premium"},
	}
	for i := range prods {
		prods[i].TenantID = tenantID
		must(s.catalog.CreateProduct(ctx, &prods[i]))
	}

	cust := []customers.Customer{
		{FullName: "Aminah binti Razak", Email: "aminah@example.com", ContactNo: "0191234567", City: "Kuala Lumpur", State: "WP", Postcode: "50000", LoyaltyCode: "ALI-0001", Type: "regular"},
		{FullName: "Hafiz bin Omar", Email: "hafiz@example.com", ContactNo: "0137654321", City: "Shah Alam", State: "Selangor", Postcode: "40000", LoyaltyCode: "ALI-0002", Type: "vip"},
	}
	for i := range cust {
		cust[i].TenantID = tenantID
		must(s.customers.Create(ctx, &cust[i]))
	}

	exp := time.Now().Add(30 * 24 * time.Hour)
	must(s.coupons.Create(ctx, &coupons.Coupon{TenantID: tenantID, Campaign: "RAYA2026", Description: "Diskaun Raya 10%", ProductID: &prods[0].ID, ExpiredDate: &exp}))

	// Order #1 — pending (no payment yet): exercises the guest/pending flow + EH-2 chase.
	must(s.orders.Create(ctx, &orders.Order{
		TenantID: tenantID, CustomerID: &cust[0].ID, Status: "pending",
		FullName: "Aminah binti Razak", Email: "aminah@example.com", ContactNo: "0191234567",
		ShippingAddress: orders.ShippingAddress{MailingAddr: "No 1, Jln Mawar", City: "Kuala Lumpur", Postcode: "50000", State: "WP"},
		Items: []orders.OrderItem{{SKU: "KL01", ProductName: "Kuih Lapis", Quantity: 2, UnitPrice: 12.50}},
	}))

	// Order #2 — shipped + verified payment: exercises the full money-flow.
	shipped := &orders.Order{
		TenantID: tenantID, CustomerID: &cust[1].ID, Status: "shipped",
		FullName: "Hafiz bin Omar", Email: "hafiz@example.com", ContactNo: "0137654321",
		ShippingAddress: orders.ShippingAddress{MailingAddr: "12, Jln Melati", City: "Shah Alam", Postcode: "40000", State: "Selangor"},
		Items: []orders.OrderItem{{SKU: "TT01", ProductName: "Teh Tarik", Quantity: 3, UnitPrice: 3.00}},
	}
	must(s.orders.Create(ctx, shipped))
	pay, err := s.orders.AddPayment(ctx, shipped.ID, "", 9.00)
	must(err)
	must(s.orders.SetPaymentStatus(ctx, tenantID, pay.ID, "verified"))

	fmt.Println("    seeded: 2 categories, 5 products, 2 customers, 1 coupon, 2 orders")
}

func (s *seeder) seedButikSiti(ctx context.Context, tenantID uuid.UUID) {
	if s.hasProducts(ctx, tenantID) {
		fmt.Println("    (butiksiti already has catalog — skipping data seed)")
		return
	}
	pakaian, err := s.catalog.CreateCategory(ctx, tenantID, "Pakaian", "Pakaian muslimah")
	must(err)
	prods := []catalog.Product{
		{SKU: "TDG01", ProductName: "Tudung Bawal Premium", Description: "Cotton voile 45 inci", UnitPrice: 29.90, CategoryID: &pakaian.ID, Status: "active", URLSlug: "tudung-bawal-premium"},
		{SKU: "BJ01", ProductName: "Baju Kurung Moden", Description: "Baju kurung pesak gantung", UnitPrice: 89.00, CategoryID: &pakaian.ID, Status: "active", URLSlug: "baju-kurung-moden"},
	}
	for i := range prods {
		prods[i].TenantID = tenantID
		must(s.catalog.CreateProduct(ctx, &prods[i]))
	}
	fmt.Println("    seeded: 1 category, 2 products")
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed error:", err)
		os.Exit(1)
	}
}
