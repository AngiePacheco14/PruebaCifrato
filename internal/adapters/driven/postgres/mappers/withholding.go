package mappers

import (
	"cifrato/internal/adapters/driven/postgres/models"
	"cifrato/internal/domain/withholding"
)

func ConceptToDomain(m *models.WithholdingConceptModel) *withholding.Concept {
	return &withholding.Concept{
		ID:          m.ID,
		Code:        m.Code,
		Name:        m.Name,
		Description: m.Description,
	}
}

func ConceptToModel(c *withholding.Concept) *models.WithholdingConceptModel {
	return &models.WithholdingConceptModel{
		ID:          c.ID,
		Code:        c.Code,
		Name:        c.Name,
		Description: c.Description,
	}
}

func TaxRuleToDomain(m *models.AdditionalTaxRuleModel) *withholding.TaxRule {
	return &withholding.TaxRule{
		ID:               m.ID,
		TaxType:          withholding.TaxType(m.TaxType),
		ConceptID:        m.ConceptID,
		CityID:           m.CityID,
		MinBaseUVT:       m.MinBaseUVT,
		TariffPercentage: m.TariffPercentage,
		LegalBasis:       m.LegalBasis,
		EffectiveFrom:    m.EffectiveFrom,
		EffectiveTo:      m.EffectiveTo,
	}
}

func TaxRuleToModel(r *withholding.TaxRule) *models.AdditionalTaxRuleModel {
	return &models.AdditionalTaxRuleModel{
		ID:               r.ID,
		TaxType:          string(r.TaxType),
		ConceptID:        r.ConceptID,
		CityID:           r.CityID,
		MinBaseUVT:       r.MinBaseUVT,
		TariffPercentage: r.TariffPercentage,
		LegalBasis:       r.LegalBasis,
		EffectiveFrom:    r.EffectiveFrom,
		EffectiveTo:      r.EffectiveTo,
	}
}

func CalculationToDomain(m *models.WithholdingCalculationModel) *withholding.Calculation {
	var conceptID *uint
	if m.ConceptID != 0 {
		id := m.ConceptID
		conceptID = &id
	}
	return &withholding.Calculation{
		ID:              m.ID,
		InvoiceLineID:   m.InvoiceLineID,
		InvoiceID:       m.InvoiceID,
		TaxType:         withholding.TaxType(m.TaxType),
		ConceptID:       conceptID,
		BaseAmount:      m.BaseAmount,
		TariffApplied:   m.TariffApplied,
		CalculatedValue: m.CalculatedValue,
		LegalBasis:      m.LegalBasis,
		Justification:   m.Justification,
	}
}

// CalculationToModel represents an unclassified line's Calculation.ConceptID
// (nil) as 0 in the column: withholding_calculations.concept_id is NOT NULL
// with no FK constraint, so 0 (never a real auto-generated ID) is safe as
// the "no concept" sentinel — the domain itself only ever deals in nil.
func CalculationToModel(c *withholding.Calculation) *models.WithholdingCalculationModel {
	var conceptID uint
	if c.ConceptID != nil {
		conceptID = *c.ConceptID
	}
	return &models.WithholdingCalculationModel{
		ID:              c.ID,
		InvoiceLineID:   c.InvoiceLineID,
		InvoiceID:       c.InvoiceID,
		TaxType:         string(c.TaxType),
		ConceptID:       conceptID,
		BaseAmount:      c.BaseAmount,
		TariffApplied:   c.TariffApplied,
		CalculatedValue: c.CalculatedValue,
		LegalBasis:      c.LegalBasis,
		Justification:   c.Justification,
	}
}
