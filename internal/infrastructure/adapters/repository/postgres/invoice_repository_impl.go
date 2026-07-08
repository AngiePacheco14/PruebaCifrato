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
		// Omit(clause.Associations) prevents GORM from auto-saving Lines with its
		// own ON CONFLICT("id"), which would collide with the natural-key upsert
		// below. Lines are persisted explicitly right after this.
		if err := tx.Omit(clause.Associations).Clauses(clause.OnConflict{
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
		for i := range row.Lines {
			row.Lines[i].InvoiceID = row.ID
		}

		// Upsert by (invoice_id, line_number) instead of delete+reinsert, so line
		// IDs stay stable across re-imports — withholding_calculations references
		// them by invoice_line_id.
		if len(row.Lines) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "invoice_id"}, {Name: "line_number"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"sku", "description", "quantity", "unit_price", "line_total",
					"iva_rate", "iva_value", "concept_id", "classification_confidence", "updated_at",
				}),
			}).Create(&row.Lines).Error; err != nil {
				return fmt.Errorf("postgres: upserting invoice lines: %w", err)
			}
		}

		// Remove lines that no longer exist in this version of the invoice
		// (e.g. a corrected re-upload with fewer lines than before).
		keepIDs := make([]uint, len(row.Lines))
		for i := range row.Lines {
			keepIDs[i] = row.Lines[i].ID
		}
		q := tx.Where("invoice_id = ?", row.ID)
		if len(keepIDs) > 0 {
			q = q.Where("id NOT IN ?", keepIDs)
		}
		if err := q.Delete(&model.InvoiceLineModel{}).Error; err != nil {
			return fmt.Errorf("postgres: removing stale invoice lines: %w", err)
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
