package main

import (
	"log"

	"order/internal/config"
	"order/internal/server"
)

func main() {
	cfg := config.LoadConfig()
	if err := server.Run(cfg); err != nil {
		log.Fatalf("[ORDER] Service stopped: %v", err)
	}
}
