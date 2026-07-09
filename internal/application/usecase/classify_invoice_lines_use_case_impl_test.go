package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"cifrato/internal/application/usecase"
	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/repository/mocks"
)

// mockCacheMiss stubs both cache lookups to miss — the common setup for
// subtests where the point is what happens after the cache misses.
func mockCacheMiss(cache *mocks.MockClassificationCacheRepository) {
	cache.EXPECT().FindByIssuerAndSKU(mock.Anything, "900290912", "PTT199").Return(nil, nil)
	cache.EXPECT().FindByDescription(mock.Anything, "mantequilla perfumada 240 ml").Return(nil, nil)
}

func baseClassifyInvoice() *entity.Invoice {
	sku := "PTT199"
	return &entity.Invoice{
		CUFE:      "test-cufe",
		IssuerNIT: "900290912",
		Lines: []entity.InvoiceLine{
			{
				LineNumber:  1,
				SKU:         &sku,
				Description: "MANTEQUILLA PERFUMADA 240 ML",
			},
		},
	}
}

func TestClassifyInvoiceLines_Execute(t *testing.T) {
	t.Run("linea con hit por issuer_nit+sku no llama al LLM", func(t *testing.T) {
		cache := mocks.NewMockClassificationCacheRepository(t)
		cache.EXPECT().FindByIssuerAndSKU(mock.Anything, "900290912", "PTT199").
			Return(&entity.ClassificationCacheEntry{ConceptID: 1, Confidence: 0.95}, nil)
		classifier := mocks.NewMockLineClassifier(t)
		uc := usecase.NewClassifyInvoiceLines(cache, classifier)

		inv := baseClassifyInvoice()
		if err := uc.Execute(context.Background(), inv); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		classifier.AssertNotCalled(t, "Classify", mock.Anything, mock.Anything)
		if inv.Lines[0].ConceptID == nil || *inv.Lines[0].ConceptID != 1 {
			t.Errorf("ConceptID = %v, want 1", inv.Lines[0].ConceptID)
		}
	})

	t.Run("linea con hit por descripcion no llama al LLM", func(t *testing.T) {
		cache := mocks.NewMockClassificationCacheRepository(t)
		cache.EXPECT().FindByDescription(mock.Anything, "mantequilla perfumada 240 ml").
			Return(&entity.ClassificationCacheEntry{ConceptID: 1, Confidence: 0.9}, nil)
		classifier := mocks.NewMockLineClassifier(t)
		uc := usecase.NewClassifyInvoiceLines(cache, classifier)

		inv := baseClassifyInvoice()
		inv.Lines[0].SKU = nil // force the description path

		if err := uc.Execute(context.Background(), inv); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		classifier.AssertNotCalled(t, "Classify", mock.Anything, mock.Anything)
		if inv.Lines[0].ConceptID == nil || *inv.Lines[0].ConceptID != 1 {
			t.Errorf("ConceptID = %v, want 1", inv.Lines[0].ConceptID)
		}
	})

	t.Run("linea sin hit llama al LLM y guarda en cache", func(t *testing.T) {
		cache := mocks.NewMockClassificationCacheRepository(t)
		mockCacheMiss(cache)
		var saved *entity.ClassificationCacheEntry
		cache.EXPECT().Save(mock.Anything, mock.Anything).Run(func(_ context.Context, entry *entity.ClassificationCacheEntry) {
			saved = entry
		}).Return(nil)

		classifier := mocks.NewMockLineClassifier(t)
		// Exact (non-normalized) description: the classifier must receive the
		// raw text, not the cache's normalized key.
		classifier.EXPECT().Classify(mock.Anything, "MANTEQUILLA PERFUMADA 240 ML").Return(&entity.LineClassification{
			ConceptID: 1, ConceptCode: "compra_bienes", Confidence: 0.8, ModelVersion: "claude-haiku-4-5",
		}, nil).Once()

		uc := usecase.NewClassifyInvoiceLines(cache, classifier)

		inv := baseClassifyInvoice()
		if err := uc.Execute(context.Background(), inv); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if saved == nil {
			t.Fatal("expected a classification to be saved to cache")
		}
		if saved.DescriptionNormalized != "mantequilla perfumada 240 ml" {
			t.Errorf("DescriptionNormalized = %q, want normalized description", saved.DescriptionNormalized)
		}
		if inv.Lines[0].ConceptID == nil || *inv.Lines[0].ConceptID != 1 {
			t.Errorf("ConceptID = %v, want 1", inv.Lines[0].ConceptID)
		}
		if inv.Lines[0].ClassificationConfidence == nil || *inv.Lines[0].ClassificationConfidence != 0.8 {
			t.Errorf("ClassificationConfidence = %v, want 0.8", inv.Lines[0].ClassificationConfidence)
		}
	})

	t.Run("clasificacion nueva con SKU guarda la entrada con issuer_nit y sku", func(t *testing.T) {
		cache := mocks.NewMockClassificationCacheRepository(t)
		mockCacheMiss(cache)
		var saved *entity.ClassificationCacheEntry
		cache.EXPECT().Save(mock.Anything, mock.Anything).Run(func(_ context.Context, entry *entity.ClassificationCacheEntry) {
			saved = entry
		}).Return(nil)

		classifier := mocks.NewMockLineClassifier(t)
		classifier.EXPECT().Classify(mock.Anything, "MANTEQUILLA PERFUMADA 240 ML").Return(&entity.LineClassification{ConceptID: 1, Confidence: 0.8}, nil).Once()

		uc := usecase.NewClassifyInvoiceLines(cache, classifier)

		inv := baseClassifyInvoice()
		if err := uc.Execute(context.Background(), inv); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if saved == nil {
			t.Fatal("expected a classification to be saved to cache")
		}
		if saved.IssuerNIT == nil || *saved.IssuerNIT != "900290912" {
			t.Errorf("IssuerNIT = %v, want 900290912", saved.IssuerNIT)
		}
		if saved.SKU == nil || *saved.SKU != "PTT199" {
			t.Errorf("SKU = %v, want PTT199", saved.SKU)
		}
	})

	t.Run("error del LLM deja la linea sin concepto y no falla la factura", func(t *testing.T) {
		cache := mocks.NewMockClassificationCacheRepository(t)
		mockCacheMiss(cache)

		classifier := mocks.NewMockLineClassifier(t)
		classifier.EXPECT().Classify(mock.Anything, mock.Anything).Return(nil, errors.New("connection refused"))

		uc := usecase.NewClassifyInvoiceLines(cache, classifier)

		inv := baseClassifyInvoice()
		if err := uc.Execute(context.Background(), inv); err != nil {
			t.Fatalf("Execute() error = %v, want nil (LLM failure must not fail the invoice)", err)
		}
		if inv.Lines[0].ConceptID != nil {
			t.Errorf("ConceptID = %v, want nil", inv.Lines[0].ConceptID)
		}
		cache.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	})

	t.Run("error del repositorio de cache propaga error", func(t *testing.T) {
		cache := mocks.NewMockClassificationCacheRepository(t)
		cache.EXPECT().FindByIssuerAndSKU(mock.Anything, "900290912", "PTT199").Return(nil, errors.New("db connection lost"))
		classifier := mocks.NewMockLineClassifier(t)
		uc := usecase.NewClassifyInvoiceLines(cache, classifier)

		inv := baseClassifyInvoice()
		if err := uc.Execute(context.Background(), inv); err == nil {
			t.Fatal("expected an error when the cache repository fails")
		}
		classifier.AssertNotCalled(t, "Classify", mock.Anything, mock.Anything)
	})

	t.Run("multiples lineas: una falla y otra no se afectan entre si", func(t *testing.T) {
		cache := mocks.NewMockClassificationCacheRepository(t)
		cache.EXPECT().FindByIssuerAndSKU(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		cache.EXPECT().FindByDescription(mock.Anything, mock.Anything).Return(nil, nil)

		classifier := mocks.NewMockLineClassifier(t)
		classifier.EXPECT().Classify(mock.Anything, mock.Anything).Return(nil, errors.New("timeout"))

		uc := usecase.NewClassifyInvoiceLines(cache, classifier)

		inv := baseClassifyInvoice()
		sku2 := "OTHER123"
		inv.Lines = append(inv.Lines, entity.InvoiceLine{
			LineNumber:  2,
			SKU:         &sku2,
			Description: "SERVICIO DE ASESORIA CONTABLE",
		})

		if err := uc.Execute(context.Background(), inv); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		for i, line := range inv.Lines {
			if line.ConceptID != nil {
				t.Errorf("line %d: ConceptID = %v, want nil (classifier always fails in this test)", i, line.ConceptID)
			}
		}
		classifier.AssertNumberOfCalls(t, "Classify", 2)
	})
}
