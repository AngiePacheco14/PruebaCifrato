package entity

// ClassificationCacheEntry represents the LLM classification cache — a
// self-feeding cache keyed by (issuer_nit, sku) or by normalized
// description, so the same line description is never sent to the LLM
// twice.
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
// ConceptID is already resolved against the in-memory concept catalog by
// the adapter — callers never need to know about ConceptCode beyond
// auditing/debugging.
type LineClassification struct {
	ConceptID    uint
	ConceptCode  string
	Confidence   float64
	Reasoning    string
	ModelVersion string
}
