package in

import "cifrato/internal/domain/entity"

// ProcessInvoiceResult bundles the persisted invoice (with lines classified
// where possible) and the withholding calculations produced for it.
type ProcessInvoiceResult struct {
	Invoice      *entity.Invoice
	Calculations []entity.Calculation
}
