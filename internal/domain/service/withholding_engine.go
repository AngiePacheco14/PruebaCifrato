package service

import (
	"fmt"

	"github.com/shopspring/decimal"

	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/enums"
)

// CalculateWithMinimumBase implements the RETEFUENTE/RETEICA pattern: the
// tariff applies only if baseAmount (pre-tax) is at or above rule.MinBaseUVT
// expressed in pesos for the given uvtValue. Equal to the minimum DOES
// apply (standard DIAN practice: "igual o superior"). Identity fields
// (ID/InvoiceLineID/InvoiceID/ConceptID) are left zero — the caller fills
// those in.
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

// CalculateReteiva implements the RETEIVA pattern: the tariff applies
// directly over ivaValue, with no UVT minimum (business decision — RETEIVA
// has none, unlike RETEFUENTE/RETEICA). Zero IVA on the line means nothing
// to withhold.
func CalculateReteiva(rule entity.TaxRule, ivaValue decimal.Decimal) entity.Calculation {
	iva := ivaValue.Round(2)

	if iva.IsZero() || iva.IsNegative() {
		return newCalculation(rule, iva, decimal.Zero, decimal.Zero, "no aplica: la línea no generó IVA")
	}

	value := iva.Mul(rule.TariffPercentage).Div(decimal.NewFromInt(100)).Round(2)
	justification := fmt.Sprintf("tarifa %s%% sobre IVA generado $%s", rule.TariffPercentage.String(), iva.StringFixed(2))
	return newCalculation(rule, iva, rule.TariffPercentage, value, justification)
}

// newCalculation builds a Calculation from a rule and its computed amounts —
// TaxType and LegalBasis always come from the rule, identical across every
// branch of CalculateWithMinimumBase/CalculateReteiva.
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

// NotApplicable builds a zero-value, auditable Calculation for cases where
// no TaxRule could even be looked up (missing concept classification,
// missing city tariff, buyer not a withholding agent, no rule configured
// for the date). There is no TaxRule to read TariffApplied/LegalBasis from,
// so both are left at their zero value — the reason lives entirely in
// Justification.
func NotApplicable(taxType enums.TaxType, reason string) entity.Calculation {
	return entity.Calculation{
		TaxType:         taxType,
		BaseAmount:      decimal.Zero,
		TariffApplied:   decimal.Zero,
		CalculatedValue: decimal.Zero,
		Justification:   reason,
	}
}
