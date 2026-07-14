package entity

import (
	"time"

	"github.com/shopspring/decimal"

	"cifrato/internal/domain/enums"
)

type Concept struct {
	ID          uint
	Code        string
	Name        string
	Description string
}

type City struct {
	ID         uint
	Name       string
	Department string
}

type UVTValue struct {
	ID                  uint
	Year                int
	Value               decimal.Decimal
	EffectiveFrom       time.Time
	EffectiveTo         *time.Time
	ResolutionReference string
}

// TaxRule is one tariff for one concept, valid in a date range. CityID nil
// means a national rule; a value means a territorial ICA rule for that city.
type TaxRule struct {
	ID               uint
	TaxType          enums.TaxType
	ConceptID        uint
	CityID           *uint
	MinBaseUVT       decimal.Decimal
	TariffPercentage decimal.Decimal
	LegalBasis       string
	EffectiveFrom    time.Time
	EffectiveTo      *time.Time
}

// Calculation is the engine's output for one invoice, one classified
// concept, and one tax type. BaseAmount is aggregated across every line of
// that concept, since the minimum-base threshold is checked per invoice, not
// per line. ConceptID is nil for lines with no classified concept.
type Calculation struct {
	ID              uint
	InvoiceID       uint
	TaxType         enums.TaxType
	ConceptID       *uint
	ConceptName     *string
	BaseAmount      decimal.Decimal
	TariffApplied   decimal.Decimal
	CalculatedValue decimal.Decimal
	LegalBasis      string
	Justification   string
}

// CalculationSummary aggregates CalculatedValue across an invoice, one
// total per tax type.
type CalculationSummary struct {
	TotalRetefuente decimal.Decimal
	TotalReteiva    decimal.Decimal
	TotalReteica    decimal.Decimal
}
