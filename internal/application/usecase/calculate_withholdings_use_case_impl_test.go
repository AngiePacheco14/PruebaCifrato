package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"

	appconfig "cifrato/internal/application/config"
	"cifrato/internal/application/usecase"
	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/enums"
	"cifrato/internal/domain/repository/mocks"
)

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

// mockReferenceData stubs FindUVTValue with uvt and FindCityByName to always
// miss (nil, nil) — the common case across most subtests, since RETEICA is
// not the thing under test there.
func mockReferenceData(t *testing.T, uvt *entity.UVTValue) *mocks.MockReferenceDataRepository {
	ref := mocks.NewMockReferenceDataRepository(t)
	ref.EXPECT().FindUVTValue(mock.Anything, mock.Anything).Return(uvt, nil)
	ref.EXPECT().FindCityByName(mock.Anything, mock.Anything).Return(nil, nil)
	ref.EXPECT().ListConcepts(mock.Anything).Return(testConcepts(), nil)
	return ref
}

// testConcepts covers every ConceptID used across this file's subtests
// (compraBienesConceptID and the ad-hoc servicioConceptID = 2).
func testConcepts() []entity.Concept {
	return []entity.Concept{
		{ID: compraBienesConceptID, Code: "compra_bienes", Name: "Compra de bienes"},
		{ID: 2, Code: "servicios_generales", Name: "Servicios generales"},
	}
}

// mockCalculationRepo stubs Upsert to accept any calculation — the common
// case across most subtests, since persistence itself isn't what's under test.
func mockCalculationRepo(t *testing.T) *mocks.MockCalculationRepository {
	calcs := mocks.NewMockCalculationRepository(t)
	calcs.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	return calcs
}

