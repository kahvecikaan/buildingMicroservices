package domain

// Product represents the product model
//
// swagger:model
type Product struct {
	// The ID of the product
	//
	// required: true
	// min: 1
	// example: 1
	ID int `json:"id"`

	// The name of the product
	//
	// required: true
	// example: Coffee
	Name string `json:"name" validate:"required"`

	// The description of the product
	//
	// required: false
	// example: Freshly brewed coffee
	Description string `json:"description"`

	// The price of the product
	//
	// required: true
	// min: 0.01
	// example: 2.99
	Price float64 `json:"price" validate:"required,gt=0"`

	// The SKU of the product in the format abc-abc-abc
	//
	// required: true
	// pattern: '^[a-zA-Z]{3}-[a-zA-Z]{3}-[a-zA-Z]{3}$'
	// example: abc-def-ghi
	SKU string `json:"sku" validate:"required,sku"`
}
