package entity

// ClassificationCacheEntry is an LLM classification cache entry, keyed by
// (issuer_nit, sku) or by normalized description.
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

// LineClassification is the LLM's answer for one invoice line description.
// ConceptID is already resolved against the concept catalog; ConceptCode is
// for auditing/debugging only.
type LineClassification struct {
	ConceptID    uint
	ConceptCode  string
	Confidence   float64
	Reasoning    string
	ModelVersion string
}
