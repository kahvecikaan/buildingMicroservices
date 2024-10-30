package main

import (
	"context"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/currency/protos"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/domain"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/events"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/repository"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/service"
	httpTransport "github.com/kahvecikaan/buildingMicroservices/product-api/internal/transport/http"
	websocketTransport "github.com/kahvecikaan/buildingMicroservices/product-api/internal/transport/websocket"
	"github.com/nicholasjackson/env"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"os"
	"os/signal"
	"time"
)

// Environment variables
var (
	bindAddress = env.String("BIND_ADDRESS", false,
		":9090", "Bind address for the server")
	logLevel = env.String("LOG_LEVEL", false,
		"debug", "Log output level for the server [debug, info, trace]")
	grpcAddress = env.String("GRPC_ADDRESS", false,
		":9092", "Address of the gRPC currency service")
)

func main() {
	env.Parse()

	// Initialize the logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "product-api",
		Level: hclog.LevelFromString(*logLevel),
	})

	// Create a standard logger for the HTTP server
	standardLogger := logger.StandardLogger(&hclog.StandardLoggerOptions{InferLevels: true})

	// Initialize the event bus - this will be shared between services
	eventBus := events.NewEventBus[any]()

	// Set up the currency gRPC client
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	grpcConn, err := grpc.Dial(*grpcAddress, dialOpts...)
	if err != nil {
		logger.Error("Failed to connect to currency service", "error", err)
		os.Exit(1)
	}
	defer grpcConn.Close()

	currencyClient := protos.NewCurrencyClient(grpcConn)

	// Check if the currency service is available
	if err := checkCurrencyService(currencyClient); err != nil {
		logger.Error("Currency service is not available", "error", err)
		os.Exit(1)
	}

	// Initialize the CurrencyService with the event bus
	cs := service.NewCurrencyService(
		logger.Named("currency-service"),
		currencyClient,
		eventBus, // Pass the event bus here
	)
	defer cs.Close()

	// Initialize the ProductRepository
	prodRep := repository.NewMemoryProductRepository()

	// Initialize the ProductService with EventBus
	ps := service.NewProductService(
		prodRep,
		cs,
		eventBus,
		logger.Named("product-service"),
	)

	// Initialize the validator
	validator := domain.NewValidation()

	// Initialize HTTP handlers
	ph := httpTransport.NewProductHandler(ps, logger.Named("http-handler"))

	// Initialize the WebSocket handler with the event bus
	wh := websocketTransport.NewHandler(
		logger.Named("websocket-handler"),
		eventBus,
	)

	// Initialize the router
	router := httpTransport.NewRouter(ph, validator, logger, wh)

	// Create the HTTP Server
	server := &http.Server{
		Addr:         *bindAddress,
		Handler:      router,
		ErrorLog:     standardLogger,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start the server in a new goroutine
	go func() {
		logger.Info("Starting server", "bind_address", *bindAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Error starting server", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
	logger.Info("Shutting down server")

	// Context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Cleanup services
	if err := ps.Close(); err != nil {
		logger.Error("Error closing product service", "error", err)
	}

	if err := cs.Close(); err != nil {
		logger.Error("Error closing currency service", "error", err)
	}

	server.Shutdown(shutdownCtx)
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
