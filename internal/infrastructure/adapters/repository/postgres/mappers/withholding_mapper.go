package mappers

import (
	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/enums"
	"cifrato/internal/infrastructure/adapters/repository/postgres/model"
)

func ConceptToDomain(m *model.WithholdingConceptModel) *entity.Concept {
	return &entity.Concept{
		ID:          m.ID,
		Code:        m.Code,
		Name:        m.Name,
		Description: m.Description,
	}
}

func ConceptToModel(c *entity.Concept) *model.WithholdingConceptModel {
	return &model.WithholdingConceptModel{
		ID:          c.ID,
		Code:        c.Code,
		Name:        c.Name,
		Description: c.Description,
	}
}

func TaxRuleToDomain(m *model.AdditionalTaxRuleModel) *entity.TaxRule {
	return &entity.TaxRule{
		ID:               m.ID,
		TaxType:          enums.TaxType(m.TaxType),
		ConceptID:        m.ConceptID,
		CityID:           m.CityID,
		MinBaseUVT:       m.MinBaseUVT,
		TariffPercentage: m.TariffPercentage,
		LegalBasis:       m.LegalBasis,
		EffectiveFrom:    m.EffectiveFrom,
		EffectiveTo:      m.EffectiveTo,
	}
}

func TaxRuleToModel(r *entity.TaxRule) *model.AdditionalTaxRuleModel {
	return &model.AdditionalTaxRuleModel{
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

func CalculationToDomain(m *model.WithholdingCalculationModel) *entity.Calculation {
	var conceptID *uint
	if m.ConceptID != 0 {
		id := m.ConceptID
		conceptID = &id
	}
	return &entity.Calculation{
		ID:              m.ID,
		InvoiceLineID:   m.InvoiceLineID,
		InvoiceID:       m.InvoiceID,
		TaxType:         enums.TaxType(m.TaxType),
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
func CalculationToModel(c *entity.Calculation) *model.WithholdingCalculationModel {
	var conceptID uint
	if c.ConceptID != nil {
		conceptID = *c.ConceptID
	}
	return &model.WithholdingCalculationModel{
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
