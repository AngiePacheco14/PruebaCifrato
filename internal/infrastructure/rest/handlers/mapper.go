package handlers

import (
	"cifrato/internal/application/ports/in"
	"cifrato/internal/infrastructure/rest/dto"
)

func toProcessInvoiceResponse(result *in.ProcessInvoiceResult) dto.ProcessInvoiceResponse {
	calculations := make([]dto.CalculationDTO, len(result.Calculations))
	for i, calc := range result.Calculations {
		calculations[i] = dto.CalculationDTO{
			TaxType:         string(calc.TaxType),
			ConceptID:       calc.ConceptID,
			ConceptName:     calc.ConceptName,
			BaseAmount:      calc.BaseAmount.String(),
			TariffApplied:   calc.TariffApplied.String(),
			CalculatedValue: calc.CalculatedValue.String(),
			LegalBasis:      calc.LegalBasis,
			Justification:   calc.Justification,
		}
	}

	inv := result.Invoice
	return dto.ProcessInvoiceResponse{
		CUFE:          inv.CUFE,
		InvoiceNumber: inv.InvoiceNumber,
		IssuerNIT:     inv.IssuerNIT,
		IssuerName:    inv.IssuerName,
		InvoiceTotal:  inv.InvoiceTotal.String(),
		Summary: dto.SummaryDTO{
			TotalRetefuente: result.Summary.TotalRetefuente.String(),
			TotalReteiva:    result.Summary.TotalReteiva.String(),
			TotalReteica:    result.Summary.TotalReteica.String(),
		},
		Calculations: calculations,
	}
}
