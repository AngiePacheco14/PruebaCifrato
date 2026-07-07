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

type ReferenceDataRepository struct{ db *gorm.DB }

func NewReferenceDataRepository(db *gorm.DB) *ReferenceDataRepository {
	return &ReferenceDataRepository{db: db}
}

var _ out.ReferenceDataRepository = (*ReferenceDataRepository)(nil)

func (r *ReferenceDataRepository) FindConceptByCode(ctx context.Context, code string) (*withholding.Concept, error) {
	var model models.WithholdingConceptModel
	found, err := findOne(r.db.WithContext(ctx).Where("code = ?", code), &model, "finding concept by code")
	if err != nil || found == nil {
		return nil, err
	}
	return mappers.ConceptToDomain(found), nil
}

func (r *ReferenceDataRepository) ListConcepts(ctx context.Context) ([]withholding.Concept, error) {
	var rows []models.WithholdingConceptModel
	if err := r.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("postgres: listing concepts: %w", err)
	}
	concepts := make([]withholding.Concept, len(rows))
	for i := range rows {
		concepts[i] = *mappers.ConceptToDomain(&rows[i])
	}
	return concepts, nil
}

func (r *ReferenceDataRepository) FindCityByName(ctx context.Context, name string) (*withholding.City, error) {
	var model models.CityModel
	found, err := findOne(r.db.WithContext(ctx).Where("name = ?", name), &model, "finding city by name")
	if err != nil || found == nil {
		return nil, err
	}
	return mappers.CityToDomain(found), nil
}

func (r *ReferenceDataRepository) FindUVTValue(ctx context.Context, at time.Time) (*withholding.UVTValue, error) {
	var model models.UVTValueModel
	q := r.db.WithContext(ctx).
		Where("effective_from <= ?", at).
		Where("effective_to IS NULL OR effective_to >= ?", at).
		Order("effective_from DESC")
	found, err := findOne(q, &model, "finding uvt value")
	if err != nil || found == nil {
		return nil, err
	}
	return mappers.UVTValueToDomain(found), nil
}
