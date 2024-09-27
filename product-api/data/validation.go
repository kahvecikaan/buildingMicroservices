package data

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"regexp"
)

// ValidationError wraps the validator.FieldError type
// so, we don't expose this to our users
type ValidationError struct {
	validator.FieldError
}

func (v ValidationError) Error() string {
	return fmt.Sprintf(
		"Key: '%s' Error: Field validation for '%s' failed on the '%s' tag",
		v.Namespace(),
		v.Field(),
		v.Tag(),
	)
}

// ValidationErrors is a collection of ValidationError
type ValidationErrors []ValidationError

// Errors converts the slice of FieldError's to a slice of strings
func (ve ValidationErrors) Errors() []string {
	errs := []string{}
	for _, v := range ve {
		errs = append(errs, v.Error())
	}

	return errs
}

type Validation struct {
	validate *validator.Validate
}

// NewValidation creates a new Validation type
func NewValidation() *Validation {
	validate := validator.New()
	validate.RegisterValidation("sku", validateSKU)

	return &Validation{validate}
}

// Validate the item
func (v *Validation) Validate(i interface{}) ValidationErrors {
	errs := v.validate.Struct(i).(validator.ValidationErrors)

	if len(errs) == 0 {
		return nil
	}

	var returnErrs []ValidationError
	for _, err := range errs {
		// cast the FieldError into our ValidationError and append to the slice
		ve := ValidationError{err.(validator.FieldError)}
		returnErrs = append(returnErrs, ve)
	}

	return returnErrs
}

// validateSKU is a custom validator for SKU
func validateSKU(fl validator.FieldLevel) bool {
	re := regexp.MustCompile(`^[a-z]+-[a-z]+-[a-z]+$`)
	matches := re.FindAllString(fl.Field().String(), -1)
	return len(matches) == 1
}
