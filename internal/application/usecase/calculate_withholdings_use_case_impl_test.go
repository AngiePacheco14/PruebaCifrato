package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	appconfig "cifrato/internal/application/config"
	"cifrato/internal/application/usecase"
	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/enums"
)

type fakeTaxRuleRepo struct{ rules []entity.TaxRule }

func (f *fakeTaxRuleRepo) FindApplicable(_ context.Context, taxType enums.TaxType, conceptID uint, cityID *uint, at time.Time) (*entity.TaxRule, error) {
	for _, r := range f.rules {
		if r.TaxType != taxType || r.ConceptID != conceptID {
			continue
		}
		if (cityID == nil) != (r.CityID == nil) {
			continue
		}
		if cityID != nil && r.CityID != nil && *cityID != *r.CityID {
			continue
		}
		if at.Before(r.EffectiveFrom) {
			continue
		}
		if r.EffectiveTo != nil && at.After(*r.EffectiveTo) {
			continue
		}
		rc := r
		return &rc, nil
	}
	return nil, nil
}

func (f *fakeTaxRuleRepo) ListByTaxType(context.Context, enums.TaxType) ([]entity.TaxRule, error) {
	return nil, nil
}

type fakeReferenceData struct {
	cities map[string]entity.City
	uvt    *entity.UVTValue
}

func (f *fakeReferenceData) FindConceptByCode(context.Context, string) (*entity.Concept, error) {
	return nil, nil
}
func (f *fakeReferenceData) ListConcepts(context.Context) ([]entity.Concept, error) {
	return nil, nil
}
func (f *fakeReferenceData) FindCityByName(_ context.Context, name string) (*entity.City, error) {
	c, ok := f.cities[name]
	if !ok {
		return nil, nil
	}
	cc := c
	return &cc, nil
}
func (f *fakeReferenceData) FindUVTValue(context.Context, time.Time) (*entity.UVTValue, error) {
	return f.uvt, nil
}

type fakeCalculationRepo struct{ saved []entity.Calculation }

func (f *fakeCalculationRepo) Upsert(_ context.Context, c *entity.Calculation) error {
	c.ID = uint(len(f.saved) + 1)
	f.saved = append(f.saved, *c)
	return nil
}
func (f *fakeCalculationRepo) ListByInvoice(context.Context, uint) ([]entity.Calculation, error) {
	return nil, nil
}

const compraBienesConceptID uint = 1

func baseInvoice() *entity.Invoice {
	conceptID := compraBienesConceptID
	return &entity.Invoice{
		ID:         1,
		IssueDate:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		IssuerCity: "BOGOTA D.C",
		Lines: []entity.InvoiceLine{
			{
				ID:        1,
				LineTotal: decimal.RequireFromString("10000000"),
				IVAValue:  decimal.RequireFromString("1900000"),
				ConceptID: &conceptID,
			},
		},
	}
}

