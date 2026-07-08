package service

import (
	"fmt"

	"github.com/shopspring/decimal"

	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/enums"
)

// CalculateWithMinimumBase applies the RETEFUENTE/RETEICA tariff only if
// baseAmount is at or above rule.MinBaseUVT in pesos; equal to the minimum
// does apply. Identity fields (ID/InvoiceID/ConceptID) are left zero.
func CalculateWithMinimumBase(rule entity.TaxRule, uvtValue decimal.Decimal, baseAmount decimal.Decimal) entity.Calculation {
	minPesos := rule.MinBaseUVT.Mul(uvtValue).Round(2)
	base := baseAmount.Round(2)

	if base.LessThan(minPesos) {
		justification := fmt.Sprintf(
			"no supera la base mínima de %s UVT ($%s): base gravable $%s",
			rule.MinBaseUVT.String(), minPesos.StringFixed(2), base.StringFixed(2),
		)
		return newCalculation(rule, base, decimal.Zero, decimal.Zero, justification)
	}

	value := base.Mul(rule.TariffPercentage).Div(decimal.NewFromInt(100)).Round(2)
	justification := fmt.Sprintf(
		"base gravable $%s supera/iguala el mínimo de %s UVT ($%s); se aplica tarifa %s%%",
		base.StringFixed(2), rule.MinBaseUVT.String(), minPesos.StringFixed(2), rule.TariffPercentage.String(),
	)
	return newCalculation(rule, base, rule.TariffPercentage, value, justification)
}

// CalculateReteiva applies the tariff directly over ivaValue, with no UVT
// minimum. Zero or negative IVA means nothing to withhold.
func CalculateReteiva(rule entity.TaxRule, ivaValue decimal.Decimal) entity.Calculation {
	iva := ivaValue.Round(2)

	if iva.IsZero() || iva.IsNegative() {
		return newCalculation(rule, iva, decimal.Zero, decimal.Zero, "no aplica: la línea no generó IVA")
	}

	value := iva.Mul(rule.TariffPercentage).Div(decimal.NewFromInt(100)).Round(2)
	justification := fmt.Sprintf("tarifa %s%% sobre IVA generado $%s", rule.TariffPercentage.String(), iva.StringFixed(2))
	return newCalculation(rule, iva, rule.TariffPercentage, value, justification)
}

// newCalculation builds a Calculation from a rule and its computed amounts.
func newCalculation(rule entity.TaxRule, base, tariff, value decimal.Decimal, justification string) entity.Calculation {
	return entity.Calculation{
		TaxType:         rule.TaxType,
		BaseAmount:      base,
		TariffApplied:   tariff,
		CalculatedValue: value,
		LegalBasis:      rule.LegalBasis,
		Justification:   justification,
	}
}

// SummarizeByTaxType rolls up CalculatedValue across every line of an
// invoice into one total per tax type.
func SummarizeByTaxType(calculations []entity.Calculation) entity.CalculationSummary {
	summary := entity.CalculationSummary{
		TotalRetefuente: decimal.Zero,
		TotalReteiva:    decimal.Zero,
		TotalReteica:    decimal.Zero,
	}
	for _, c := range calculations {
		switch c.TaxType {
		case enums.TaxTypeRetefuente:
			summary.TotalRetefuente = summary.TotalRetefuente.Add(c.CalculatedValue)
		case enums.TaxTypeReteiva:
			summary.TotalReteiva = summary.TotalReteiva.Add(c.CalculatedValue)
		case enums.TaxTypeReteica:
			summary.TotalReteica = summary.TotalReteica.Add(c.CalculatedValue)
		}
	}
	return summary
}

// NotApplicable builds a zero-value Calculation for when no TaxRule could
// be looked up; the reason lives entirely in Justification.
func NotApplicable(taxType enums.TaxType, reason string) entity.Calculation {
	return entity.Calculation{
		TaxType:         taxType,
		BaseAmount:      decimal.Zero,
		TariffApplied:   decimal.Zero,
		CalculatedValue: decimal.Zero,
		Justification:   reason,
	}
}
