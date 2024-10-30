package http

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/domain"
	websocketTransport "github.com/kahvecikaan/buildingMicroservices/product-api/internal/transport/websocket"
	"net/http"
	"path/filepath"
	"runtime"
)

func NewRouter(
	ph *ProductHandler,
	validator *domain.Validation,
	logger hclog.Logger,
	wsh *websocketTransport.Handler,
) *mux.Router {
	router := mux.NewRouter()

	// Create a middleware instance
	mw := NewMiddleware(logger, validator, nil) // nil for default CORS config

	// Apply global middleware
	router.Use(mw.LoggingMiddleware)
	router.Use(mw.CORSMiddleware)
	router.Use(mw.ContentTypeMiddleware)

	// Public routes (no authentication or validation needed)
	router.HandleFunc("/products", ph.GetProducts).Methods("GET")
	router.HandleFunc("/products/{id:[0-9]+}", ph.GetProductByID).Methods("GET")
	router.HandleFunc("/currencies", ph.ListCurrencies).Methods("GET")
	router.HandleFunc("/ws", wsh.HandleWebSocket).Methods("GET")

	// Routes requiring validation middleware (for request body validation)
	postRouter := router.Methods("POST").Subrouter()
	postRouter.HandleFunc("/products", ph.AddProduct)
	postRouter.Use(mw.ValidationMiddleware)

	putRouter := router.Methods("PUT").Subrouter()
	putRouter.HandleFunc("/products/{id:[0-9]+}", ph.UpdateProduct)
	putRouter.Use(mw.ValidationMiddleware)

	// Delete route (no request body, so validation middleware not needed)
	router.HandleFunc("/products/{id:[0-9]+}", ph.DeleteProduct).Methods("DELETE")

	// Swagger UI and specification routes
	// Determine the absolute path to the swagger.yaml file
	_, filename, _, _ := runtime.Caller(0)
	// filename is the path to this file (router.go)
	// Navigate to the root directory from the current file's location
	basePath := filepath.Dir(filename)                        // .../internal/transport/http
	rootDir := filepath.Join(basePath, "..", "..", "..")      // Navigate up to the root
	swaggerFilePath := filepath.Join(rootDir, "swagger.yaml") // .../product-api/swagger.yaml

	// Serve the swagger.yaml file
	router.HandleFunc("/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, swaggerFilePath)
	}).Methods("GET")

	// Configure the Redoc middleware to point to the correct SpecURL
	swaggerOpts := middleware.RedocOpts{SpecURL: "/swagger.yaml"}
	swaggerHandler := middleware.Redoc(swaggerOpts, nil)
	router.Handle("/docs", swaggerHandler).Methods("GET")

	// Return the configured router
	return router
}
