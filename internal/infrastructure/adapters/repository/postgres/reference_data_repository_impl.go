package postgres

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"gorm.io/gorm"

	"cifrato/internal/domain/entity"
	"cifrato/internal/domain/repository"
	"cifrato/internal/infrastructure/adapters/repository/postgres/mappers"
	"cifrato/internal/infrastructure/adapters/repository/postgres/model"
)

type ReferenceDataRepository struct {
	db *gorm.DB

	// conceptsCache holds the withholding-concept catalog: small, seeded
	// reference data that doesn't change at runtime. ListConcepts is called
	// once per invoice during batch processing (up to 5 concurrently), so
	// caching avoids redundant round-trips for data that's already static.
	conceptsMu    sync.RWMutex
	conceptsCache []entity.Concept
}

func NewReferenceDataRepository(db *gorm.DB) *ReferenceDataRepository {
	return &ReferenceDataRepository{db: db}
}

var _ repository.ReferenceDataRepository = (*ReferenceDataRepository)(nil)

func (r *ReferenceDataRepository) FindConceptByCode(ctx context.Context, code string) (*entity.Concept, error) {
	var row model.WithholdingConceptModel
	found, err := findOne(r.db.WithContext(ctx).Where("code = ?", code), &row, "finding concept by code")
	if err != nil || found == nil {
		return nil, err
	}
	return mappers.ConceptToDomain(found), nil
}

func (r *ReferenceDataRepository) ListConcepts(ctx context.Context) ([]entity.Concept, error) {
	r.conceptsMu.RLock()
	cached := r.conceptsCache
	r.conceptsMu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	var rows []model.WithholdingConceptModel
	if err := r.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("postgres: listing concepts: %w", err)
	}
	concepts := make([]entity.Concept, len(rows))
	for i := range rows {
		concepts[i] = *mappers.ConceptToDomain(&rows[i])
	}

	r.conceptsMu.Lock()
	r.conceptsCache = concepts
	r.conceptsMu.Unlock()
	return concepts, nil
}

// FindCityByName matches loosely (normalized + bidirectional substring),
// since issuer XML city names rarely match the seeded canonical form exactly.
// Known risk: substring matching can false-positive between distinct cities
// sharing a prefix, e.g. "Girardot" vs the seeded "Girardota".
func (r *ReferenceDataRepository) FindCityByName(ctx context.Context, name string) (*entity.City, error) {
	var rows []model.CityModel
	if err := r.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("postgres: listing cities: %w", err)
	}
	target := normalizeCityName(name)
	if target == "" {
		return nil, nil
	}
	for i := range rows {
		candidate := normalizeCityName(rows[i].Name)
		if candidate == target || strings.Contains(target, candidate) || strings.Contains(candidate, target) {
			return mappers.CityToDomain(&rows[i]), nil
		}
	}
	return nil, nil
}

var accentFold = strings.NewReplacer(
	"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ñ", "N", "Ü", "U",
)

// normalizeCityName uppercases, folds Spanish accents, and drops everything
// that isn't a letter or digit — so punctuation/spacing differences
// ("Bogotá, D.C." vs "BOGOTA D.C") don't prevent a match.
func normalizeCityName(s string) string {
	s = accentFold.Replace(strings.ToUpper(strings.TrimSpace(s)))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (r *ReferenceDataRepository) FindUVTValue(ctx context.Context, at time.Time) (*entity.UVTValue, error) {
	var row model.UVTValueModel
	q := r.db.WithContext(ctx).
		Where("effective_from <= ?", at).
		Where("effective_to IS NULL OR effective_to >= ?", at).
		Order("effective_from DESC")
	found, err := findOne(q, &row, "finding uvt value")
	if err != nil || found == nil {
		return nil, err
	}
	return mappers.UVTValueToDomain(found), nil
}
