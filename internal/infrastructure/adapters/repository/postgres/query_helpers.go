package postgres

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// findOne runs q against dest, translating gorm.ErrRecordNotFound into
// (nil, nil) instead of an error. errCtx labels the wrapped error message.
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
