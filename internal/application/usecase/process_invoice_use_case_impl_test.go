package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"

	inmocks "cifrato/internal/application/ports/in/mocks"
	"cifrato/internal/application/usecase"
	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/enums"
	repomocks "cifrato/internal/domain/repository/mocks"
)

func parsedInvoice() *entity.Invoice {
	return &entity.Invoice{
		CUFE: "test-cufe",
		Lines: []entity.InvoiceLine{
			{LineNumber: 1, Description: "MANTEQUILLA PERFUMADA 240 ML", LineTotal: decimal.RequireFromString("100000")},
		},
	}
}

func TestProcessInvoice_Execute(t *testing.T) {
	xmlData := []byte("<Invoice/>")

	t.Run("pipeline exitoso: parsea, clasifica antes de guardar, y calcula", func(t *testing.T) {
		parser := repomocks.NewMockInvoiceParser(t)
		invoices := repomocks.NewMockInvoiceRepository(t)
		classifyLines := inmocks.NewMockClassifyInvoiceLines(t)
		calculateWithholdings := inmocks.NewMockCalculateWithholdings(t)

		inv := parsedInvoice()
		parseCall := parser.EXPECT().Parse(xmlData).Return(inv, nil)

		classifyCall := classifyLines.EXPECT().Execute(mock.Anything, inv).Run(func(_ context.Context, inv *entity.Invoice) {
			conceptID := uint(1)
			inv.Lines[0].ConceptID = &conceptID
		}).Return(nil)

		saveCall := invoices.EXPECT().Save(mock.Anything, inv).Run(func(_ context.Context, inv *entity.Invoice) {
			inv.ID = 1
		}).Return(nil)

		calcs := []entity.Calculation{
			{TaxType: enums.TaxTypeRetefuente, CalculatedValue: decimal.RequireFromString("2500")},
			{TaxType: enums.TaxTypeReteiva, CalculatedValue: decimal.RequireFromString("1500")},
		}
		calcCall := calculateWithholdings.EXPECT().Execute(mock.Anything, inv).Return(calcs, nil)

		// The invoice must be classified before it's saved, so concept_id lands
		// on the first write instead of a second update.
		mock.InOrder(parseCall.Call, classifyCall.Call, saveCall.Call, calcCall.Call)

		uc := usecase.NewProcessInvoice(parser, invoices, classifyLines, calculateWithholdings)
		result, err := uc.Execute(context.Background(), xmlData, "factura.xml", "factura.pdf")
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if result.Invoice.SourceXMLPath != "factura.xml" || result.Invoice.SourcePDFPath != "factura.pdf" {
			t.Errorf("SourceXMLPath/SourcePDFPath = %q/%q, want factura.xml/factura.pdf", result.Invoice.SourceXMLPath, result.Invoice.SourcePDFPath)
		}
		if result.Invoice.Lines[0].ConceptID == nil || *result.Invoice.Lines[0].ConceptID != 1 {
			t.Errorf("ConceptID = %v, want 1 (classification must land on the returned invoice)", result.Invoice.Lines[0].ConceptID)
		}
		if len(result.Calculations) != 2 {
			t.Fatalf("len(Calculations) = %d, want 2", len(result.Calculations))
		}
		wantRetefuente := decimal.RequireFromString("2500")
		if !result.Summary.TotalRetefuente.Equal(wantRetefuente) {
			t.Errorf("Summary.TotalRetefuente = %s, want %s", result.Summary.TotalRetefuente, wantRetefuente)
		}
		wantReteiva := decimal.RequireFromString("1500")
		if !result.Summary.TotalReteiva.Equal(wantReteiva) {
			t.Errorf("Summary.TotalReteiva = %s, want %s", result.Summary.TotalReteiva, wantReteiva)
		}
	})

	t.Run("error al parsear detiene el pipeline", func(t *testing.T) {
		parser := repomocks.NewMockInvoiceParser(t)
		parser.EXPECT().Parse(xmlData).Return(nil, errors.New("malformed XML"))
		invoices := repomocks.NewMockInvoiceRepository(t)
		classifyLines := inmocks.NewMockClassifyInvoiceLines(t)
		calculateWithholdings := inmocks.NewMockCalculateWithholdings(t)

		uc := usecase.NewProcessInvoice(parser, invoices, classifyLines, calculateWithholdings)
		_, err := uc.Execute(context.Background(), xmlData, "", "")
		if err == nil {
			t.Fatal("expected an error when parsing fails")
		}
	})

	t.Run("error al clasificar detiene el pipeline antes de guardar", func(t *testing.T) {
		parser := repomocks.NewMockInvoiceParser(t)
		inv := parsedInvoice()
		parser.EXPECT().Parse(xmlData).Return(inv, nil)

		classifyLines := inmocks.NewMockClassifyInvoiceLines(t)
		classifyLines.EXPECT().Execute(mock.Anything, inv).Return(errors.New("cache unavailable"))

		invoices := repomocks.NewMockInvoiceRepository(t)
		calculateWithholdings := inmocks.NewMockCalculateWithholdings(t)

		uc := usecase.NewProcessInvoice(parser, invoices, classifyLines, calculateWithholdings)
		_, err := uc.Execute(context.Background(), xmlData, "", "")
		if err == nil {
			t.Fatal("expected an error when classification fails")
		}
	})

	t.Run("error al guardar detiene el pipeline antes de calcular", func(t *testing.T) {
		parser := repomocks.NewMockInvoiceParser(t)
		inv := parsedInvoice()
		parser.EXPECT().Parse(xmlData).Return(inv, nil)

		classifyLines := inmocks.NewMockClassifyInvoiceLines(t)
		classifyLines.EXPECT().Execute(mock.Anything, inv).Return(nil)

		invoices := repomocks.NewMockInvoiceRepository(t)
		invoices.EXPECT().Save(mock.Anything, inv).Return(errors.New("connection refused"))

		calculateWithholdings := inmocks.NewMockCalculateWithholdings(t)

		uc := usecase.NewProcessInvoice(parser, invoices, classifyLines, calculateWithholdings)
		_, err := uc.Execute(context.Background(), xmlData, "", "")
		if err == nil {
			t.Fatal("expected an error when saving fails")
		}
	})

	t.Run("error al calcular retenciones se propaga", func(t *testing.T) {
		parser := repomocks.NewMockInvoiceParser(t)
		inv := parsedInvoice()
		parser.EXPECT().Parse(xmlData).Return(inv, nil)

		classifyLines := inmocks.NewMockClassifyInvoiceLines(t)
		classifyLines.EXPECT().Execute(mock.Anything, inv).Return(nil)

		invoices := repomocks.NewMockInvoiceRepository(t)
		invoices.EXPECT().Save(mock.Anything, inv).Return(nil)

		calculateWithholdings := inmocks.NewMockCalculateWithholdings(t)
		calculateWithholdings.EXPECT().Execute(mock.Anything, inv).Return(nil, errors.New("no UVT value configured"))

		uc := usecase.NewProcessInvoice(parser, invoices, classifyLines, calculateWithholdings)
		_, err := uc.Execute(context.Background(), xmlData, "", "")
		if err == nil {
			t.Fatal("expected an error when calculating withholdings fails")
		}
	})
}
