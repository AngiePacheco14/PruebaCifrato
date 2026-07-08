package model

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

// AdditionalTaxRuleModel unifies RETEFUENTE, RETEIVA and RETEICA as a tariff
// over a minimum base (UVT) for a concept and date range. ICA's "per thousand"
// rates are normalized to a percentage at load time.
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
// (InvoiceID, ConceptID, TaxType); recalculations overwrite. BaseAmount is
// aggregated across all lines of that concept, not a single line.
type WithholdingCalculationModel struct {
	ID              uint            `gorm:"primaryKey"`
	InvoiceID       uint            `gorm:"not null;uniqueIndex:idx_invoice_concept_taxtype,priority:1"`
	TaxType         string          `gorm:"size:20;not null;uniqueIndex:idx_invoice_concept_taxtype,priority:3"`
	ConceptID       uint            `gorm:"not null;uniqueIndex:idx_invoice_concept_taxtype,priority:2;index"`
	BaseAmount      decimal.Decimal `gorm:"type:numeric(18,2);not null"`
	TariffApplied   decimal.Decimal `gorm:"type:numeric(7,4);not null"`
	CalculatedValue decimal.Decimal `gorm:"type:numeric(18,2);not null"`
	LegalBasis      string          `gorm:"type:text"`
	Justification   string          `gorm:"type:text"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (WithholdingCalculationModel) TableName() string { return "withholding_calculations" }
