package usecase

import (
	"context"
	"fmt"

	"cifrato/internal/application/ports/in"
	"cifrato/internal/domain/repository"
)

// ProcessInvoice orchestrates the full pipeline for one invoice XML: parse,
// persist, classify lines, and calculate withholdings. It is the composition
// root's single entry point for a driving adapter (HTTP handler, CLI
// command, etc.) — none of them need to know the individual steps exist.
type ProcessInvoice struct {
	parser                repository.InvoiceParser
	invoices              repository.InvoiceRepository
	classifyLines         in.ClassifyInvoiceLines
	calculateWithholdings in.CalculateWithholdings
}

func NewProcessInvoice(
	parser repository.InvoiceParser,
	invoices repository.InvoiceRepository,
	classifyLines in.ClassifyInvoiceLines,
	calculateWithholdings in.CalculateWithholdings,
) *ProcessInvoice {
	return &ProcessInvoice{
		parser:                parser,
		invoices:              invoices,
		classifyLines:         classifyLines,
		calculateWithholdings: calculateWithholdings,
	}
}

var _ in.ProcessInvoice = (*ProcessInvoice)(nil)

func (uc *ProcessInvoice) Execute(ctx context.Context, xmlData []byte, sourceXMLPath, sourcePDFPath string) (*in.ProcessInvoiceResult, error) {
	inv, err := uc.parser.Parse(xmlData)
	if err != nil {
		return nil, fmt.Errorf("usecase: parsing invoice XML: %w", err)
	}
	inv.SourceXMLPath = sourceXMLPath
	inv.SourcePDFPath = sourcePDFPath

	if err := uc.invoices.Save(ctx, inv); err != nil {
		return nil, fmt.Errorf("usecase: saving invoice %s: %w", inv.CUFE, err)
	}

	if err := uc.classifyLines.Execute(ctx, inv); err != nil {
		return nil, fmt.Errorf("usecase: classifying lines for invoice %s: %w", inv.CUFE, err)
	}

	calculations, err := uc.calculateWithholdings.Execute(ctx, inv)
	if err != nil {
		return nil, fmt.Errorf("usecase: calculating withholdings for invoice %s: %w", inv.CUFE, err)
	}

	return &in.ProcessInvoiceResult{Invoice: inv, Calculations: calculations}, nil
}
