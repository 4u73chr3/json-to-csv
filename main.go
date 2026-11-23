package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"json-to-csv/config"
	"json-to-csv/handlers"
	"json-to-csv/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	apiClient := services.NewAPIClient(cfg.APIURL, cfg.APIParameter, cfg.APITimeout)
	handler, err := handlers.New(apiClient)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handler.Index)
	mux.HandleFunc("POST /export", handler.Export)
	mux.HandleFunc("GET /health", handler.Health)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handlers.SecurityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      cfg.APITimeout + 10*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("json-to-csv listening on http://localhost:%d", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
