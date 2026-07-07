package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"cifrato/internal/adapters/driven/postgres/mappers"
	"cifrato/internal/adapters/driven/postgres/models"
	"cifrato/internal/application/ports/out"
)

type ClassificationCacheRepository struct{ db *gorm.DB }

func NewClassificationCacheRepository(db *gorm.DB) *ClassificationCacheRepository {
	return &ClassificationCacheRepository{db: db}
}

var _ out.ClassificationCacheRepository = (*ClassificationCacheRepository)(nil)

func (r *ClassificationCacheRepository) FindByIssuerAndSKU(ctx context.Context, issuerNIT, sku string) (*out.ClassificationCacheEntry, error) {
	var model models.LineClassificationModel
	q := r.db.WithContext(ctx).Where("issuer_nit = ? AND sku = ?", issuerNIT, sku)
	found, err := findOne(q, &model, "finding classification by issuer+sku")
	if err != nil || found == nil {
		return nil, err
	}
	return mappers.ClassificationEntryToDomain(found), nil
}

func (r *ClassificationCacheRepository) FindByDescription(ctx context.Context, descriptionNormalized string) (*out.ClassificationCacheEntry, error) {
	var model models.LineClassificationModel
	q := r.db.WithContext(ctx).Where("description_normalized = ?", descriptionNormalized)
	found, err := findOne(q, &model, "finding classification by description")
	if err != nil || found == nil {
		return nil, err
	}
	return mappers.ClassificationEntryToDomain(found), nil
}

func (r *ClassificationCacheRepository) Save(ctx context.Context, entry *out.ClassificationCacheEntry) error {
	model := mappers.ClassificationEntryToModel(entry)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("postgres: saving classification cache entry: %w", err)
	}
	entry.ID = model.ID
	return nil
}
