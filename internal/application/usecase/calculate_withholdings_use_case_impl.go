package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	appconfig "cifrato/internal/application/config"
	"cifrato/internal/application/ports/in"
	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/enums"
	"cifrato/internal/domain/repository"
	"cifrato/internal/domain/service"
)

type CalculateWithholdings struct {
	taxRules      repository.TaxRuleRepository
	calculations  repository.CalculationRepository
	referenceData repository.ReferenceDataRepository
	cfg           appconfig.Config
}

func NewCalculateWithholdings(
	taxRules repository.TaxRuleRepository,
	calculations repository.CalculationRepository,
	referenceData repository.ReferenceDataRepository,
	cfg appconfig.Config,
) *CalculateWithholdings {
	return &CalculateWithholdings{taxRules: taxRules, calculations: calculations, referenceData: referenceData, cfg: cfg}
}

var _ in.CalculateWithholdings = (*CalculateWithholdings)(nil)

var allTaxTypes = []enums.TaxType{
	enums.TaxTypeRetefuente,
	enums.TaxTypeReteiva,
	enums.TaxTypeReteica,
}

// conceptGroup aggregates every line of an invoice that shares a concept.
// ConceptID nil is the "lines with no classified concept" bucket.
type conceptGroup struct {
	ConceptID *uint
	LineTotal decimal.Decimal
	IVAValue  decimal.Decimal
}

// groupLinesByConcept sums LineTotal and IVAValue per concept; nil-concept
// lines form their own group. RETEFUENTE/RETEICA's minimum-base check is
// per invoice, not per line.
func groupLinesByConcept(lines []entity.InvoiceLine) []conceptGroup {
	groups := make([]conceptGroup, 0, len(lines))

	for i := range lines {
		line := &lines[i]

		var group *conceptGroup
		for j := range groups {
			if sameConcept(groups[j].ConceptID, line.ConceptID) {
				group = &groups[j]
				break
			}
		}
		if group == nil {
			groups = append(groups, conceptGroup{ConceptID: line.ConceptID, LineTotal: decimal.Zero, IVAValue: decimal.Zero})
			group = &groups[len(groups)-1]
		}
		group.LineTotal = group.LineTotal.Add(line.LineTotal)
		group.IVAValue = group.IVAValue.Add(line.IVAValue)
	}

	return groups
}

func sameConcept(a, b *uint) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// Execute treats business/data-quality edge cases (unclassified lines,
// missing ICA tariff, non-agent buyer, base below minimum, no rule for the
// date) as zero-value Calculations, not errors. It only errors on
// infrastructure failures and the missing-UVT precondition.
func (uc *CalculateWithholdings) Execute(ctx context.Context, inv *entity.Invoice) ([]entity.Calculation, error) {
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

	// RETEICA keys off the issuer/seller city, resolved once per invoice.
	icaCity, err := uc.referenceData.FindCityByName(ctx, inv.IssuerCity)
	if err != nil {
		return nil, fmt.Errorf("usecase: finding city %q: %w", inv.IssuerCity, err)
	}

	conceptNames, err := uc.loadConceptNames(ctx)
	if err != nil {
		return nil, err
	}

	var results []entity.Calculation

	for _, group := range groupLinesByConcept(inv.Lines) {
		if group.ConceptID == nil {
			for _, tt := range allTaxTypes {
				calc := service.NotApplicable(tt, "línea(s) sin concepto clasificado, no se puede determinar la regla aplicable")
				if err := uc.persist(ctx, inv, nil, conceptNames, calc, &results); err != nil {
					return results, err
				}
			}
			continue
		}
		conceptID := *group.ConceptID

		refuenteCalc, err := uc.calculateMinimumBaseTax(ctx, enums.TaxTypeRetefuente, conceptID, nil, inv.IssueDate, uvt.Value, group.LineTotal,
			"sin regla RETEFUENTE configurada para este concepto vigente a la fecha de la factura")
		if err != nil {
			return results, err
		}
		if err := uc.persist(ctx, inv, group.ConceptID, conceptNames, refuenteCalc, &results); err != nil {
			return results, err
		}

		reteivaCalc, err := uc.calculateReteiva(ctx, conceptID, inv.IssueDate, group.IVAValue)
		if err != nil {
			return results, err
		}
		if err := uc.persist(ctx, inv, group.ConceptID, conceptNames, reteivaCalc, &results); err != nil {
			return results, err
		}

		reteicaCalc, err := uc.calculateReteica(ctx, conceptID, icaCity, inv.IssuerCity, inv.IssueDate, uvt.Value, group.LineTotal)
		if err != nil {
			return results, err
		}
		if err := uc.persist(ctx, inv, group.ConceptID, conceptNames, reteicaCalc, &results); err != nil {
			return results, err
		}
	}

	return results, nil
}

