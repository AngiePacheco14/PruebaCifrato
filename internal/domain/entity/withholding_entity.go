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

// TaxRule is a single row of the additional_taxes_rules table: one tariff
// for one concept, valid in a date range. CityID nil means a national rule
// (RETEFUENTE/RETEIVA); a value means a territorial ICA rule for that city.
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

// Calculation is the engine's output for one invoice line and one tax type.
// ConceptID is nil when the line had no classified concept — same shape as
// InvoiceLine.ConceptID. How a persistence adapter represents "no concept"
// in its own schema (NULL, a sentinel, etc.) is that adapter's concern, not
// the domain's.
type Calculation struct {
	ID              uint
	InvoiceLineID   uint
	InvoiceID       uint
	TaxType         enums.TaxType
	ConceptID       *uint
	BaseAmount      decimal.Decimal
	TariffApplied   decimal.Decimal
	CalculatedValue decimal.Decimal
	LegalBasis      string
	Justification   string
}
