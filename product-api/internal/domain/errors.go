package domain

import "errors"

// Domain-level errors
var (
	ErrProductNotFound = errors.New("product not found")
	ErrInvalidCurrency = errors.New("invalid currency")
)
