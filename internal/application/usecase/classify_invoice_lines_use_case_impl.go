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

// normalizeDescription builds the cache key for DescriptionNormalized: a
// simple, deterministic transform (lowercase, trimmed, collapsed internal
// whitespace) — not NLP. No accent-stripping: it's a cache key, not a
// semantic match, and unicode normalization would need a real
// transliteration table to do correctly.
func normalizeDescription(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return multiSpace.ReplaceAllString(s, " ")
}

// Execute never fails the whole invoice because a single line's LLM call
// failed (network error, exhausted retries, unexpected response) — that
// line is simply left with ConceptID/ClassificationConfidence nil, which
// the withholding engine already handles gracefully ("no aplica: línea sin
// concepto clasificado"), the same path as a never-classified line. This
// keeps the pipeline non-blocking (no human review queue). The failure is
// logged for observability, since repeated failures across many lines may
// indicate a systemic problem (e.g. an invalid API key) worth
// investigating even though it doesn't halt the batch.
//
// Execute DOES return an error when the classification cache itself fails
// to read or write — that is an infrastructure failure analogous to how
// calculate_withholdings_use_case_impl.go treats
// TaxRuleRepository/CalculationRepository errors, not a per-line business
// edge case.
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
