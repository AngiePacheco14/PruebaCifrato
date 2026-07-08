package usecase

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"cifrato/internal/application/ports/in"
	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/repository"
)

type ClassifyInvoiceLines struct {
	cache      repository.ClassificationCacheRepository
	classifier repository.LineClassifier
}

func NewClassifyInvoiceLines(cache repository.ClassificationCacheRepository, classifier repository.LineClassifier) *ClassifyInvoiceLines {
	return &ClassifyInvoiceLines{cache: cache, classifier: classifier}
}

var _ in.ClassifyInvoiceLines = (*ClassifyInvoiceLines)(nil)

var multiSpace = regexp.MustCompile(`\s+`)

// normalizeDescription builds the cache key: lowercase, trimmed, collapsed
// whitespace. No accent-stripping — this is a cache key, not a semantic match.
func normalizeDescription(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return multiSpace.ReplaceAllString(s, " ")
}

// Execute leaves a line unclassified (nil ConceptID) and logs it if the LLM
// call fails, rather than failing the whole invoice. It only returns an
// error on cache read/write failures.
func (uc *ClassifyInvoiceLines) Execute(ctx context.Context, inv *entity.Invoice) error {
	for i := range inv.Lines {
		line := &inv.Lines[i]
		normalized := normalizeDescription(line.Description)

		entry, err := uc.lookupCache(ctx, inv.IssuerNIT, line, normalized)
		if err != nil {
			return fmt.Errorf("usecase: looking up classification cache for line %d: %w", line.LineNumber, err)
		}

		if entry != nil {
			assignClassification(line, entry.ConceptID, entry.Confidence)
			continue
		}

		result, err := uc.classifier.Classify(ctx, line.Description)
		if err != nil {
			log.Printf("classify_invoice_lines: LLM classification failed for invoice %s line %d (%q): %v — leaving unclassified", inv.CUFE, line.LineNumber, line.Description, err)
			continue
		}

		newEntry := &entity.ClassificationCacheEntry{
			DescriptionNormalized: normalized,
			ConceptID:             result.ConceptID,
			Confidence:            result.Confidence,
			ModelVersion:          result.ModelVersion,
			Reasoning:             result.Reasoning,
		}
		if line.SKU != nil && *line.SKU != "" {
			issuerNIT := inv.IssuerNIT
			newEntry.IssuerNIT = &issuerNIT
			newEntry.SKU = line.SKU
		}
		if err := uc.cache.Save(ctx, newEntry); err != nil {
			return fmt.Errorf("usecase: saving classification cache entry for line %d: %w", line.LineNumber, err)
		}

		assignClassification(line, result.ConceptID, result.Confidence)
	}
	return nil
}

func (uc *ClassifyInvoiceLines) lookupCache(ctx context.Context, issuerNIT string, line *entity.InvoiceLine, normalizedDescription string) (*entity.ClassificationCacheEntry, error) {
	if line.SKU != nil && *line.SKU != "" {
		entry, err := uc.cache.FindByIssuerAndSKU(ctx, issuerNIT, *line.SKU)
		if err != nil {
			return nil, err
		}
		if entry != nil {
			return entry, nil
		}
	}
	return uc.cache.FindByDescription(ctx, normalizedDescription)
}

// assignClassification sets ConceptID/ClassificationConfidence on line —
// both the cache-hit path and the fresh-LLM-result path end the same way.
func assignClassification(line *entity.InvoiceLine, conceptID uint, confidence float64) {
	line.ConceptID = &conceptID
	line.ClassificationConfidence = &confidence
}
