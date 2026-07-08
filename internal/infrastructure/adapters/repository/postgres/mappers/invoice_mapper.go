package mappers

import (
	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/enums"
	"cifrato/internal/infrastructure/adapters/repository/postgres/model"
)

func InvoiceToModel(inv *entity.Invoice) *model.InvoiceModel {
	lines := make([]model.InvoiceLineModel, len(inv.Lines))
	for i, l := range inv.Lines {
		lines[i] = *InvoiceLineToModel(&l)
	}

	return &model.InvoiceModel{
		ID:                      inv.ID,
		CUFE:                    inv.CUFE,
		InvoiceNumber:           inv.InvoiceNumber,
		IssueDate:               inv.IssueDate,
		XMLType:                 string(inv.XMLType),
		IssuerNIT:               inv.IssuerNIT,
		IssuerName:              inv.IssuerName,
		IssuerCity:              inv.IssuerCity,
		IssuerTaxResponsibility: inv.IssuerTaxResponsibility,
		BuyerNIT:                inv.BuyerNIT,
		BuyerName:               inv.BuyerName,
		Subtotal:                inv.Subtotal,
		IVATotal:                inv.IVATotal,
		InvoiceTotal:            inv.InvoiceTotal,
		SourceXMLPath:           inv.SourceXMLPath,
		SourcePDFPath:           inv.SourcePDFPath,
		ReportedRetefuente:      inv.ReportedRetefuente,
		ReportedReteiva:         inv.ReportedReteiva,
		ReportedReteica:         inv.ReportedReteica,
		Lines:                   lines,
	}
}

func ModelToInvoice(m *model.InvoiceModel) *entity.Invoice {
	lines := make([]entity.InvoiceLine, len(m.Lines))
	for i := range m.Lines {
		lines[i] = *ModelToInvoiceLine(&m.Lines[i])
	}

	return &entity.Invoice{
		ID:                      m.ID,
		CUFE:                    m.CUFE,
		InvoiceNumber:           m.InvoiceNumber,
		IssueDate:               m.IssueDate,
		XMLType:                 enums.XMLType(m.XMLType),
		IssuerNIT:               m.IssuerNIT,
		IssuerName:              m.IssuerName,
		IssuerCity:              m.IssuerCity,
		IssuerTaxResponsibility: m.IssuerTaxResponsibility,
		BuyerNIT:                m.BuyerNIT,
		BuyerName:               m.BuyerName,
		Subtotal:                m.Subtotal,
		IVATotal:                m.IVATotal,
		InvoiceTotal:            m.InvoiceTotal,
		SourceXMLPath:           m.SourceXMLPath,
		SourcePDFPath:           m.SourcePDFPath,
		ReportedRetefuente:      m.ReportedRetefuente,
		ReportedReteiva:         m.ReportedReteiva,
		ReportedReteica:         m.ReportedReteica,
		Lines:                   lines,
	}
}

// InvoiceLineToModel deliberately omits ID: it is only used to build lines
// for a fresh insert (Save always deletes and reinserts lines), so leaving
// it zero lets GORM auto-generate it.
func InvoiceLineToModel(l *entity.InvoiceLine) *model.InvoiceLineModel {
	return &model.InvoiceLineModel{
		InvoiceID:                l.InvoiceID,
		LineNumber:               l.LineNumber,
		SKU:                      l.SKU,
		Description:              l.Description,
		Quantity:                 l.Quantity,
		UnitPrice:                l.UnitPrice,
		LineTotal:                l.LineTotal,
		IVARate:                  l.IVARate,
		IVAValue:                 l.IVAValue,
		ConceptID:                l.ConceptID,
		ClassificationConfidence: l.ClassificationConfidence,
	}
}

func ModelToInvoiceLine(m *model.InvoiceLineModel) *entity.InvoiceLine {
	return &entity.InvoiceLine{
		ID:                       m.ID,
		InvoiceID:                m.InvoiceID,
		LineNumber:               m.LineNumber,
		SKU:                      m.SKU,
		Description:              m.Description,
		Quantity:                 m.Quantity,
		UnitPrice:                m.UnitPrice,
		LineTotal:                m.LineTotal,
		IVARate:                  m.IVARate,
		IVAValue:                 m.IVAValue,
		ConceptID:                m.ConceptID,
		ClassificationConfidence: m.ClassificationConfidence,
	}
}
