package repositories

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// findOne runs the given query against dest and translates
// gorm.ErrRecordNotFound into a nil result instead of an error — the
// "not found means nil, nil" shape every FindByX repository method needs.
// errCtx labels the wrapped error message on any other failure.
func findOne[T any](q *gorm.DB, dest *T, errCtx string) (*T, error) {
	err := q.First(dest).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: %s: %w", errCtx, err)
	}
	return dest, nil
}
