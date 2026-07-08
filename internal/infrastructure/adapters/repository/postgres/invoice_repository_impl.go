package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/repository"
	"cifrato/internal/infrastructure/adapters/repository/postgres/mappers"
	"cifrato/internal/infrastructure/adapters/repository/postgres/model"
)

type InvoiceRepository struct{ db *gorm.DB }

func NewInvoiceRepository(db *gorm.DB) *InvoiceRepository { return &InvoiceRepository{db: db} }

var _ repository.InvoiceRepository = (*InvoiceRepository)(nil)

func (r *InvoiceRepository) Save(ctx context.Context, inv *entity.Invoice) error {
	row := mappers.InvoiceToModel(inv)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "cufe"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"invoice_number", "issue_date", "xml_type", "issuer_nit", "issuer_name",
				"issuer_city", "issuer_tax_responsibility", "buyer_nit", "buyer_name",
				"subtotal", "iva_total", "invoice_total", "source_xml_path", "source_pdf_path",
				"reported_retefuente", "reported_reteiva", "reported_reteica", "updated_at",
			}),
		}).Create(row).Error; err != nil {
			return fmt.Errorf("postgres: upserting invoice: %w", err)
		}
		if err := tx.Where("invoice_id = ?", row.ID).Delete(&model.InvoiceLineModel{}).Error; err != nil {
			return fmt.Errorf("postgres: clearing previous invoice lines: %w", err)
		}
		for i := range row.Lines {
			row.Lines[i].InvoiceID = row.ID
		}
		if len(row.Lines) > 0 {
			if err := tx.Create(&row.Lines).Error; err != nil {
				return fmt.Errorf("postgres: inserting invoice lines: %w", err)
			}
		}
		inv.ID = row.ID
		for i := range row.Lines {
			inv.Lines[i].ID = row.Lines[i].ID
		}
		return nil
	})
}

func (r *InvoiceRepository) FindByCUFE(ctx context.Context, cufe string) (*entity.Invoice, error) {
	var row model.InvoiceModel
	q := r.db.WithContext(ctx).Preload("Lines").Where("cufe = ?", cufe)
	found, err := findOne(q, &row, "finding invoice by cufe")
	if err != nil || found == nil {
		return nil, err
	}
	return mappers.ModelToInvoice(found), nil
}

func (r *InvoiceRepository) ExistsByCUFE(ctx context.Context, cufe string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.InvoiceModel{}).Where("cufe = ?", cufe).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("postgres: checking invoice existence: %w", err)
	}
	return count > 0, nil
}
