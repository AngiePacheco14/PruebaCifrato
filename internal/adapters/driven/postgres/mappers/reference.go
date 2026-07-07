package mappers

import (
	"cifrato/internal/adapters/driven/postgres/models"
	"cifrato/internal/domain/withholding"
)

func CityToDomain(m *models.CityModel) *withholding.City {
	return &withholding.City{
		ID:         m.ID,
		Name:       m.Name,
		Department: m.Department,
	}
}

func CityToModel(c *withholding.City) *models.CityModel {
	return &models.CityModel{
		ID:         c.ID,
		Name:       c.Name,
		Department: c.Department,
	}
}

func UVTValueToDomain(m *models.UVTValueModel) *withholding.UVTValue {
	return &withholding.UVTValue{
		ID:                  m.ID,
		Year:                m.Year,
		Value:               m.Value,
		EffectiveFrom:       m.EffectiveFrom,
		EffectiveTo:         m.EffectiveTo,
		ResolutionReference: m.ResolutionReference,
	}
}

func UVTValueToModel(v *withholding.UVTValue) *models.UVTValueModel {
	return &models.UVTValueModel{
		ID:                  v.ID,
		Year:                v.Year,
		Value:               v.Value,
		EffectiveFrom:       v.EffectiveFrom,
		EffectiveTo:         v.EffectiveTo,
		ResolutionReference: v.ResolutionReference,
	}
}
