package main

import (
	"auth/internal/app/config"
	"auth/internal/app/server"
)

func main() {
	env := config.LoadConfig()
	server.Run(env)
}
