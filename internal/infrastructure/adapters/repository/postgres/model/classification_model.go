package model

import "time"

// LineClassificationModel is an LLM classification cache. Lookup priority:
// (IssuerNIT, SKU) first, falling back to DescriptionNormalized. The unique
// index on (issuer_nit, sku) naturally excludes NULL rows in Postgres.
type LineClassificationModel struct {
	ID                    uint                    `gorm:"primaryKey"`
	IssuerNIT             *string                 `gorm:"size:20;uniqueIndex:idx_issuer_sku"`
	SKU                   *string                 `gorm:"size:100;uniqueIndex:idx_issuer_sku"`
	DescriptionNormalized string                  `gorm:"size:500;index"`
	ConceptID             uint                    `gorm:"not null;index"`
	Concept               WithholdingConceptModel `gorm:"foreignKey:ConceptID"`
	Confidence            float64                 `gorm:"not null"`
	ModelVersion          string                  `gorm:"size:50;not null"`
	Reasoning             string                  `gorm:"type:text"`
	CreatedAt             time.Time
}

func (LineClassificationModel) TableName() string { return "line_classifications" }
