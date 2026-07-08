package anthropic_test

import (
	"context"
	"os"
	"testing"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"

	"cifrato/internal/domain/entity"
	"cifrato/internal/infrastructure/adapters/api/anthropic"
)

// TestClassifier_Classify_RealAPI hits the real Anthropic API. It's skipped
// unless ANTHROPIC_API_KEY is set, same pattern as the DB_HOST skip used for
// Postgres integration tests (see postgres/invoice_repository_impl_test.go).
func TestClassifier_Classify_RealAPI(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test against the real Anthropic API")
	}

	concepts := []entity.Concept{
		{ID: 1, Code: "compra_bienes", Name: "Compra de bienes"},
		{ID: 2, Code: "servicios_generales", Name: "Servicios generales"},
		{ID: 3, Code: "transporte_carga", Name: "Transporte de carga"},
	}

	client := anthropicsdk.NewClient()
	classifier, err := anthropic.NewClassifier(client, "claude-haiku-4-5", concepts)
	if err != nil {
		t.Fatalf("NewClassifier() error = %v", err)
	}

	// Real descriptions taken from sample-invoices/, one per concept.
	cases := []struct {
		name          string
		description   string
		wantConceptID uint
	}{
		{
			name:          "compra_bienes",
			description:   "MANTEQUILLA PERFUMADA 240 ML",
			wantConceptID: 1,
		},
		{
			name:          "servicios_generales",
			description:   "Servicio de parqueadero. Placa vehículo: XIN53G. Fecha Entrada: 08/04/2025 11:00:02 - Fecha Salida: 08/04/2025 11:14:47",
			wantConceptID: 2,
		},
		{
			name:          "transporte_carga",
			description:   "Servicios de transporte de carga insumos desde planta a club los arayanes",
			wantConceptID: 3,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := classifier.Classify(context.Background(), c.description)
			if err != nil {
				t.Fatalf("Classify() error = %v", err)
			}
			if got.ConceptID != c.wantConceptID {
				t.Errorf("ConceptID = %d, want %d (model reasoning: %s)", got.ConceptID, c.wantConceptID, got.Reasoning)
			}
			if got.Confidence <= 0 || got.Confidence > 1 {
				t.Errorf("Confidence = %f, want in (0, 1]", got.Confidence)
			}
			if got.ModelVersion != "claude-haiku-4-5" {
				t.Errorf("ModelVersion = %q, want claude-haiku-4-5", got.ModelVersion)
			}
		})
	}
}
