package http

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/domain"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Middleware struct holds dependencies for middleware functions
type Middleware struct {
	Logger     hclog.Logger
	Validator  *domain.Validation
	corsConfig *CORSConfig
}

// CORSConfig holds configuration for CORS middleware
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	MaxAge           int  // Cache preflight requests
	AllowCredentials bool // Allow credentials like cookies
}

func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"},
		MaxAge:           86400, // 24 hours
		AllowCredentials: true,
	}
}

// NewMiddleware creates a new Middleware instance
func NewMiddleware(logger hclog.Logger, validator *domain.Validation, corsConfig *CORSConfig) *Middleware {
	if corsConfig == nil {
		corsConfig = DefaultCORSConfig()
	}
	return &Middleware{
		Logger:     logger,
		Validator:  validator,
		corsConfig: corsConfig,
	}
}

func (m *Middleware) CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if the origin is allowed
		allowed := false
		for _, allowedOrigin := range m.corsConfig.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				// Set the specific origin instead of wildcard for better security
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		if !allowed {
			// If origin is not allowed, still process the request but don't set CORS headers
			next.ServeHTTP(w, r)
			return
		}

		// Set standard CORS headers
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(m.corsConfig.AllowedMethods, ","))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(m.corsConfig.AllowedHeaders, ","))

		if m.corsConfig.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			// Set max age for preflight cache
			if m.corsConfig.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(m.corsConfig.MaxAge))
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ContentTypeMiddleware sets the Content-Type header to application/json
func (m *Middleware) ContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs the incoming requests and responses
func (m *Middleware) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := uuid.New().String()

		m.Logger.Info("Incoming request",
			"method", r.Method,
			"url", r.URL.Path,
			"request_id", requestID,
		)

		// Add the request ID to the response header
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r)

		duration := time.Since(start)
		m.Logger.Info("Completed request",
			"method", r.Method,
			"url", r.URL.Path,
			"request_id", requestID,
			"duration", duration,
		)
	})
}

// ValidationMiddleware validates the product in the request and adds it to the context
func (m *Middleware) ValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var product domain.Product
		err := json.NewDecoder(r.Body).Decode(&product)
		if err != nil {
			m.Logger.Error("Error decoding product", "error", err)
			http.Error(w, "Invalid product data", http.StatusBadRequest)
			return
		}

		errs := m.Validator.Validate(&product)
		if len(errs) > 0 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(errs)
			return
		}

		// Add the validated product to the context
		ctx := context.WithValue(r.Context(), ContextKeyProduct, &product)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
