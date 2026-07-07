package mappers

import (
	"cifrato/internal/adapters/driven/postgres/models"
	"cifrato/internal/application/ports/out"
)

func ClassificationEntryToDomain(m *models.LineClassificationModel) *out.ClassificationCacheEntry {
	return &out.ClassificationCacheEntry{
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

func ClassificationEntryToModel(e *out.ClassificationCacheEntry) *models.LineClassificationModel {
	return &models.LineClassificationModel{
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
