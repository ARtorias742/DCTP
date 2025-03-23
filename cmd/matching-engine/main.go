package main

import (
	"log"

	"github.com/ARtorias742/DCTP/internal/config"
	"github.com/ARtorias742/DCTP/internal/matching"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	engine := matching.NewEngine(cfg)
	if err := engine.Start(); err != nil {
		log.Fatal("Failed to start matching engine:", err)
	}
}
