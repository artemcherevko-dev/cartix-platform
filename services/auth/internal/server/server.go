package server

import (
	"auth/internal/config"
	"auth/internal/db"
	"auth/internal/nats"
	"auth/internal/verify"
	"context"
	"fmt"
	"log"
	natsclient "nats"
	"net"
	"os"
	"os/signal"
	authpb "proto/auth"
	redisclient "redis"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

const shutdownTimeout = 10 * time.Second

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

	log.Println("[AUTH] Connecting Redis...")
	redisClient, err := redisclient.New(cfg.RedisUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("[AUTH] Redis close: %v", err)
		}
	}()

	sqlDB, err := database.DB()
	if err != nil {
		log.Printf("[AUTH] Failed to get DB handle: %v", err)
	} else {
		defer func() {
			if err := sqlDB.Close(); err != nil {
				log.Printf("[AUTH] DB close: %v", err)
			}
		}()
	}

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
	verifyStore := verify.NewStore(redisClient, cfg.VerifyEmailTokenTTL)

	log.Println("[AUTH] Starting server...")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPortAuth))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	h := NewHandler(repo, natsClient, verifyStore)

	authpb.RegisterAuthServer(grpcServer, h)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(lis)
	}()

	select {
	case err := <-serveErr:
		if err != nil {
			log.Printf("[AUTH] Server stopped: %v", err)
		}
	case <-ctx.Done():
	}

	log.Println("[AUTH] Shutting down...")
	cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("[AUTH] Server stopped gracefully")
	case <-time.After(shutdownTimeout):
		log.Println("[AUTH] Graceful stop timed out, forcing stop")
		grpcServer.Stop()
	}
}
