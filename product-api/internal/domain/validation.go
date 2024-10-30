package domain

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"regexp"
)

type Validation struct {
	validator *validator.Validate
}

func NewValidation() *Validation {
	v := validator.New()
	v.RegisterValidation("sku", validateSKU)
	return &Validation{validator: v}
}

func validateSKU(fl validator.FieldLevel) bool {
	// SKU must be in format abc-abc-abc
	re := regexp.MustCompile(`^[a-z]+-[a-z]+-[a-z]+$`)
	return re.MatchString(fl.Field().String())
}

// ValidationError wraps the validator's FieldError
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface
func (v ValidationError) Error() string {
	return fmt.Sprintf("Field '%s': %s", v.Field, v.Message)
}

// ValidationErrors is a slice of ValidationError
type ValidationErrors []ValidationError

func (v *Validation) Validate(i interface{}) ValidationErrors {
	var errors ValidationErrors

	err := v.validator.Struct(i)
	if err != nil {
		validationErrors := err.(validator.ValidationErrors)
		for _, ve := range validationErrors {
			errors = append(errors, ValidationError{
				Field:   ve.Field(),
				Message: fmt.Sprintf("failed on the '%s' tag", ve.Tag()),
			})
		}
	}

	return errors
}
