package mappers

import (
	"cifrato/internal/domain/entity"
	"cifrato/internal/infrastructure/adapters/repository/postgres/model"
)

func ClassificationEntryToDomain(m *model.LineClassificationModel) *entity.ClassificationCacheEntry {
	return &entity.ClassificationCacheEntry{
		ID:                    m.ID,
		IssuerNIT:             m.IssuerNIT,
		SKU:                   m.SKU,
		DescriptionNormalized: m.DescriptionNormalized,
		ConceptID:             m.ConceptID,
		Confidence:            m.Confidence,
		ModelVersion:          m.ModelVersion,
		Reasoning:             m.Reasoning,
	}
}

func ClassificationEntryToModel(e *entity.ClassificationCacheEntry) *model.LineClassificationModel {
	return &model.LineClassificationModel{
		ID:                    e.ID,
		IssuerNIT:             e.IssuerNIT,
		SKU:                   e.SKU,
		DescriptionNormalized: e.DescriptionNormalized,
		ConceptID:             e.ConceptID,
		Confidence:            e.Confidence,
		ModelVersion:          e.ModelVersion,
		Reasoning:             e.Reasoning,
	}
}
