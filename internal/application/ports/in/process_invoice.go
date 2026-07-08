package in

import "context"

// ProcessInvoice runs the full pipeline for one UBL DIAN invoice XML: parse,
// persist, classify lines, and calculate withholdings. Implemented by
// usecase.ProcessInvoice.
type ProcessInvoice interface {
	Execute(ctx context.Context, xmlData []byte, sourceXMLPath, sourcePDFPath string) (*ProcessInvoiceResult, error)
}
