package handlers

import (
	"fmt"
	"github.com/kahvecikaan/buildingMicroservices/product-api/data"
	"net/http"
)

// swagger:route POST /products products createProduct
// Create a new product
//
// responses:
//	200: productResponse
//  422: errorValidation
//  501: errorResponse

// Create handles POST requests to add new products
func (p *Products) Create(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")

	// fetch the product from the context
	prod := r.Context().Value(KeyProduct{}).(*data.Product)

	p.productDB.AddProduct(prod)

	p.l.Debug("Inserting product", "product", fmt.Sprintf("%+v", prod))

	// Serialize the created product and send it back in the response
	err := data.ToJSON(prod, rw)
	if err != nil {
		p.l.Error("Error serializing product", "error", err)
	}
}
