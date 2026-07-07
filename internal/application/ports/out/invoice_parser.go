package out

import "cifrato/internal/domain/invoice"

// InvoiceParser parses raw UBL DIAN invoice XML bytes — either a direct
// Invoice or an AttachedDocument wrapping one — into the domain model. It
// does not populate SourceXMLPath/SourcePDFPath; the caller (future import
// use case) knows the file path and fills those in after a successful parse.
type InvoiceParser interface {
	Parse(xmlData []byte) (*invoice.Invoice, error)
}
