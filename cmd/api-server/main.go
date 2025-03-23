package main

import (
	"log"
	"net/http"

	"github.com/ARtorias742/DCTP/internal/api"
	"github.com/ARtorias742/DCTP/internal/config"
)

func main() {

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	handler := api.NewHandler(cfg)
	server := &http.Server{
		Addr:    cfg.APIPort,
		Handler: handler,
	}

	log.Printf("API server starting on %s", cfg.APIPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server failed: ", err)
	}
}
