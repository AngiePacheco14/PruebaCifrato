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

type CalculationRepository struct{ db *gorm.DB }

func NewCalculationRepository(db *gorm.DB) *CalculationRepository {
	return &CalculationRepository{db: db}
}

var _ repository.CalculationRepository = (*CalculationRepository)(nil)

func (r *CalculationRepository) Upsert(ctx context.Context, calc *entity.Calculation) error {
	row := mappers.CalculationToModel(calc)
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "invoice_line_id"}, {Name: "tax_type"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"concept_id", "base_amount", "tariff_applied", "calculated_value",
			"legal_basis", "justification", "updated_at",
		}),
	}).Create(row).Error
	if err != nil {
		return fmt.Errorf("postgres: upserting calculation: %w", err)
	}
	calc.ID = row.ID
	return nil
}

func (r *CalculationRepository) ListByInvoice(ctx context.Context, invoiceID uint) ([]entity.Calculation, error) {
	var rows []model.WithholdingCalculationModel
	if err := r.db.WithContext(ctx).Where("invoice_id = ?", invoiceID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("postgres: listing calculations by invoice: %w", err)
	}
	calcs := make([]entity.Calculation, len(rows))
	for i := range rows {
		calcs[i] = *mappers.CalculationToDomain(&rows[i])
	}
	return calcs, nil
}
