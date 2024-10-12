package main

import (
	"context"
	"github.com/go-openapi/runtime/middleware"
	gohandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/currency/protos"
	"github.com/kahvecikaan/buildingMicroservices/product-api/data"
	"github.com/kahvecikaan/buildingMicroservices/product-api/handlers"
	"github.com/nicholasjackson/env"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var (
	bindAddress = env.String("BIND_ADDRESS", false, ":9090", "Bind address for the server")
	logLevel    = env.String("LOG_LEVEL", false, "debug", "Log output level for the server [debug, info, trace]")
)

func main() {
	env.Parse()

	l := hclog.New(&hclog.LoggerOptions{
		Name:  "product-api",
		Level: hclog.LevelFromString(*logLevel),
	})

	// create a standard logger for the HTTP server
	sl := l.StandardLogger(&hclog.StandardLoggerOptions{InferLevels: true})
	v := data.NewValidation()

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	currencyServiceAddress := "localhost:9092"
	// create the gRPC client connection
	conn, err := grpc.NewClient(currencyServiceAddress, dialOpts...)
	if err != nil {
		l.Error("failed to connect to currency service", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	// create currency client
	currencyClient := protos.NewCurrencyClient(conn)

	// check if the currency service is available
	if err := checkCurrencyService(currencyClient); err != nil {
		l.Error("Currency service is not available", "error", err)
		os.Exit(1)
	}

	// create a new database instance
	db := data.NewProductsDB(l, currencyClient)

	ph := handlers.NewProducts(l, v, db)

	// create a new router
	sm := mux.NewRouter()

	// handlers for API
	getRouter := sm.Methods(http.MethodGet).Subrouter()
	getRouter.HandleFunc("/products", ph.ListAll).Queries("currency", "{[A-Z]}{3}}")
	getRouter.HandleFunc("/products", ph.ListAll)

	getRouter.HandleFunc("/products/{id:[0-9]+}", ph.ListSingle).Queries("currency", "{[A-Z]}{3}}")
	getRouter.HandleFunc("/products/{id:[0-9]+}", ph.ListSingle)

	putRouter := sm.Methods(http.MethodPut).Subrouter()
	putRouter.HandleFunc("/products", ph.Update)
	putRouter.Use(ph.MiddlewareProductValidation)

	postRouter := sm.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/products", ph.Create)
	postRouter.Use(ph.MiddlewareProductValidation)

	deleteRouter := sm.Methods(http.MethodDelete).Subrouter()
	deleteRouter.HandleFunc("/products/{id:[0-9]+}", ph.Delete)

	// Handler for documentation
	opts := middleware.RedocOpts{SpecURL: "/swagger.yaml"}
	sh := middleware.Redoc(opts, nil)

	getRouter.Handle("/docs", sh)
	getRouter.Handle("/swagger.yaml", http.FileServer(http.Dir("./")))

	// CORS middleware
	corsHandler := gohandlers.CORS(
		gohandlers.AllowedOrigins([]string{"http://localhost:3000"}),
		gohandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		gohandlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
	)

	// apply CORS middleware to our router
	corsRouter := corsHandler(sm)

	// create a new server
	s := &http.Server{
		Addr:         *bindAddress,      // Configure the bind address
		Handler:      corsRouter,        // Use the CORS-enabled router
		ErrorLog:     sl,                // Set the logger for the server
		ReadTimeout:  5 * time.Second,   // Max time to read request from the client
		WriteTimeout: 10 * time.Second,  // Max time to write response to the client
		IdleTimeout:  120 * time.Second, // Max time for connections using TCP Keep-Alive
	}

	// start the server
	go func() {
		l.Info("Starting server", "bind_address", *bindAddress)

		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			l.Error("Error starting server", "error", err)
			os.Exit(1)
		}
	}()

	// trap sigterm or interrupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)

	// Block until a signal is received
	sig := <-c
	l.Info("Shutting down the server", "signal", sig)

	// gracefully shutdown the server, waiting max 30 seconds for current operations to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s.Shutdown(ctx)
}

func checkCurrencyService(client protos.CurrencyClient) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.GetRate(ctx, &protos.RateRequest{
		Base:        protos.Currencies_EUR,
		Destination: protos.Currencies_USD,
	})

	return err
}
