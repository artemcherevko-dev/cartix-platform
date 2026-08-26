package main

import (
	"auth/internal/config"
	"auth/internal/server"
)

func main() {
	env := config.LoadConfig()
	server.Run(env)
}
