package main

import (
	"log"

	"payments/internal/config"
	"payments/internal/server"
)

func main() {
	cfg := config.LoadConfig()
	if err := server.Run(cfg); err != nil {
		log.Fatalf("[PAYMENTS] Service stopped: %v", err)
	}
}
