package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"cifrato/internal/adapters/driven/postgres/mappers"
	"cifrato/internal/adapters/driven/postgres/models"
	"cifrato/internal/application/ports/out"
	"cifrato/internal/domain/invoice"
)

type InvoiceRepository struct{ db *gorm.DB }

func NewInvoiceRepository(db *gorm.DB) *InvoiceRepository { return &InvoiceRepository{db: db} }

var _ out.InvoiceRepository = (*InvoiceRepository)(nil)

func (r *InvoiceRepository) Save(ctx context.Context, inv *invoice.Invoice) error {
	model := mappers.InvoiceToModel(inv)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "cufe"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"invoice_number", "issue_date", "xml_type", "issuer_nit", "issuer_name",
				"issuer_city", "issuer_tax_responsibility", "buyer_nit", "buyer_name",
				"subtotal", "iva_total", "invoice_total", "source_xml_path", "source_pdf_path",
				"reported_retefuente", "reported_reteiva", "reported_reteica", "updated_at",
			}),
		}).Create(model).Error; err != nil {
			return fmt.Errorf("postgres: upserting invoice: %w", err)
		}
		if err := tx.Where("invoice_id = ?", model.ID).Delete(&models.InvoiceLineModel{}).Error; err != nil {
			return fmt.Errorf("postgres: clearing previous invoice lines: %w", err)
		}
		for i := range model.Lines {
			model.Lines[i].InvoiceID = model.ID
		}
		if len(model.Lines) > 0 {
			if err := tx.Create(&model.Lines).Error; err != nil {
				return fmt.Errorf("postgres: inserting invoice lines: %w", err)
			}
		}
		inv.ID = model.ID
		for i := range model.Lines {
			inv.Lines[i].ID = model.Lines[i].ID
		}
		return nil
	})
}

func (r *InvoiceRepository) FindByCUFE(ctx context.Context, cufe string) (*invoice.Invoice, error) {
	var model models.InvoiceModel
	q := r.db.WithContext(ctx).Preload("Lines").Where("cufe = ?", cufe)
	found, err := findOne(q, &model, "finding invoice by cufe")
	if err != nil || found == nil {
		return nil, err
	}
	return mappers.ModelToInvoice(found), nil
}

func (r *InvoiceRepository) ExistsByCUFE(ctx context.Context, cufe string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.InvoiceModel{}).Where("cufe = ?", cufe).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("postgres: checking invoice existence: %w", err)
	}
	return count > 0, nil
}
