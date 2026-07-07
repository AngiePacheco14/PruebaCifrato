package invoice

import (
	"time"

	"github.com/shopspring/decimal"
)

type Invoice struct {
	ID                      uint
	CUFE                    string
	InvoiceNumber           string
	IssueDate               time.Time
	XMLType                 XMLType
	IssuerNIT               string
	IssuerName              string
	IssuerCity              string
	IssuerTaxResponsibility string
	BuyerNIT                string
	BuyerName               string
	Subtotal                decimal.Decimal
	IVATotal                decimal.Decimal
	InvoiceTotal            decimal.Decimal
	SourceXMLPath           string
	SourcePDFPath           string
	// ReportedRetefuente/Reteiva/Reteica are the values the supplier already
	// reports in WithholdingTaxTotal. Informational only, used for
	// cross-validation against the engine's own calculation.
	ReportedRetefuente *decimal.Decimal
	ReportedReteiva    *decimal.Decimal
	ReportedReteica    *decimal.Decimal
	Lines              []InvoiceLine
}

type InvoiceLine struct {
	ID                       uint
	InvoiceID                uint
	LineNumber               int
	SKU                      *string
	Description              string
	Quantity                 decimal.Decimal
	UnitPrice                decimal.Decimal
	LineTotal                decimal.Decimal
	IVARate                  decimal.Decimal
	IVAValue                 decimal.Decimal
	ConceptID                *uint
	ClassificationConfidence *float64
}
