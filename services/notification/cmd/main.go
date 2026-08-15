package main

import (
	"log"
	"notification/internal/app/config"
	"notification/internal/app/server"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("warning: .env not loaded")
	}
	cfg := config.LoadConfig()
	err := app.Run(cfg)
	if err != nil {
		log.Fatalf("[PROFILE] Worker stopped: %v", err)
	}
}
