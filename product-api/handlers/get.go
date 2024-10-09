package handlers

import (
	"github.com/kahvecikaan/buildingMicroservices/product-api/data"
	"net/http"
)

// swagger:route GET /products products listProducts
// Returns a list of products
// responses:
//
//	200: productsResponse

// ListAll handles GET requests and returns all current products
func (p *Products) ListAll(rw http.ResponseWriter, r *http.Request) {
	p.l.Debug("Get all product records")

	rw.Header().Set("Content-Type", "application/json")

	cur := r.URL.Query().Get("currency")

	prods, err := p.productDB.GetProducts(cur)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		data.ToJSON(&GenericError{Message: err.Error()}, rw)
		return
	}

	err = data.ToJSON(prods, rw)
	if err != nil {
		// we should never be here but log the error just in case
		p.l.Error("Error serializing product list", "error", err)
	}
}

// swagger:route GET /products/{id} products listSingleProduct
// Returns a single product from the database
// responses:
//
//	200: productResponse
//	404: errorResponse

// ListSingle handles GET requests to retrieve a single product
func (p *Products) ListSingle(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")

	id := getProductID(r)
	cur := r.URL.Query().Get("currency")

	p.l.Debug("Get product by ID", "id", id)

	prod, err := p.productDB.GetProductByID(id, cur)

	switch err {
	case nil:

	case data.ErrProductNotFound:
		p.l.Error("Product not found", "id", id, "error", err)

		rw.WriteHeader(http.StatusNotFound)
		data.ToJSON(&GenericError{Message: err.Error()}, rw)
		return
	default:
		p.l.Error("Error fetching product", "id", id, "error", err)

		rw.WriteHeader(http.StatusInternalServerError)
		data.ToJSON(&GenericError{Message: err.Error()}, rw)
		return
	}

	err = data.ToJSON(prod, rw)
	if err != nil {
		// we should never be here but log the error just in case
		p.l.Error("Error serializing product", "id", id, "error", err)
	}
}
