package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type CityModel struct {
	ID         uint   `gorm:"primaryKey"`
	Name       string `gorm:"size:100;not null;uniqueIndex:idx_city_name_dept"`
	Department string `gorm:"size:100;not null;uniqueIndex:idx_city_name_dept"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (CityModel) TableName() string { return "cities" }

type UVTValueModel struct {
	ID                  uint            `gorm:"primaryKey"`
	Year                int             `gorm:"not null;uniqueIndex"`
	Value               decimal.Decimal `gorm:"type:numeric(12,2);not null"`
	EffectiveFrom       time.Time       `gorm:"not null;index"`
	EffectiveTo         *time.Time      `gorm:"index"`
	ResolutionReference string          `gorm:"type:text"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (UVTValueModel) TableName() string { return "uvt_values" }
