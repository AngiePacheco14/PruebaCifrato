package out

import (
	"context"

	"cifrato/internal/domain/withholding"
)

type CalculationRepository interface {
	// Upsert overwrites the previous calculation for (InvoiceLineID, TaxType).
	Upsert(ctx context.Context, calc *withholding.Calculation) error
	ListByInvoice(ctx context.Context, invoiceID uint) ([]withholding.Calculation, error)
}
