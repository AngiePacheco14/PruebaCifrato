package in

import (
	"context"

	"cifrato/internal/domain/entity"
)

// ClassifyInvoiceLines populates ConceptID/ClassificationConfidence for every
// line of inv, checking the classification cache before falling back to the
// LLM. A single line's classification failure is logged, not fatal.
type ClassifyInvoiceLines interface {
	Execute(ctx context.Context, inv *entity.Invoice) error
}
