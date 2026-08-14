package server

import (
	"context"
	"fmt"
	"log"
	nats2 "nats"
	"net"
	"os"
	"os/signal"
	"profile/internal/app/config"
	"profile/internal/app/database"
	"profile/internal/app/nats"
	profilepb "proto/profile"
	"syscall"

	"google.golang.org/grpc"
)

func Run(cfg *config.Config) {
	log.Println("[PROFILE] Connecting DB...")
	db := database.Connect(cfg.DSN)

	log.Printf("[PROFILE] Connecting NATS...")
	natsClient, err := nats2.New(cfg.NATSUrl)
	if err != nil {
		log.Fatalf("[PROFILE] Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()
	consumer, err := nats.InitConsumer(ctx, natsClient)
	if err != nil {
		return
	}

	repo := database.NewRepository(db)
	service := NewService(repo)
	worker := nats.NewWorker(service, consumer)

	log.Println("[PROFILE] Starting worker...")
	go func() {
		if err := worker.Run(ctx); err != nil {
			log.Printf("[PROFILE] Worker stopped: %v", err)
		}
	}()

	log.Println("[PROFILE] Server started")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	h := NewHandler(service)
	profilepb.RegisterProfileServer(grpcServer, h)

	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