// loadConceptNames builds an ID→name lookup from the concept catalog, used
// to attach a human-readable ConceptName to each Calculation for display.
func (uc *CalculateWithholdings) loadConceptNames(ctx context.Context) (map[uint]string, error) {
	concepts, err := uc.referenceData.ListConcepts(ctx)
	if err != nil {
		return nil, fmt.Errorf("usecase: listing concepts: %w", err)
	}
	names := make(map[uint]string, len(concepts))
	for _, c := range concepts {
		names[c.ID] = c.Name
	}
	return names, nil
}

func (uc *CalculateWithholdings) calculateMinimumBaseTax(
	ctx context.Context,
	taxType enums.TaxType,
	conceptID uint,
	cityID *uint,
	at time.Time,
	uvtValue decimal.Decimal,
	baseAmount decimal.Decimal,
	notFoundReason string,
) (entity.Calculation, error) {
	rule, err := uc.taxRules.FindApplicable(ctx, taxType, conceptID, cityID, at)
	if err != nil {
		return entity.Calculation{}, fmt.Errorf("usecase: finding %s rule: %w", taxType, err)
	}
	if rule == nil {
		return service.NotApplicable(taxType, notFoundReason), nil
	}
	return service.CalculateWithMinimumBase(*rule, uvtValue, baseAmount), nil
}

func (uc *CalculateWithholdings) calculateReteiva(ctx context.Context, conceptID uint, at time.Time, ivaValue decimal.Decimal) (entity.Calculation, error) {
	if !uc.cfg.IsVATWithholdingAgent {
		return service.NotApplicable(enums.TaxTypeReteiva, "el comprador no es agente de retención de IVA"), nil
	}
	rule, err := uc.taxRules.FindApplicable(ctx, enums.TaxTypeReteiva, conceptID, nil, at)
	if err != nil {
		return entity.Calculation{}, fmt.Errorf("usecase: finding RETEIVA rule: %w", err)
	}
	if rule == nil {
		return service.NotApplicable(enums.TaxTypeReteiva, "sin regla RETEIVA configurada para este concepto vigente a la fecha de la factura"), nil
	}
	return service.CalculateReteiva(*rule, ivaValue), nil
}

func (uc *CalculateWithholdings) calculateReteica(ctx context.Context, conceptID uint, city *entity.City, issuerCityName string, at time.Time, uvtValue decimal.Decimal, baseAmount decimal.Decimal) (entity.Calculation, error) {
	if city == nil {
		return service.NotApplicable(enums.TaxTypeReteica, fmt.Sprintf("sin tarifa ICA configurada para la ciudad %s", issuerCityName)), nil
	}
	return uc.calculateMinimumBaseTax(ctx, enums.TaxTypeReteica, conceptID, &city.ID, at, uvtValue, baseAmount,
		fmt.Sprintf("sin tarifa RETEICA configurada para la ciudad %s y este concepto vigente a la fecha de la factura", issuerCityName))
}

func (uc *CalculateWithholdings) persist(ctx context.Context, inv *entity.Invoice, conceptID *uint, conceptNames map[uint]string, calc entity.Calculation, results *[]entity.Calculation) error {
	calc.InvoiceID = inv.ID
	calc.ConceptID = conceptID
	if conceptID != nil {
		if name, ok := conceptNames[*conceptID]; ok {
			calc.ConceptName = &name
		}
	}
	if err := uc.calculations.Upsert(ctx, &calc); err != nil {
		return fmt.Errorf("usecase: persisting %s calculation for invoice %d: %w", calc.TaxType, inv.ID, err)
	}
	*results = append(*results, calc)
	return nil
}
