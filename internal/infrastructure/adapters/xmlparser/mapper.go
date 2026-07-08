package xmlparser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/enums"
)

func mapInvoice(xi *xmlInvoice, xmlType enums.XMLType) (*entity.Invoice, error) {
	cufe := strings.TrimSpace(xi.UUID)
	if cufe == "" {
		return nil, fmt.Errorf("missing required field CUFE (UUID)")
	}
	number := strings.TrimSpace(xi.ID)
	if number == "" {
		return nil, fmt.Errorf("missing required field InvoiceNumber (ID)")
	}

	issueDate, err := time.Parse("2006-01-02", strings.TrimSpace(xi.IssueDate))
	if err != nil {
		return nil, fmt.Errorf("parsing IssueDate %q: %w", xi.IssueDate, err)
	}

	issuerNIT := strings.TrimSpace(xi.AccountingSupplierParty.Party.PartyTaxScheme.CompanyID)
	if issuerNIT == "" {
		return nil, fmt.Errorf("missing required field IssuerNIT (AccountingSupplierParty/Party/PartyTaxScheme/CompanyID)")
	}
	buyerNIT := strings.TrimSpace(xi.AccountingCustomerParty.Party.PartyTaxScheme.CompanyID)
	if buyerNIT == "" {
		return nil, fmt.Errorf("missing required field BuyerNIT (AccountingCustomerParty/Party/PartyTaxScheme/CompanyID)")
	}

	invoiceTotal, err := parseDecimalRequired(xi.LegalMonetaryTotal.PayableAmount, "InvoiceTotal (LegalMonetaryTotal/PayableAmount)")
	if err != nil {
		return nil, err
	}

	subtotal := parseDecimalOptional(xi.LegalMonetaryTotal.LineExtensionAmount)

	issuerCity := strings.TrimSpace(xi.AccountingSupplierParty.Party.PhysicalLocation.Address.CityName)
	if issuerCity == "" {
		issuerCity = strings.TrimSpace(xi.AccountingSupplierParty.Party.PartyTaxScheme.RegistrationAddress.CityName)
	}

	ivaTotal := sumIVA(xi.TaxTotal)

	retefuente, reteiva, reteica := extractWithholding(xi.WithholdingTaxTotal)

	lines, err := mapLines(xi.InvoiceLine)
	if err != nil {
		return nil, err
	}

	return &entity.Invoice{
		CUFE:                    cufe,
		InvoiceNumber:           number,
		IssueDate:               issueDate,
		XMLType:                 xmlType,
		IssuerNIT:               issuerNIT,
		IssuerName:              strings.TrimSpace(xi.AccountingSupplierParty.Party.PartyTaxScheme.RegistrationName),
		IssuerCity:              issuerCity,
		IssuerTaxResponsibility: strings.TrimSpace(xi.AccountingSupplierParty.Party.PartyTaxScheme.TaxLevelCode),
		BuyerNIT:                buyerNIT,
		BuyerName:               strings.TrimSpace(xi.AccountingCustomerParty.Party.PartyTaxScheme.RegistrationName),
		Subtotal:                subtotal,
		IVATotal:                ivaTotal,
		InvoiceTotal:            invoiceTotal,
		ReportedRetefuente:      retefuente,
		ReportedReteiva:         reteiva,
		ReportedReteica:         reteica,
		Lines:                   lines,
	}, nil
}

// isIVAScheme matches TaxScheme/ID="01", falling back to a case-insensitive
// Name match otherwise.
func isIVAScheme(ts xmlTaxScheme) bool {
	if strings.TrimSpace(ts.ID) == "01" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(ts.Name), "IVA")
}

// ivaSubtotalKey identifies a TaxSubtotal for deduplication purposes.
type ivaSubtotalKey struct {
	schemeID   string
	schemeName string
	percent    string
	amount     string
}

