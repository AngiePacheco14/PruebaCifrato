package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"cifrato/internal/adapters/driven/postgres/mappers"
	"cifrato/internal/adapters/driven/postgres/models"
	"cifrato/internal/application/ports/out"
	"cifrato/internal/domain/withholding"
)

type CalculationRepository struct{ db *gorm.DB }

func NewCalculationRepository(db *gorm.DB) *CalculationRepository {
	return &CalculationRepository{db: db}
}

var _ out.CalculationRepository = (*CalculationRepository)(nil)

func (r *CalculationRepository) Upsert(ctx context.Context, calc *withholding.Calculation) error {
	model := mappers.CalculationToModel(calc)
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "invoice_line_id"}, {Name: "tax_type"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"concept_id", "base_amount", "tariff_applied", "calculated_value",
			"legal_basis", "justification", "updated_at",
		}),
	}).Create(model).Error
	if err != nil {
		return fmt.Errorf("postgres: upserting calculation: %w", err)
	}
	calc.ID = model.ID
	return nil
}

func (r *CalculationRepository) ListByInvoice(ctx context.Context, invoiceID uint) ([]withholding.Calculation, error) {
	var rows []models.WithholdingCalculationModel
	if err := r.db.WithContext(ctx).Where("invoice_id = ?", invoiceID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("postgres: listing calculations by invoice: %w", err)
	}
	calcs := make([]withholding.Calculation, len(rows))
	for i := range rows {
		calcs[i] = *mappers.CalculationToDomain(&rows[i])
	}
	return calcs, nil
}
