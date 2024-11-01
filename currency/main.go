package main

import (
	"context"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/currency/data"
	"github.com/kahvecikaan/buildingMicroservices/currency/protos"
	"github.com/kahvecikaan/buildingMicroservices/currency/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Initialize Logger
	log := hclog.New(&hclog.LoggerOptions{
		Name:  "CurrencyService",
		Color: hclog.AutoColor,
		Level: hclog.LevelFromString("debug"),
	})

	// Initialize ExchangeRates
	rates, err := data.NewRates(log)
	if err != nil {
		log.Error("Unable to generate rates", "error", err)
		os.Exit(1)
	}

	// Create Currency server instance
	currencyServer := server.NewCurrency(log, rates)

	// Create a new gRPC server
	gs := grpc.NewServer()

	// Register the Currency server with the gRPC server
	protos.RegisterCurrencyServer(gs, currencyServer)

	// Register the reflection service for debugging and introspection
	reflection.Register(gs)

	// Create a TCP listener on port 9092
	lis, err := net.Listen("tcp", ":9092")
	if err != nil {
		log.Error("Unable to create listener", "error", err)
		os.Exit(1)
	}

	// Start the gRPC server in a separate goroutine
	go func() {
		log.Info("Currency gRPC server is running on port :9092")
		if err := gs.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Error("Failed to serve gRPC server", "error", err)
			os.Exit(1)
		}
	}()

	// Channel to listen for OS signals
	sigChan := make(chan os.Signal, 1)
	// Notify on Interrupt and Terminate signals
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-sigChan
	log.Info("Received signal, initiating graceful shutdown", "signal", sig)

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Channel to signal that shutdown is complete
	doneChan := make(chan struct{})

	// Start a goroutine to perform the shutdown
	go func() {
		// Stop accepting new connections and gracefully shutdown gRPC server
		gs.GracefulStop()

		// Call Close on Currency server to terminate goroutines and release resources
		currencyServer.Close()

		// Signal that shutdown is complete
		close(doneChan)
	}()

	// Wait for either shutdown to complete or timeout
	select {
	case <-doneChan:
		log.Info("Graceful shutdown completed successfully")
	case <-ctx.Done():
		log.Warn("Graceful shutdown timed out, forcing exit")
	}

	log.Info("Shutdown complete")
}
