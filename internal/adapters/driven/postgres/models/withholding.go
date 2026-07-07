package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type WithholdingConceptModel struct {
	ID          uint   `gorm:"primaryKey"`
	Code        string `gorm:"size:50;uniqueIndex;not null"`
	Name        string `gorm:"size:150;not null"`
	Description string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (WithholdingConceptModel) TableName() string { return "withholding_concepts" }

// AdditionalTaxRuleModel unifies RETEFUENTE, RETEIVA and RETEICA: all three
// are structurally "a tariff percentage over a minimum base in UVT, valid in
// a date range, for a concept". ICA "per thousand" rates are normalized to
// their percentage equivalent at load time so TariffPercentage always means
// the same thing regardless of TaxType.
type AdditionalTaxRuleModel struct {
	ID               uint                    `gorm:"primaryKey"`
	TaxType          string                  `gorm:"size:20;not null;index:idx_rule_lookup,priority:1"`
	ConceptID        uint                    `gorm:"not null;index:idx_rule_lookup,priority:2"`
	Concept          WithholdingConceptModel `gorm:"foreignKey:ConceptID"`
	CityID           *uint                   `gorm:"index:idx_rule_lookup,priority:3"` // nil = national rule
	City             *CityModel              `gorm:"foreignKey:CityID"`
	MinBaseUVT       decimal.Decimal         `gorm:"type:numeric(10,2);not null;default:0"`
	TariffPercentage decimal.Decimal         `gorm:"type:numeric(7,4);not null"`
	LegalBasis       string                  `gorm:"type:text"`
	EffectiveFrom    time.Time               `gorm:"not null;index:idx_rule_lookup,priority:4"`
	EffectiveTo      *time.Time              `gorm:"index"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (AdditionalTaxRuleModel) TableName() string { return "additional_taxes_rules" }

// WithholdingCalculationModel stores only the latest calculation per
// (InvoiceLineID, TaxType) — recalculations overwrite, no history is kept.
type WithholdingCalculationModel struct {
	ID              uint            `gorm:"primaryKey"`
	InvoiceLineID   uint            `gorm:"not null;uniqueIndex:idx_line_taxtype,priority:1"`
	InvoiceID       uint            `gorm:"not null;index"`
	TaxType         string          `gorm:"size:20;not null;uniqueIndex:idx_line_taxtype,priority:2"`
	ConceptID       uint            `gorm:"not null;index"`
	BaseAmount      decimal.Decimal `gorm:"type:numeric(18,2);not null"`
	TariffApplied   decimal.Decimal `gorm:"type:numeric(7,4);not null"`
	CalculatedValue decimal.Decimal `gorm:"type:numeric(18,2);not null"`
	LegalBasis      string          `gorm:"type:text"`
	Justification   string          `gorm:"type:text"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (WithholdingCalculationModel) TableName() string { return "withholding_calculations" }
