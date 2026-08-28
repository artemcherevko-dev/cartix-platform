package main

import (
	"log"
	"profile/internal/config"
	"profile/internal/server"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
	cfg := config.LoadConfig()
	server.Run(cfg)
}
