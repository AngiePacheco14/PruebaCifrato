package out

import (
	"context"
	"time"

	"cifrato/internal/domain/withholding"
)

type ReferenceDataRepository interface {
	FindConceptByCode(ctx context.Context, code string) (*withholding.Concept, error)
	ListConcepts(ctx context.Context) ([]withholding.Concept, error)
	FindCityByName(ctx context.Context, name string) (*withholding.City, error)
	FindUVTValue(ctx context.Context, at time.Time) (*withholding.UVTValue, error)
}
