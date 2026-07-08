package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/enums"
	"cifrato/internal/domain/repository"
	"cifrato/internal/infrastructure/adapters/repository/postgres/mappers"
	"cifrato/internal/infrastructure/adapters/repository/postgres/model"
)

type TaxRuleRepository struct{ db *gorm.DB }

func NewTaxRuleRepository(db *gorm.DB) *TaxRuleRepository { return &TaxRuleRepository{db: db} }

var _ repository.TaxRuleRepository = (*TaxRuleRepository)(nil)

func (r *TaxRuleRepository) FindApplicable(ctx context.Context, taxType enums.TaxType, conceptID uint, cityID *uint, at time.Time) (*entity.TaxRule, error) {
	q := r.db.WithContext(ctx).
		Where("tax_type = ? AND concept_id = ?", string(taxType), conceptID).
		Where("effective_from <= ?", at).
		Where("effective_to IS NULL OR effective_to >= ?", at)

	if cityID != nil {
		q = q.Where("city_id = ?", *cityID)
	} else {
		q = q.Where("city_id IS NULL")
	}

	var row model.AdditionalTaxRuleModel
	found, err := findOne(q.Order("effective_from DESC"), &row, "finding applicable tax rule")
	if err != nil || found == nil {
		return nil, err
	}
	return mappers.TaxRuleToDomain(found), nil
}

func (r *TaxRuleRepository) ListByTaxType(ctx context.Context, taxType enums.TaxType) ([]entity.TaxRule, error) {
	var rows []model.AdditionalTaxRuleModel
	if err := r.db.WithContext(ctx).Where("tax_type = ?", string(taxType)).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("postgres: listing tax rules: %w", err)
	}
	rules := make([]entity.TaxRule, len(rows))
	for i := range rows {
		rules[i] = *mappers.TaxRuleToDomain(&rows[i])
	}
	return rules, nil
}
