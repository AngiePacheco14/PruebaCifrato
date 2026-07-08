package in

import (
	"context"

	"cifrato/internal/domain/entity"
)

// ClassifyInvoiceLines populates ConceptID/ClassificationConfidence for
// every line of inv, using the classification cache first and falling
// back to the LLM. Never leaves the pipeline blocked — a single line's
// classification failure is logged and leaves that line unclassified
// rather than failing the whole invoice.
type ClassifyInvoiceLines interface {
	Execute(ctx context.Context, inv *entity.Invoice) error
}
