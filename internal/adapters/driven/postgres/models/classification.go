package models

import "time"

// LineClassificationModel is a self-feeding LLM classification cache, not a
// hand-curated keyword catalog. Lookup priority: (IssuerNIT, SKU) first
// (stronger identity, same item from the same supplier), falling back to
// DescriptionNormalized when no SKU is available. The composite uniqueIndex
// on (issuer_nit, sku) naturally excludes rows where either is NULL, per
// standard Postgres unique-index semantics — no partial index needed.
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
