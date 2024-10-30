// Package classification of Product API
//
// # Documentation for Product API
//
// Schemes: http
// BasePath: /
// Version: 1.0.0
//
// Consumes:
// - application/json
//
// Produces:
// - application/json
//
// swagger:meta
package http

import "github.com/kahvecikaan/buildingMicroservices/product-api/internal/domain"

// NOTE: Types defined here are purely for documentation purposes
// These types are not used by any of the handlers

// Generic error message returned as a string
// swagger:response errorResponse
type errorResponseWrapper struct {
	// Description of the error
	// in: body
	Body ErrorResponse
}

// Validation errors defined as an array of strings
// swagger:response validationErrorResponse
type validationErrorResponseWrapper struct {
	// Collection of the errors
	// in: body
	Body ValidationError
}

// A list of products
// swagger:response productsResponse
type productsResponseWrapper struct {
	// All current products
	// in: body
	Body []domain.Product
}

// Data structure representing a single product
// swagger:response productResponse
type productResponseWrapper struct {
	// A single product
	// in: body
	Body domain.Product
}

// No content response for endpoints that return 204
// swagger:response noContentResponse
type noContentResponseWrapper struct{}

// A list of currency codes
// swagger:response currenciesResponse
type currenciesResponseWrapper struct {
	// List of available currency codes
	// in: body
	Body []string
}

// swagger:parameters getProductByID deleteProduct updateProduct
type productIDParamsWrapper struct {
	// The ID of the product
	// in: path
	// required: true
	ID int `json:"id"`
}

// swagger:parameters addProduct updateProduct
type productBodyParamsWrapper struct {
	// Product data structure to create or update.
	// in: body
	// required: true
	Body domain.Product
}

// ErrorResponse defines the structure for API error responses
//
// swagger:model
type ErrorResponse struct {
	// The error message
	//
	// required: true
	Message string `json:"message"`
}

// ValidationError defines the structure for API validation error responses
//
// swagger:model
type ValidationError struct {
	// The validation errors
	//
	// required: true
	Messages []string `json:"messages"`
}
