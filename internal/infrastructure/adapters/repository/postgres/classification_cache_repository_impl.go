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

type ClassificationCacheRepository struct{ db *gorm.DB }

func NewClassificationCacheRepository(db *gorm.DB) *ClassificationCacheRepository {
	return &ClassificationCacheRepository{db: db}
}

var _ repository.ClassificationCacheRepository = (*ClassificationCacheRepository)(nil)

func (r *ClassificationCacheRepository) FindByIssuerAndSKU(ctx context.Context, issuerNIT, sku string) (*entity.ClassificationCacheEntry, error) {
	var row model.LineClassificationModel
	q := r.db.WithContext(ctx).Where("issuer_nit = ? AND sku = ?", issuerNIT, sku)
	found, err := findOne(q, &row, "finding classification by issuer+sku")
	if err != nil || found == nil {
		return nil, err
	}
	return mappers.ClassificationEntryToDomain(found), nil
}

func (r *ClassificationCacheRepository) FindByDescription(ctx context.Context, descriptionNormalized string) (*entity.ClassificationCacheEntry, error) {
	var row model.LineClassificationModel
	q := r.db.WithContext(ctx).Where("description_normalized = ?", descriptionNormalized)
	found, err := findOne(q, &row, "finding classification by description")
	if err != nil || found == nil {
		return nil, err
	}
	return mappers.ClassificationEntryToDomain(found), nil
}

// Save upserts on (issuer_nit, sku) to avoid a unique-index violation when
// two concurrent lookups miss the cache for the same key. Rows with a NULL
// issuer_nit/sku are excluded from that index and always insert as new.
func (r *ClassificationCacheRepository) Save(ctx context.Context, entry *entity.ClassificationCacheEntry) error {
	row := mappers.ClassificationEntryToModel(entry)
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "issuer_nit"}, {Name: "sku"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"concept_id", "confidence", "model_version", "reasoning",
		}),
	}).Create(row).Error
	if err != nil {
		return fmt.Errorf("postgres: saving classification cache entry: %w", err)
	}
	entry.ID = row.ID
	return nil
}
