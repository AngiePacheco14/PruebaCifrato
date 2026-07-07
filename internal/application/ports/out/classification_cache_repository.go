package out

import "context"

// ClassificationCacheEntry is an infrastructure DTO, not a domain entity: it
// represents the LLM classification cache, a detail of the LLM+postgres
// adapter that the domain never needs to know about.
type ClassificationCacheEntry struct {
	ID                    uint
	IssuerNIT             *string
	SKU                   *string
	DescriptionNormalized string
	ConceptID             uint
	Confidence            float64
	ModelVersion          string
	Reasoning             string
}

type ClassificationCacheRepository interface {
	FindByIssuerAndSKU(ctx context.Context, issuerNIT, sku string) (*ClassificationCacheEntry, error)
	FindByDescription(ctx context.Context, descriptionNormalized string) (*ClassificationCacheEntry, error)
	Save(ctx context.Context, entry *ClassificationCacheEntry) error
}
