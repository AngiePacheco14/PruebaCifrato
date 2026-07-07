package repositories_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"cifrato/internal/adapters/driven/postgres"
	"cifrato/internal/adapters/driven/postgres/repositories"
	"cifrato/internal/domain/invoice"
)

func TestInvoiceRoundTrip(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set, skipping integration test (requires docker-compose postgres)")
	}

	db, err := postgres.Open(postgres.ConfigFromEnv())
	if err != nil {
		t.Fatalf("opening connection: %v", err)
	}
	if err := postgres.Migrate(db); err != nil {
		t.Fatalf("migrating: %v", err)
	}

	repo := repositories.NewInvoiceRepository(db)
	ctx := context.Background()

	inv := &invoice.Invoice{
		CUFE:          "test-cufe-roundtrip",
		InvoiceNumber: "TEST-001",
		IssueDate:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		XMLType:       invoice.XMLTypeInvoice,
		IssuerNIT:     "900000000",
		IssuerName:    "Proveedor de Prueba",
		BuyerNIT:      "800000000",
		BuyerName:     "Comprador de Prueba",
		Subtotal:      decimal.NewFromInt(100000),
		IVATotal:      decimal.NewFromInt(19000),
		InvoiceTotal:  decimal.NewFromInt(119000),
		Lines: []invoice.InvoiceLine{
			{
				LineNumber:  1,
				Description: "MANTEQUILLA PERFUMADA 240 ML",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(100000),
				LineTotal:   decimal.NewFromInt(100000),
				IVARate:     decimal.NewFromInt(19),
				IVAValue:    decimal.NewFromInt(19000),
			},
		},
	}

	if err := repo.Save(ctx, inv); err != nil {
		t.Fatalf("saving invoice: %v", err)
	}
	if inv.ID == 0 {
		t.Fatal("expected invoice ID to be set after save")
	}

	found, err := repo.FindByCUFE(ctx, "test-cufe-roundtrip")
	if err != nil {
		t.Fatalf("finding invoice: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find invoice by cufe")
	}
	if len(found.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(found.Lines))
	}
	if found.Lines[0].Description != "MANTEQUILLA PERFUMADA 240 ML" {
		t.Fatalf("unexpected line description: %s", found.Lines[0].Description)
	}

	exists, err := repo.ExistsByCUFE(ctx, "test-cufe-roundtrip")
	if err != nil {
		t.Fatalf("checking existence: %v", err)
	}
	if !exists {
		t.Fatal("expected invoice to exist")
	}
}
