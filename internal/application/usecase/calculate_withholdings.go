package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	appconfig "cifrato/internal/application/config"
	"cifrato/internal/application/ports/in"
	"cifrato/internal/application/ports/out"
	"cifrato/internal/domain/invoice"
	"cifrato/internal/domain/withholding"
)

type CalculateWithholdings struct {
	taxRules      out.TaxRuleRepository
	calculations  out.CalculationRepository
	referenceData out.ReferenceDataRepository
	cfg           appconfig.Config
}

func NewCalculateWithholdings(
	taxRules out.TaxRuleRepository,
	calculations out.CalculationRepository,
	referenceData out.ReferenceDataRepository,
	cfg appconfig.Config,
) *CalculateWithholdings {
	return &CalculateWithholdings{taxRules: taxRules, calculations: calculations, referenceData: referenceData, cfg: cfg}
}

var _ in.CalculateWithholdings = (*CalculateWithholdings)(nil)

var allTaxTypes = []withholding.TaxType{
	withholding.TaxTypeRetefuente,
	withholding.TaxTypeReteiva,
	withholding.TaxTypeReteica,
}

// Execute never fails because of a per-line business/data-quality edge case
// (unclassified line, missing ICA city tariff, non-agent buyer, base below
// minimum, no rule configured for the date) — each of those always produces
// an auditable zero-value Calculation, persisted like any other. It DOES
// return an error on genuine infrastructure failures (repository errors)
// and on the missing-UVT precondition, since that means the environment is
// misconfigured, not that this particular invoice/line has a business
// reason to skip withholding.
func (uc *CalculateWithholdings) Execute(ctx context.Context, inv *invoice.Invoice) ([]withholding.Calculation, error) {
	if inv.ID == 0 {
		return nil, fmt.Errorf("usecase: invoice must be persisted (ID=0) before calculating withholdings")
	}

	uvt, err := uc.referenceData.FindUVTValue(ctx, inv.IssueDate)
	if err != nil {
		return nil, fmt.Errorf("usecase: finding UVT value for %s: %w", inv.IssueDate.Format("2006-01-02"), err)
	}
	if uvt == nil {
		return nil, fmt.Errorf("usecase: no UVT value configured for %s", inv.IssueDate.Format("2006-01-02"))
	}

	// Resolved once per invoice, not per line: RETEICA always keys off the
	// issuer/seller city (business decision — RETEICA looks at where the
	// seller operates, not the buyer), and IssuerCity is invoice-level.
	icaCity, err := uc.referenceData.FindCityByName(ctx, inv.IssuerCity)
	if err != nil {
		return nil, fmt.Errorf("usecase: finding city %q: %w", inv.IssuerCity, err)
	}

	var results []withholding.Calculation

	for i := range inv.Lines {
		line := &inv.Lines[i]

		if line.ConceptID == nil {
			for _, tt := range allTaxTypes {
				calc := withholding.NotApplicable(tt, "línea sin concepto clasificado, no se puede determinar la regla aplicable")
				if err := uc.persist(ctx, inv, line, nil, calc, &results); err != nil {
					return results, err
				}
			}
			continue
		}
		conceptID := *line.ConceptID

		refuenteCalc, err := uc.calculateMinimumBaseTax(ctx, withholding.TaxTypeRetefuente, conceptID, nil, inv.IssueDate, uvt.Value, line.LineTotal,
			"sin regla RETEFUENTE configurada para este concepto vigente a la fecha de la factura")
		if err != nil {
			return results, err
		}
		if err := uc.persist(ctx, inv, line, line.ConceptID, refuenteCalc, &results); err != nil {
			return results, err
		}

		reteivaCalc, err := uc.calculateReteiva(ctx, conceptID, inv.IssueDate, line.IVAValue)
		if err != nil {
			return results, err
		}
		if err := uc.persist(ctx, inv, line, line.ConceptID, reteivaCalc, &results); err != nil {
			return results, err
		}

		reteicaCalc, err := uc.calculateReteica(ctx, conceptID, icaCity, inv.IssuerCity, inv.IssueDate, uvt.Value, line.LineTotal)
		if err != nil {
			return results, err
		}
		if err := uc.persist(ctx, inv, line, line.ConceptID, reteicaCalc, &results); err != nil {
			return results, err
		}
	}

	return results, nil
}

func (uc *CalculateWithholdings) calculateMinimumBaseTax(
	ctx context.Context,
	taxType withholding.TaxType,
	conceptID uint,
	cityID *uint,
	at time.Time,
	uvtValue decimal.Decimal,
	baseAmount decimal.Decimal,
	notFoundReason string,
) (withholding.Calculation, error) {
	rule, err := uc.taxRules.FindApplicable(ctx, taxType, conceptID, cityID, at)
	if err != nil {
		return withholding.Calculation{}, fmt.Errorf("usecase: finding %s rule: %w", taxType, err)
	}
	if rule == nil {
		return withholding.NotApplicable(taxType, notFoundReason), nil
	}
	return withholding.CalculateWithMinimumBase(*rule, uvtValue, baseAmount), nil
}

func (uc *CalculateWithholdings) calculateReteiva(ctx context.Context, conceptID uint, at time.Time, ivaValue decimal.Decimal) (withholding.Calculation, error) {
	if !uc.cfg.IsVATWithholdingAgent {
		return withholding.NotApplicable(withholding.TaxTypeReteiva, "el comprador no es agente de retención de IVA"), nil
	}
	rule, err := uc.taxRules.FindApplicable(ctx, withholding.TaxTypeReteiva, conceptID, nil, at)
	if err != nil {
		return withholding.Calculation{}, fmt.Errorf("usecase: finding RETEIVA rule: %w", err)
	}
	if rule == nil {
		return withholding.NotApplicable(withholding.TaxTypeReteiva, "sin regla RETEIVA configurada para este concepto vigente a la fecha de la factura"), nil
	}
	return withholding.CalculateReteiva(*rule, ivaValue), nil
}

func (uc *CalculateWithholdings) calculateReteica(ctx context.Context, conceptID uint, city *withholding.City, issuerCityName string, at time.Time, uvtValue decimal.Decimal, baseAmount decimal.Decimal) (withholding.Calculation, error) {
	if city == nil {
		return withholding.NotApplicable(withholding.TaxTypeReteica, fmt.Sprintf("sin tarifa ICA configurada para la ciudad %s", issuerCityName)), nil
	}
	return uc.calculateMinimumBaseTax(ctx, withholding.TaxTypeReteica, conceptID, &city.ID, at, uvtValue, baseAmount,
		fmt.Sprintf("sin tarifa RETEICA configurada para la ciudad %s y este concepto vigente a la fecha de la factura", issuerCityName))
}

func (uc *CalculateWithholdings) persist(ctx context.Context, inv *invoice.Invoice, line *invoice.InvoiceLine, conceptID *uint, calc withholding.Calculation, results *[]withholding.Calculation) error {
	calc.InvoiceLineID = line.ID
	calc.InvoiceID = inv.ID
	calc.ConceptID = conceptID
	if err := uc.calculations.Upsert(ctx, &calc); err != nil {
		return fmt.Errorf("usecase: persisting %s calculation for line %d: %w", calc.TaxType, line.ID, err)
	}
	*results = append(*results, calc)
	return nil
}
