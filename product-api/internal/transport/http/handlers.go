package http

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/domain"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/service"
	"net/http"
	"strconv"
)

type ProductHandler struct {
	productService service.ProductService
	logger         hclog.Logger
}

func NewProductHandler(ps service.ProductService, log hclog.Logger) *ProductHandler {
	return &ProductHandler{
		productService: ps,
		logger:         log,
	}
}

// GetProducts handles GET /products
//
// swagger:route GET /products products listProducts
//
// Returns a list of products.
//
// Responses:
//
//	200: productsResponse
//	500: errorResponse
func (h *ProductHandler) GetProducts(w http.ResponseWriter, r *http.Request) {
	currency := r.URL.Query().Get("currency")

	products, err := h.productService.GetProducts(r.Context(), currency)
	if err != nil {
		h.logger.Error("Error getting products", "error", err)
		http.Error(w, "Error getting products", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application-json")
	json.NewEncoder(w).Encode(products)
}

// GetProductByID handles GET /products/{id}
//
// swagger:route GET /products/{id} products getProductByID
//
// Returns a product by ID.
//
// Responses:
//
//	200: productResponse
//	400: errorResponse
//	404: errorResponse
func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	currency := r.URL.Query().Get("currency")

	product, err := h.productService.GetProductByID(r.Context(), id, currency)
	if err != nil {
		if err == domain.ErrProductNotFound {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}

		h.logger.Error("Error getting product", "error", err)
		http.Error(w, "Error getting product", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

// AddProduct handles POST /products
//
// swagger:route POST /products products addProduct
//
// Adds a new product.
//
// Responses:
//
//	201: productResponse
//	400: validationErrorResponse
//	500: errorResponse
func (h *ProductHandler) AddProduct(w http.ResponseWriter, r *http.Request) {
	// Retrieve the validated product from the context
	product, ok := r.Context().Value(ContextKeyProduct).(*domain.Product)
	if !ok {
		http.Error(w, "Invalid product data", http.StatusBadRequest)
		return
	}

	err := h.productService.AddProduct(r.Context(), product)
	if err != nil {
		h.logger.Error("Error adding product", "error", err)
		http.Error(w, "Error adding product", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// UpdateProduct handles PUT /products/{id}
//
// swagger:route PUT /products/{id} products updateProduct
//
// Updates an existing product.
//
// Responses:
//
//	204: noContentResponse
//	400: validationErrorResponse
//	404: errorResponse
//	500: errorResponse
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	// Retrieve the validated product from the context
	product, ok := r.Context().Value(ContextKeyProduct).(*domain.Product)
	if !ok {
		http.Error(w, "Invalid product data", http.StatusBadRequest)
		return
	}

	product.ID = id

	err = h.productService.UpdateProduct(r.Context(), product)
	if err != nil {
		if err == domain.ErrProductNotFound {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Error updating product", "error", err)
		http.Error(w, "Error updating product", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteProduct handles DELETE /products/{id}
//
// swagger:route DELETE /products/{id} products deleteProduct
//
// Deletes a product.
//
// Responses:
//
//	204: noContentResponse
//	404: errorResponse
//	500: errorResponse
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	err = h.productService.DeleteProduct(r.Context(), id)
	if err != nil {
		if err == domain.ErrProductNotFound {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Error deleting product", "error", err)
		http.Error(w, "Error deleting product", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListCurrencies handles GET /currencies
//
// swagger:route GET /currencies currencies listCurrencies
//
// Returns a list of available currency codes.
//
// Responses:
//
//	200: currenciesResponse
//	500: errorResponse
func (h *ProductHandler) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	currencies, err := h.productService.ListCurrencies(r.Context())
	if err != nil {
		h.logger.Error("Error listing currencies", "error", err)
		return
	}

	json.NewEncoder(w).Encode(currencies)
}
