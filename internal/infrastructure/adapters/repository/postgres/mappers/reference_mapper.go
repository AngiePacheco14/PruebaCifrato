package mappers

import (
	"cifrato/internal/domain/entity"
	"cifrato/internal/infrastructure/adapters/repository/postgres/model"
)

func CityToDomain(m *model.CityModel) *entity.City {
	return &entity.City{
		ID:         m.ID,
		Name:       m.Name,
		Department: m.Department,
	}
}

func CityToModel(c *entity.City) *model.CityModel {
	return &model.CityModel{
		ID:         c.ID,
		Name:       c.Name,
		Department: c.Department,
	}
}

func UVTValueToDomain(m *model.UVTValueModel) *entity.UVTValue {
	return &entity.UVTValue{
		ID:                  m.ID,
		Year:                m.Year,
		Value:               m.Value,
		EffectiveFrom:       m.EffectiveFrom,
		EffectiveTo:         m.EffectiveTo,
		ResolutionReference: m.ResolutionReference,
	}
}

func UVTValueToModel(v *entity.UVTValue) *model.UVTValueModel {
	return &model.UVTValueModel{
		ID:                  v.ID,
		Year:                v.Year,
		Value:               v.Value,
		EffectiveFrom:       v.EffectiveFrom,
		EffectiveTo:         v.EffectiveTo,
		ResolutionReference: v.ResolutionReference,
	}
}