// sumIVA sums IVA TaxSubtotal entries across all header TaxTotal blocks,
// deduping by (scheme ID, name, percent, amount) — some invoices repeat the
// same block verbatim.
func sumIVA(taxTotals []xmlTaxTotal) decimal.Decimal {
	seen := make(map[ivaSubtotalKey]bool)
	total := decimal.Zero
	for _, tt := range taxTotals {
		for _, ts := range tt.TaxSubtotal {
			if !isIVAScheme(ts.TaxCategory.TaxScheme) {
				continue
			}
			key := ivaSubtotalKey{
				schemeID:   strings.TrimSpace(ts.TaxCategory.TaxScheme.ID),
				schemeName: strings.TrimSpace(ts.TaxCategory.TaxScheme.Name),
				percent:    strings.TrimSpace(ts.TaxCategory.Percent),
				amount:     strings.TrimSpace(ts.TaxAmount),
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			total = total.Add(parseDecimalOptional(ts.TaxAmount))
		}
	}
	return total
}

// classifyWithholdingName maps a TaxScheme/Name to one of the three tracked
// withholding types via case-insensitive substring matching (TaxScheme/ID is
// not a reliable code for withholdings). Returns "" when nothing matches.
func classifyWithholdingName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(n, "retefuente"), strings.Contains(n, "reterenta"):
		return "retefuente"
	case strings.Contains(n, "reteiva"):
		return "reteiva"
	case strings.Contains(n, "reteica"):
		return "reteica"
	default:
		return ""
	}
}

// extractWithholding classifies each WithholdingTaxTotal block by its first
// matching TaxSubtotal Name, using the block-level TaxAmount. Unmatched
// blocks are ignored; if a category repeats, the last block wins.
func extractWithholding(blocks []xmlWithholdingTaxTotal) (retefuente, reteiva, reteica *decimal.Decimal) {
	for _, wht := range blocks {
		category := ""
		for _, ts := range wht.TaxSubtotal {
			if c := classifyWithholdingName(ts.TaxCategory.TaxScheme.Name); c != "" {
				category = c
				break
			}
		}
		if category == "" {
			continue
		}
		amount := parseDecimalOptional(wht.TaxAmount)
		switch category {
		case "retefuente":
			retefuente = &amount
		case "reteiva":
			reteiva = &amount
		case "reteica":
			reteica = &amount
		}
	}
	return retefuente, reteiva, reteica
}

// findIVASubtotal returns the rate and value of the first IVA TaxSubtotal
// across a line's TaxTotal blocks, or zero if none match.
func findIVASubtotal(taxTotals []xmlTaxTotal) (rate, value decimal.Decimal) {
	for _, tt := range taxTotals {
		for _, ts := range tt.TaxSubtotal {
			if isIVAScheme(ts.TaxCategory.TaxScheme) {
				return parseDecimalOptional(ts.TaxCategory.Percent), parseDecimalOptional(ts.TaxAmount)
			}
		}
	}
	return decimal.Zero, decimal.Zero
}

func mapLines(xlines []xmlInvoiceLine) ([]entity.InvoiceLine, error) {
	lines := make([]entity.InvoiceLine, 0, len(xlines))
	for _, xl := range xlines {
		lineNumber, err := strconv.Atoi(strings.TrimSpace(xl.ID))
		if err != nil {
			return nil, fmt.Errorf("parsing InvoiceLine ID %q as line number: %w", xl.ID, err)
		}

		var sku *string
		if raw := strings.TrimSpace(xl.Item.StandardItemIdentification.ID); raw != "" {
			sku = &raw
		}

		description := ""
		if len(xl.Item.Description) > 0 {
			description = strings.TrimSpace(xl.Item.Description[0])
		}

		ivaRate, ivaValue := findIVASubtotal(xl.TaxTotal)

		lines = append(lines, entity.InvoiceLine{
			LineNumber:  lineNumber,
			SKU:         sku,
			Description: description,
			Quantity:    parseDecimalOptional(xl.InvoicedQuantity),
			UnitPrice:   parseDecimalOptional(xl.Price.PriceAmount),
			LineTotal:   parseDecimalOptional(xl.LineExtensionAmount),
			IVARate:     ivaRate,
			IVAValue:    ivaValue,
		})
	}
	return lines, nil
}
