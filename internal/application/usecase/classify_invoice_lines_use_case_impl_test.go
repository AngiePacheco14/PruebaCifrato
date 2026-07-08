package usecase_test

import (
	"context"
	"errors"
	"testing"

	"cifrato/internal/application/usecase"
	"cifrato/internal/domain/entity"
)

type fakeLineClassifier struct {
	calls  int
	result *entity.LineClassification
	err    error
}

func (f *fakeLineClassifier) Classify(context.Context, string) (*entity.LineClassification, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

type fakeClassificationCache struct {
	byIssuerSKU map[string]entity.ClassificationCacheEntry
	byDesc      map[string]entity.ClassificationCacheEntry
	saved       []entity.ClassificationCacheEntry
	findErr     error
	saveErr     error
}

func newFakeCache() *fakeClassificationCache {
	return &fakeClassificationCache{
		byIssuerSKU: map[string]entity.ClassificationCacheEntry{},
		byDesc:      map[string]entity.ClassificationCacheEntry{},
	}
}

func (f *fakeClassificationCache) FindByIssuerAndSKU(_ context.Context, issuerNIT, sku string) (*entity.ClassificationCacheEntry, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	e, ok := f.byIssuerSKU[issuerNIT+"|"+sku]
	if !ok {
		return nil, nil
	}
	ec := e
	return &ec, nil
}

func (f *fakeClassificationCache) FindByDescription(_ context.Context, desc string) (*entity.ClassificationCacheEntry, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	e, ok := f.byDesc[desc]
	if !ok {
		return nil, nil
	}
	ec := e
	return &ec, nil
}

func (f *fakeClassificationCache) Save(_ context.Context, entry *entity.ClassificationCacheEntry) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	entry.ID = uint(len(f.saved) + 1)
	f.saved = append(f.saved, *entry)
	return nil
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
		cache := newFakeCache()
		cache.byIssuerSKU["900290912|PTT199"] = entity.ClassificationCacheEntry{ConceptID: 1, Confidence: 0.95}
		classifier := &fakeLineClassifier{}
		uc := usecase.NewClassifyInvoiceLines(cache, classifier)

		inv := baseClassifyInvoice()
		if err := uc.Execute(context.Background(), inv); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if classifier.calls != 0 {
			t.Errorf("classifier.calls = %d, want 0", classifier.calls)
		}
		if inv.Lines[0].ConceptID == nil || *inv.Lines[0].ConceptID != 1 {
			t.Errorf("ConceptID = %v, want 1", inv.Lines[0].ConceptID)
		}
	})

	t.Run("linea con hit por descripcion no llama al LLM", func(t *testing.T) {
		cache := newFakeCache()
		cache.byDesc["mantequilla perfumada 240 ml"] = entity.ClassificationCacheEntry{ConceptID: 1, Confidence: 0.9}
		classifier := &fakeLineClassifier{}
		uc := usecase.NewClassifyInvoiceLines(cache, classifier)

		inv := baseClassifyInvoice()
		inv.Lines[0].SKU = nil // force the description path

		if err := uc.Execute(context.Background(), inv); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if classifier.calls != 0 {
			t.Errorf("classifier.calls = %d, want 0", classifier.calls)
		}
		if inv.Lines[0].ConceptID == nil || *inv.Lines[0].ConceptID != 1 {
			t.Errorf("ConceptID = %v, want 1", inv.Lines[0].ConceptID)
		}
	})

	t.Run("linea sin hit llama al LLM y guarda en cache", func(t *testing.T) {
		cache := newFakeCache()
		classifier := &fakeLineClassifier{result: &entity.LineClassification{
			ConceptID: 1, ConceptCode: "compra_bienes", Confidence: 0.8, ModelVersion: "claude-haiku-4-5",
		}}
		uc := usecase.NewClassifyInvoiceLines(cache, classifier)

		inv := baseClassifyInvoice()
		if err := uc.Execute(context.Background(), inv); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if classifier.calls != 1 {
			t.Errorf("classifier.calls = %d, want 1", classifier.calls)
		}
		if len(cache.saved) != 1 {
			t.Fatalf("len(cache.saved) = %d, want 1", len(cache.saved))
		}
		if cache.saved[0].DescriptionNormalized != "mantequilla perfumada 240 ml" {
			t.Errorf("DescriptionNormalized = %q, want normalized description", cache.saved[0].DescriptionNormalized)
		}
		if inv.Lines[0].ConceptID == nil || *inv.Lines[0].ConceptID != 1 {
			t.Errorf("ConceptID = %v, want 1", inv.Lines[0].ConceptID)
		}
		if inv.Lines[0].ClassificationConfidence == nil || *inv.Lines[0].ClassificationConfidence != 0.8 {
			t.Errorf("ClassificationConfidence = %v, want 0.8", inv.Lines[0].ClassificationConfidence)
		}
	})

	t.Run("clasificacion nueva con SKU guarda la entrada con issuer_nit y sku", func(t *testing.T) {
		cache := newFakeCache()
		classifier := &fakeLineClassifier{result: &entity.LineClassification{ConceptID: 1, Confidence: 0.8}}
		uc := usecase.NewClassifyInvoiceLines(cache, classifier)

		inv := baseClassifyInvoice()
		if err := uc.Execute(context.Background(), inv); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(cache.saved) != 1 {
			t.Fatalf("len(cache.saved) = %d, want 1", len(cache.saved))
		}
		saved := cache.saved[0]
		if saved.IssuerNIT == nil || *saved.IssuerNIT != "900290912" {
			t.Errorf("IssuerNIT = %v, want 900290912", saved.IssuerNIT)
		}
		if saved.SKU == nil || *saved.SKU != "PTT199" {
			t.Errorf("SKU = %v, want PTT199", saved.SKU)
		}
	})

	t.Run("error del LLM deja la linea sin concepto y no falla la factura", func(t *testing.T) {
		cache := newFakeCache()
		classifier := &fakeLineClassifier{err: errors.New("connection refused")}
		uc := usecase.NewClassifyInvoiceLines(cache, classifier)

		inv := baseClassifyInvoice()
		if err := uc.Execute(context.Background(), inv); err != nil {
			t.Fatalf("Execute() error = %v, want nil (LLM failure must not fail the invoice)", err)
		}
		if inv.Lines[0].ConceptID != nil {
			t.Errorf("ConceptID = %v, want nil", inv.Lines[0].ConceptID)
		}
		if len(cache.saved) != 0 {
			t.Errorf("len(cache.saved) = %d, want 0 (a failed classification must not be cached)", len(cache.saved))
		}
	})

	t.Run("error del repositorio de cache propaga error", func(t *testing.T) {
		cache := newFakeCache()
		cache.findErr = errors.New("db connection lost")
		classifier := &fakeLineClassifier{}
		uc := usecase.NewClassifyInvoiceLines(cache, classifier)

		inv := baseClassifyInvoice()
		if err := uc.Execute(context.Background(), inv); err == nil {
			t.Fatal("expected an error when the cache repository fails")
		}
	})

	t.Run("multiples lineas: una falla y otra no se afectan entre si", func(t *testing.T) {
		cache := newFakeCache()
		classifier := &fakeLineClassifier{err: errors.New("timeout")}
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
	})
}