func baseRetefuenteRule() entity.TaxRule {
	return entity.TaxRule{
		TaxType:          enums.TaxTypeRetefuente,
		ConceptID:        compraBienesConceptID,
		MinBaseUVT:       decimal.RequireFromString("10"),
		TariffPercentage: decimal.RequireFromString("2.5"),
		LegalBasis:       "Art. 401 E.T.",
		EffectiveFrom:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func baseReteivaRule() entity.TaxRule {
	return entity.TaxRule{
		TaxType:          enums.TaxTypeReteiva,
		ConceptID:        compraBienesConceptID,
		TariffPercentage: decimal.RequireFromString("15"),
		LegalBasis:       "Art. 437-2 E.T.",
		EffectiveFrom:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func findByTaxType(calcs []entity.Calculation, tt enums.TaxType) *entity.Calculation {
	for i := range calcs {
		if calcs[i].TaxType == tt {
			return &calcs[i]
		}
	}
	return nil
}

func TestCalculateWithholdings_Execute(t *testing.T) {
	uvt := &entity.UVTValue{Value: decimal.RequireFromString("52374")}

	t.Run("linea supera minimo RETEFUENTE", func(t *testing.T) {
		taxRules := &fakeTaxRuleRepo{rules: []entity.TaxRule{baseRetefuenteRule()}}
		calcs := &fakeCalculationRepo{}
		uc := usecase.NewCalculateWithholdings(taxRules, calcs, &fakeReferenceData{uvt: uvt}, appconfig.Config{})

		results, err := uc.Execute(context.Background(), baseInvoice())
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		refuente := findByTaxType(results, enums.TaxTypeRetefuente)
		if refuente == nil {
			t.Fatal("no RETEFUENTE calculation produced")
		}
		want := decimal.RequireFromString("250000") // 10,000,000 * 2.5%
		if !refuente.CalculatedValue.Equal(want) {
			t.Errorf("CalculatedValue = %s, want %s", refuente.CalculatedValue, want)
		}
	})

	t.Run("linea no supera minimo RETEFUENTE", func(t *testing.T) {
		taxRules := &fakeTaxRuleRepo{rules: []entity.TaxRule{baseRetefuenteRule()}}
		calcs := &fakeCalculationRepo{}
		uc := usecase.NewCalculateWithholdings(taxRules, calcs, &fakeReferenceData{uvt: uvt}, appconfig.Config{})

		inv := baseInvoice()
		inv.Lines[0].LineTotal = decimal.RequireFromString("100000") // below 10 UVT ($523,740)

		results, err := uc.Execute(context.Background(), inv)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		refuente := findByTaxType(results, enums.TaxTypeRetefuente)
		if refuente == nil {
			t.Fatal("no RETEFUENTE calculation produced")
		}
		if !refuente.CalculatedValue.IsZero() {
			t.Errorf("CalculatedValue = %s, want 0", refuente.CalculatedValue)
		}
		if refuente.Justification == "" {
			t.Error("expected a non-empty justification")
		}
	})

	t.Run("ciudad sin tarifa ICA", func(t *testing.T) {
		taxRules := &fakeTaxRuleRepo{}
		calcs := &fakeCalculationRepo{}
		uc := usecase.NewCalculateWithholdings(taxRules, calcs, &fakeReferenceData{uvt: uvt, cities: map[string]entity.City{}}, appconfig.Config{})

		inv := baseInvoice()
		inv.IssuerCity = "CARTAGENA"

		results, err := uc.Execute(context.Background(), inv)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		reteica := findByTaxType(results, enums.TaxTypeReteica)
		if reteica == nil {
			t.Fatal("no RETEICA calculation produced")
		}
		if !reteica.CalculatedValue.IsZero() {
			t.Errorf("CalculatedValue = %s, want 0", reteica.CalculatedValue)
		}
		want := "sin tarifa ICA configurada para la ciudad CARTAGENA"
		if reteica.Justification != want {
			t.Errorf("Justification = %q, want %q", reteica.Justification, want)
		}
	})

	t.Run("comprador no agente de IVA", func(t *testing.T) {
		taxRules := &fakeTaxRuleRepo{rules: []entity.TaxRule{baseReteivaRule()}}
		calcs := &fakeCalculationRepo{}
		uc := usecase.NewCalculateWithholdings(taxRules, calcs, &fakeReferenceData{uvt: uvt}, appconfig.Config{IsVATWithholdingAgent: false})

		results, err := uc.Execute(context.Background(), baseInvoice())
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		reteiva := findByTaxType(results, enums.TaxTypeReteiva)
		if reteiva == nil {
			t.Fatal("no RETEIVA calculation produced")
		}
		if !reteiva.CalculatedValue.IsZero() {
			t.Errorf("CalculatedValue = %s, want 0", reteiva.CalculatedValue)
		}
		want := "el comprador no es agente de retención de IVA"
		if reteiva.Justification != want {
			t.Errorf("Justification = %q, want %q", reteiva.Justification, want)
		}
	})

	t.Run("linea sin ConceptID", func(t *testing.T) {
		taxRules := &fakeTaxRuleRepo{}
		calcs := &fakeCalculationRepo{}
		uc := usecase.NewCalculateWithholdings(taxRules, calcs, &fakeReferenceData{uvt: uvt}, appconfig.Config{})

		inv := baseInvoice()
		inv.Lines[0].ConceptID = nil

		results, err := uc.Execute(context.Background(), inv)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("len(results) = %d, want 3 (one per tax type)", len(results))
		}
		for _, c := range results {
			if !c.CalculatedValue.IsZero() {
				t.Errorf("%s CalculatedValue = %s, want 0", c.TaxType, c.CalculatedValue)
			}
			if c.ConceptID != nil {
				t.Errorf("%s ConceptID = %v, want nil", c.TaxType, c.ConceptID)
			}
			want := "línea sin concepto clasificado, no se puede determinar la regla aplicable"
			if c.Justification != want {
				t.Errorf("%s Justification = %q, want %q", c.TaxType, c.Justification, want)
			}
		}
		if len(calcs.saved) != 3 {
			t.Errorf("len(calcs.saved) = %d, want 3", len(calcs.saved))
		}
	})

	t.Run("factura sin persistir retorna error", func(t *testing.T) {
		taxRules := &fakeTaxRuleRepo{}
		calcs := &fakeCalculationRepo{}
		uc := usecase.NewCalculateWithholdings(taxRules, calcs, &fakeReferenceData{uvt: uvt}, appconfig.Config{})

		inv := baseInvoice()
		inv.ID = 0

		_, err := uc.Execute(context.Background(), inv)
		if err == nil {
			t.Fatal("expected an error for an unpersisted invoice")
		}
		if len(calcs.saved) != 0 {
			t.Errorf("expected no repository calls, got %d saved calculations", len(calcs.saved))
		}
	})
}