func TestCalculateWithholdings_Execute(t *testing.T) {
	uvt := &entity.UVTValue{Value: decimal.RequireFromString("52374")}

	t.Run("linea supera minimo RETEFUENTE", func(t *testing.T) {
		rule := baseRetefuenteRule()
		taxRules := mocks.NewMockTaxRuleRepository(t)
		taxRules.EXPECT().FindApplicable(mock.Anything, enums.TaxTypeRetefuente, compraBienesConceptID, mock.Anything, mock.Anything).Return(&rule, nil)
		uc := usecase.NewCalculateWithholdings(taxRules, mockCalculationRepo(t), mockReferenceData(t, uvt), appconfig.Config{})

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
		rule := baseRetefuenteRule()
		taxRules := mocks.NewMockTaxRuleRepository(t)
		taxRules.EXPECT().FindApplicable(mock.Anything, enums.TaxTypeRetefuente, compraBienesConceptID, mock.Anything, mock.Anything).Return(&rule, nil)
		uc := usecase.NewCalculateWithholdings(taxRules, mockCalculationRepo(t), mockReferenceData(t, uvt), appconfig.Config{})

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
		taxRules := mocks.NewMockTaxRuleRepository(t)
		taxRules.EXPECT().FindApplicable(mock.Anything, enums.TaxTypeRetefuente, compraBienesConceptID, mock.Anything, mock.Anything).Return(nil, nil)
		ref := mocks.NewMockReferenceDataRepository(t)
		ref.EXPECT().FindUVTValue(mock.Anything, mock.Anything).Return(uvt, nil)
		ref.EXPECT().FindCityByName(mock.Anything, "CARTAGENA").Return(nil, nil)
		ref.EXPECT().ListConcepts(mock.Anything).Return(testConcepts(), nil)
		uc := usecase.NewCalculateWithholdings(taxRules, mockCalculationRepo(t), ref, appconfig.Config{})

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
		taxRules := mocks.NewMockTaxRuleRepository(t)
		taxRules.EXPECT().FindApplicable(mock.Anything, enums.TaxTypeRetefuente, compraBienesConceptID, mock.Anything, mock.Anything).Return(nil, nil)
		uc := usecase.NewCalculateWithholdings(taxRules, mockCalculationRepo(t), mockReferenceData(t, uvt), appconfig.Config{IsVATWithholdingAgent: false})

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
		// RETEIVA's gate is checked before any repository lookup — FindApplicable
		// must never be called for RETEIVA when the buyer isn't a withholding agent.
		taxRules.AssertNotCalled(t, "FindApplicable", mock.Anything, enums.TaxTypeReteiva, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("linea sin ConceptID", func(t *testing.T) {
		taxRules := mocks.NewMockTaxRuleRepository(t)
		calcs := mockCalculationRepo(t)
		uc := usecase.NewCalculateWithholdings(taxRules, calcs, mockReferenceData(t, uvt), appconfig.Config{})

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
			want := "línea(s) sin concepto clasificado, no se puede determinar la regla aplicable"
			if c.Justification != want {
				t.Errorf("%s Justification = %q, want %q", c.TaxType, c.Justification, want)
			}
		}
		calcs.AssertNumberOfCalls(t, "Upsert", 3)
		taxRules.AssertNotCalled(t, "FindApplicable", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("varias lineas del mismo concepto se suman antes de evaluar el minimo", func(t *testing.T) {
		// Two lines individually below the minimum, but summed together they clear it.
		rule := baseRetefuenteRule()
		taxRules := mocks.NewMockTaxRuleRepository(t)
		taxRules.EXPECT().FindApplicable(mock.Anything, enums.TaxTypeRetefuente, compraBienesConceptID, mock.Anything, mock.Anything).Return(&rule, nil)
		uc := usecase.NewCalculateWithholdings(taxRules, mockCalculationRepo(t), mockReferenceData(t, uvt), appconfig.Config{})

		conceptID := compraBienesConceptID
		inv := baseInvoice()
		inv.Lines = []entity.InvoiceLine{
			{ID: 1, LineTotal: decimal.RequireFromString("300000"), IVAValue: decimal.Zero, ConceptID: &conceptID},
			{ID: 2, LineTotal: decimal.RequireFromString("300000"), IVAValue: decimal.Zero, ConceptID: &conceptID},
		}

		results, err := uc.Execute(context.Background(), inv)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		refuenteCalcs := 0
		for _, c := range results {
			if c.TaxType == enums.TaxTypeRetefuente {
				refuenteCalcs++
				want := decimal.RequireFromString("15000") // 600,000 * 2.5%
				if !c.CalculatedValue.Equal(want) {
					t.Errorf("CalculatedValue = %s, want %s (combined base, not per-line)", c.CalculatedValue, want)
				}
				if !c.BaseAmount.Equal(decimal.RequireFromString("600000")) {
					t.Errorf("BaseAmount = %s, want 600000 (sum of both lines)", c.BaseAmount)
				}
			}
		}
		if refuenteCalcs != 1 {
			t.Errorf("got %d RETEFUENTE calculations, want exactly 1 (one per concept, not one per line)", refuenteCalcs)
		}
		// One call for the whole group, not one per line.
		taxRules.AssertNumberOfCalls(t, "FindApplicable", 1)
	})

	t.Run("lineas de conceptos distintos producen grupos separados, no se mezclan", func(t *testing.T) {
		servicioConceptID := uint(2)
		bienesRule := baseRetefuenteRule() // compra_bienes: 2.5%, min 10 UVT
		servicioRule := entity.TaxRule{
			TaxType: enums.TaxTypeRetefuente, ConceptID: servicioConceptID,
			MinBaseUVT: decimal.RequireFromString("2"), TariffPercentage: decimal.RequireFromString("4"),
			LegalBasis: "Art. 392 E.T.", EffectiveFrom: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		taxRules := mocks.NewMockTaxRuleRepository(t)
		taxRules.EXPECT().FindApplicable(mock.Anything, enums.TaxTypeRetefuente, compraBienesConceptID, mock.Anything, mock.Anything).Return(&bienesRule, nil)
		taxRules.EXPECT().FindApplicable(mock.Anything, enums.TaxTypeRetefuente, servicioConceptID, mock.Anything, mock.Anything).Return(&servicioRule, nil)
		calcs := mockCalculationRepo(t)
		uc := usecase.NewCalculateWithholdings(taxRules, calcs, mockReferenceData(t, uvt), appconfig.Config{})

		bienesConceptID := compraBienesConceptID
		inv := baseInvoice()
		inv.Lines = []entity.InvoiceLine{
			{ID: 1, LineTotal: decimal.RequireFromString("10000000"), IVAValue: decimal.Zero, ConceptID: &bienesConceptID},
			{ID: 2, LineTotal: decimal.RequireFromString("1000000"), IVAValue: decimal.Zero, ConceptID: &servicioConceptID},
		}

		results, err := uc.Execute(context.Background(), inv)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		var refuenteResults []entity.Calculation
		for _, c := range results {
			if c.TaxType == enums.TaxTypeRetefuente {
				refuenteResults = append(refuenteResults, c)
			}
		}
		if len(refuenteResults) != 2 {
			t.Fatalf("got %d RETEFUENTE calculations, want 2 (one per concept)", len(refuenteResults))
		}
		for _, c := range refuenteResults {
			if c.ConceptID == nil {
				t.Fatal("ConceptID = nil, want it set on a classified group")
			}
			switch *c.ConceptID {
			case bienesConceptID:
				want := decimal.RequireFromString("250000") // 10,000,000 * 2.5%
				if !c.CalculatedValue.Equal(want) {
					t.Errorf("compra_bienes CalculatedValue = %s, want %s", c.CalculatedValue, want)
				}
			case servicioConceptID:
				want := decimal.RequireFromString("40000") // 1,000,000 * 4%
				if !c.CalculatedValue.Equal(want) {
					t.Errorf("servicios_generales CalculatedValue = %s, want %s", c.CalculatedValue, want)
				}
			default:
				t.Errorf("unexpected ConceptID %d", *c.ConceptID)
			}
		}
		calcs.AssertNumberOfCalls(t, "Upsert", 6) // 2 concepts x 3 tax types
	})

	t.Run("factura sin persistir retorna error", func(t *testing.T) {
		// No expectations set on any mock: Execute must return before touching
		// any repository, or testify panics on the first unexpected call.
		taxRules := mocks.NewMockTaxRuleRepository(t)
		calcs := mocks.NewMockCalculationRepository(t)
		ref := mocks.NewMockReferenceDataRepository(t)
		uc := usecase.NewCalculateWithholdings(taxRules, calcs, ref, appconfig.Config{})

		inv := baseInvoice()
		inv.ID = 0

		_, err := uc.Execute(context.Background(), inv)
		if err == nil {
			t.Fatal("expected an error for an unpersisted invoice")
		}
	})
}
