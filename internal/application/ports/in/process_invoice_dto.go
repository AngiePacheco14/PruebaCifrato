package in

import "cifrato/internal/domain/entity"

// ProcessInvoiceResult bundles the persisted invoice (with lines classified
// where possible), the withholding calculations produced for it, and a
// per-tax-type summary rolled up across all of its lines.
type ProcessInvoiceResult struct {
	Invoice      *entity.Invoice
	Calculations []entity.Calculation
	Summary      entity.CalculationSummary
}
