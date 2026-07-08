package repository

import (
	"context"
	"time"

	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/enums"
)

type (
	CalculationRepository interface {
		// Upsert overwrites the previous calculation for (InvoiceID, ConceptID, TaxType).
		Upsert(ctx context.Context, calc *entity.Calculation) error
		ListByInvoice(ctx context.Context, invoiceID uint) ([]entity.Calculation, error)
	}

	ClassificationCacheRepository interface {
		FindByIssuerAndSKU(ctx context.Context, issuerNIT, sku string) (*entity.ClassificationCacheEntry, error)
		FindByDescription(ctx context.Context, descriptionNormalized string) (*entity.ClassificationCacheEntry, error)
		Save(ctx context.Context, entry *entity.ClassificationCacheEntry) error
	}

	// InvoiceParser parses raw UBL DIAN invoice XML (direct Invoice or an
	// AttachedDocument wrapping one) into the domain model. Does not populate
	// SourceXMLPath/SourcePDFPath; the caller fills those in.
	InvoiceParser interface {
		Parse(xmlData []byte) (*entity.Invoice, error)
	}

	InvoiceRepository interface {
		// Save upserts by CUFE and replaces existing lines; re-importing the
		// same invoice does not duplicate lines.
		Save(ctx context.Context, inv *entity.Invoice) error
		FindByCUFE(ctx context.Context, cufe string) (*entity.Invoice, error)
		ExistsByCUFE(ctx context.Context, cufe string) (bool, error)
	}

	// LineClassifier classifies a line description into a withholding concept.
	// Must return a non-nil result or a non-nil error, never both nil.
	LineClassifier interface {
		Classify(ctx context.Context, description string) (*entity.LineClassification, error)
	}

	ReferenceDataRepository interface {
		FindConceptByCode(ctx context.Context, code string) (*entity.Concept, error)
		ListConcepts(ctx context.Context) ([]entity.Concept, error)
		FindCityByName(ctx context.Context, name string) (*entity.City, error)
		FindUVTValue(ctx context.Context, at time.Time) (*entity.UVTValue, error)
	}

	TaxRuleRepository interface {
		// FindApplicable returns the tax rule valid at the given date for the
		// concept. cityID nil means national rule; set means territorial ICA rule.
		FindApplicable(ctx context.Context, taxType enums.TaxType, conceptID uint, cityID *uint, at time.Time) (*entity.TaxRule, error)
		ListByTaxType(ctx context.Context, taxType enums.TaxType) ([]entity.TaxRule, error)
	}
)
