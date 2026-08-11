package server

import (
	"auth/cmd/internal/app"
	"auth/cmd/internal/app/nats"
	"auth/cmd/internal/config"
	"auth/cmd/internal/db"
	"context"
	"fmt"
	"log"
	natsclient "nats"
	"net"
	"os"
	"os/signal"
	authpb "proto/auth"
	"syscall"

	"google.golang.org/grpc"
)

func Run(cfg *config.Config) {
	log.Println("[AUTH] Connecting DB...")
	database, err := db.ConnectDB(cfg.DSNAuth)
	if err != nil {
		panic(err)
	}

	log.Println("[AUTH] Connecting NATS...")
	natsClient, err := natsclient.New(cfg.NATSUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer natsClient.Close()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()
	if err := nats.InitStreams(ctx, natsClient); err != nil {
		log.Fatal(err)
	}

	repo := db.NewRepo(database)

	log.Println("[AUTH] Starting server...")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPortAuth))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	h := app.NewHandler(repo, natsClient)

	authpb.RegisterAuthServer(grpcServer, h)

	err = grpcServer.Serve(lis)
	if err != nil {
		return
	}
}
