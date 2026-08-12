package main

import (
	"fmt"
	"gateway/cmd/internal/app"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
	port := os.Getenv("GATEWAY_PORT")
	if port == "" {
		port = "3000"
	}
	app.Run(fmt.Sprintf("%s", port))
}
