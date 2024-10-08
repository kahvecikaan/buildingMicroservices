package main

import (
	"context"
	gohandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/product-images/files"
	"github.com/kahvecikaan/buildingMicroservices/product-images/handlers"
	"github.com/nicholasjackson/env"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var bindAddress = env.String("BIND_ADDRESS", false,
	":9091", "Bind address for the server")
var logLevel = env.String("LOG_LEVEL", false,
	"debug", "Log output level for the server [debug, info, trace]")
var basePath = env.String("BASE_PATH", false,
	"./imagestore", "Base path to save images")

func main() {
	env.Parse()

	l := hclog.New(
		&hclog.LoggerOptions{
			Name:  "product-images",
			Level: hclog.LevelFromString(*logLevel),
		})

	// create a logger for the server from the default logger
	sl := l.StandardLogger(&hclog.StandardLoggerOptions{InferLevels: true})

	// create the storage class, use local storage
	// max filesize 5MB
	stor, err := files.NewLocal(*basePath, 1024*1024*5)
	if err != nil {
		l.Error("Unable to create the storage", "error", err)
		os.Exit(1)
	}

	// create the handlers
	fh := handlers.NewFiles(l, stor)
	gzipHandler := handlers.GzipHandler{}

	// create a new serve mux and register the handlers
	sm := mux.NewRouter()

	// Filename regex: {filename:[a-zA-Z]+\\.[a-z]{3}}
	route := "/images/{id:[0-9]+}/{filename:[a-zA-Z]+\\.[a-z]{3}}"

	// Use the same handler for both POST and GET methods
	// sm.HandleFunc(route, fh.ServeHTTP).Methods(http.MethodPost, http.MethodGet)

	// upload files
	postRouter := sm.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc(route, fh.UploadREST)
	postRouter.HandleFunc("/images", fh.UploadMultipart)
	postRouter.Use(gzipHandler.GzipMiddleware)

	// get files
	getRouter := sm.Methods(http.MethodGet).Subrouter()
	getRouter.HandleFunc(route, fh.GetFile)
	getRouter.Use(gzipHandler.GzipMiddleware)

	//ph := sm.Methods(http.MethodPost).Subrouter()
	// ph.HandleFunc("/images/{id:[0-9]+}/{filename:[a-zA-Z]+\\.[a-z]{3}}", fh.ServeHTTP)

	// get files (previous implementation)
	// gh := sm.Methods(http.MethodGet).Subrouter()
	// gh.Handle(
	//	"/images/{id:[0-9]+}/{filename:[a-zA-Z]+\\.[a-z]{3}}",
	//	http.StripPrefix("/images/", http.FileServer(http.Dir(*basePath))),
	//)

	// CORS middleware
	corsHandler := gohandlers.CORS(
		gohandlers.AllowedOrigins([]string{"http://localhost:3000"}),
		gohandlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"}),
		gohandlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	// Apply CORS middleware to our router
	corsRouter := corsHandler(sm)

	// create a new server
	s := http.Server{
		Addr:         *bindAddress,      // configure the bind address
		Handler:      corsRouter,        // use the CORS-enabled router
		ErrorLog:     sl,                // the logger for the server
		ReadTimeout:  5 * time.Second,   // max time to read request from the client
		WriteTimeout: 10 * time.Second,  // max time to write response to the client
		IdleTimeout:  120 * time.Second, // max time for connections using TCP Keep-Alive
	}

	// start the server
	go func() {
		l.Info("Starting server", "bind_address", *bindAddress)

		err := s.ListenAndServe()
		if err != nil {
			l.Error("Unable to start the server", "error", err)
			os.Exit(1)
		}
	}()

	// trap sigterm or interrupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)

	// block until a signal is received
	sig := <-c
	l.Info("Shutting down the server with", "signal", sig)

	// gracefully shutdown the server, waiting max 30 seconds for current operations to complete
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	s.Shutdown(ctx)
}
