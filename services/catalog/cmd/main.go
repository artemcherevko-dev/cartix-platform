package main

import (
	"log"

	"catalog/internal/config"
	"catalog/internal/server"
)

func main() {
	cfg := config.LoadConfig()
	if err := server.Run(cfg); err != nil {
		log.Fatalf("[CATALOG] Service stopped: %v", err)
	}
}
