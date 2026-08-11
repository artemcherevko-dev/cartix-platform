package main

import (
	"auth/cmd/internal/config"
	"auth/cmd/server"
)

func main() {
	env := config.LoadConfig()
	server.Run(env)
}
