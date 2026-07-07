package repositories

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"cifrato/internal/adapters/driven/postgres/mappers"
	"cifrato/internal/adapters/driven/postgres/models"
	"cifrato/internal/application/ports/out"
	"cifrato/internal/domain/withholding"
)

type TaxRuleRepository struct{ db *gorm.DB }

func NewTaxRuleRepository(db *gorm.DB) *TaxRuleRepository { return &TaxRuleRepository{db: db} }

var _ out.TaxRuleRepository = (*TaxRuleRepository)(nil)

func (r *TaxRuleRepository) FindApplicable(ctx context.Context, taxType withholding.TaxType, conceptID uint, cityID *uint, at time.Time) (*withholding.TaxRule, error) {
	q := r.db.WithContext(ctx).
		Where("tax_type = ? AND concept_id = ?", string(taxType), conceptID).
		Where("effective_from <= ?", at).
		Where("effective_to IS NULL OR effective_to >= ?", at)

	if cityID != nil {
		q = q.Where("city_id = ?", *cityID)
	} else {
		q = q.Where("city_id IS NULL")
	}

	var model models.AdditionalTaxRuleModel
	found, err := findOne(q.Order("effective_from DESC"), &model, "finding applicable tax rule")
	if err != nil || found == nil {
		return nil, err
	}
	return mappers.TaxRuleToDomain(found), nil
}

func (r *TaxRuleRepository) ListByTaxType(ctx context.Context, taxType withholding.TaxType) ([]withholding.TaxRule, error) {
	var rows []models.AdditionalTaxRuleModel
	if err := r.db.WithContext(ctx).Where("tax_type = ?", string(taxType)).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("postgres: listing tax rules: %w", err)
	}
	rules := make([]withholding.TaxRule, len(rows))
	for i := range rows {
		rules[i] = *mappers.TaxRuleToDomain(&rows[i])
	}
	return rules, nil
}
