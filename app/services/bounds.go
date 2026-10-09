package services

import (
	"errors"
	"fmt"
	frameworkerrors "github.com/goravel/framework/errors"
)

var ErrNotFound = errors.New("record not found")
var ErrConflict = errors.New("operation conflicts with current state")

func IsInfrastructureError(err error) bool {
	return err != nil && !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrConflict)
}

// NormalizePagination bounds both allocations and offset arithmetic.
func NormalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if page > 1000000 {
		page = 1000000
	}
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func recordError(err error, id uint, resource string) error {
	if err != nil && !errors.Is(err, frameworkerrors.OrmRecordNotFound) {
		return fmt.Errorf("load %s: %w", resource, err)
	}
	if id == 0 || errors.Is(err, frameworkerrors.OrmRecordNotFound) {
		return fmt.Errorf("%s: %w", resource, ErrNotFound)
	}
	return nil
}
