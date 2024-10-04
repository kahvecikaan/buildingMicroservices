package handlers

import (
	"context"
	"github.com/kahvecikaan/buildingMicroservices/product-api/data"
	"net/http"
)

// MiddlewareProductValidation validates the product in the request and adds it to the request context.
// Then, calls the next handler
func (p *Products) MiddlewareProductValidation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")

		prod := &data.Product{}

		err := data.FromJSON(prod, r.Body)
		if err != nil {
			p.l.Error("Error deserializing product", "error", err)
			http.Error(rw, "Error reading product", http.StatusBadRequest)
			return
		}

		// Validate the product
		errs := p.v.Validate(prod)
		if len(errs) != 0 {
			p.l.Error("Validation errors", "errors", errs.Errors())

			// return the validation messages as an array
			rw.WriteHeader(http.StatusUnprocessableEntity)
			data.ToJSON(&ValidationError{Messages: errs.Errors()}, rw)
			return
		}

		// Add the validated product to the context
		ctx := context.WithValue(r.Context(), KeyProduct{}, prod)
		req := r.WithContext(ctx)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, req)
	})
}
