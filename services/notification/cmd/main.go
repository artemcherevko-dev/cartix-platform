package main

import (
	"log"
	"notification/internal/config"
	"notification/internal/server"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("warning: .env not loaded")
	}
	cfg := config.LoadConfig()
	err := server.Run(cfg)
	if err != nil {
		log.Fatalf("[NOTIFICATION] Worker stopped: %v", err)
	}
}
