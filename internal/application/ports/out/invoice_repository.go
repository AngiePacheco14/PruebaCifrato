package out

import (
	"context"

	"cifrato/internal/domain/invoice"
)

type InvoiceRepository interface {
	// Save upserts by CUFE and replaces existing lines; re-importing the
	// same invoice does not duplicate lines.
	Save(ctx context.Context, inv *invoice.Invoice) error
	FindByCUFE(ctx context.Context, cufe string) (*invoice.Invoice, error)
	ExistsByCUFE(ctx context.Context, cufe string) (bool, error)
}
