package handlers

import (
	"github.com/kahvecikaan/buildingMicroservices/product-api/data"
	"net/http"
)

// swagger:route PUT /products products updateProduct
// Update a products details
//
// responses:
//
//	204: noContentResponse
//	404: errorResponse
//	422: errorValidation

// Update handles PUT requests to update products
func (p *Products) Update(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")

	// fetch the product from the context
	prod := r.Context().Value(KeyProduct{}).(*data.Product)
	p.l.Debug("Updating record id", "id", prod.ID)

	err := p.productDB.UpdateProduct(prod)
	if err == data.ErrProductNotFound {
		p.l.Error("Product not found", "id", prod.ID, "error", err)

		rw.WriteHeader(http.StatusNotFound)
		data.ToJSON(&GenericError{Message: "Product not found in database"}, rw)
		return
	}

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	// write the no content success header
	rw.WriteHeader(http.StatusNoContent)
}
