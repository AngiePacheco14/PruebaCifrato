package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type InvoiceModel struct {
	ID                      uint               `gorm:"primaryKey"`
	CUFE                    string             `gorm:"size:100;uniqueIndex;not null"`
	InvoiceNumber           string             `gorm:"size:50;not null;index"`
	IssueDate               time.Time          `gorm:"not null;index"`
	XMLType                 string             `gorm:"size:20;not null"`
	IssuerNIT               string             `gorm:"size:20;not null;index"`
	IssuerName              string             `gorm:"size:255;not null"`
	IssuerCity              string             `gorm:"size:100"`
	IssuerTaxResponsibility string             `gorm:"size:255"`
	BuyerNIT                string             `gorm:"size:20;not null;index"`
	BuyerName               string             `gorm:"size:255;not null"`
	Subtotal                decimal.Decimal    `gorm:"type:numeric(18,2);not null"`
	IVATotal                decimal.Decimal    `gorm:"type:numeric(18,2);not null"`
	InvoiceTotal            decimal.Decimal    `gorm:"type:numeric(18,2);not null"`
	SourceXMLPath           string             `gorm:"size:500"`
	SourcePDFPath           string             `gorm:"size:500"`
	ReportedRetefuente      *decimal.Decimal   `gorm:"type:numeric(18,2)"`
	ReportedReteiva         *decimal.Decimal   `gorm:"type:numeric(18,2)"`
	ReportedReteica         *decimal.Decimal   `gorm:"type:numeric(18,2)"`
	Lines                   []InvoiceLineModel `gorm:"foreignKey:InvoiceID;constraint:OnDelete:CASCADE"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (InvoiceModel) TableName() string { return "invoices" }

type InvoiceLineModel struct {
	ID                       uint                     `gorm:"primaryKey"`
	InvoiceID                uint                     `gorm:"not null;index"`
	LineNumber               int                      `gorm:"not null"`
	SKU                      *string                  `gorm:"size:100;index"`
	Description              string                   `gorm:"type:text;not null"`
	Quantity                 decimal.Decimal          `gorm:"type:numeric(18,4);not null"`
	UnitPrice                decimal.Decimal          `gorm:"type:numeric(18,2);not null"`
	LineTotal                decimal.Decimal          `gorm:"type:numeric(18,2);not null"`
	IVARate                  decimal.Decimal          `gorm:"type:numeric(5,2);not null;default:0"`
	IVAValue                 decimal.Decimal          `gorm:"type:numeric(18,2);not null;default:0"`
	ConceptID                *uint                    `gorm:"index"`
	Concept                  *WithholdingConceptModel `gorm:"foreignKey:ConceptID"`
	ClassificationConfidence *float64
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func (InvoiceLineModel) TableName() string { return "invoice_lines" }
