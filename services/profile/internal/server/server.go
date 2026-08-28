package server

import (
	"context"
	"fmt"
	"log"
	natsclient "nats"
	"net"
	"os"
	"os/signal"
	"profile/internal/config"
	"profile/internal/db"
	"profile/internal/nats"
	profilepb "proto/profile"
	telemetry "shared/metrics"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

const shutdownTimeout = 10 * time.Second

func Run(cfg *config.Config) {
	log.Println("[PROFILE] Connecting DB...")
	conn := db.Connect(cfg.DSN)

	log.Printf("[PROFILE] Connecting NATS...")
	natsClient, err := natsclient.New(cfg.NATSUrl)
	if err != nil {
		log.Fatalf("[PROFILE] Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	sqlDB, err := conn.DB()
	if err != nil {
		log.Printf("[PROFILE] Failed to get DB handle: %v", err)
	} else {
		defer func() {
			if err := sqlDB.Close(); err != nil {
				log.Printf("[PROFILE] DB close: %v", err)
			}
		}()
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()
	obs, err := telemetry.New(ctx, telemetry.ConfigFromEnv("profile-service"))
	if err != nil {
		log.Printf("[PROFILE] telemetry initialization: %v", err)
		return
	}
	if err := obs.Start(); err != nil {
		log.Printf("[PROFILE] telemetry start: %v", err)
		return
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()
		if err := obs.Shutdown(shutdownCtx); err != nil {
			log.Printf("[PROFILE] telemetry shutdown: %v", err)
		}
	}()
	consumer, err := nats.InitConsumer(ctx, natsClient)
	if err != nil {
		return
	}

	repo := db.NewRepository(conn)
	service := NewService(repo)
	worker := nats.NewWorker(service, consumer)

	workerDone := make(chan struct{})
	log.Println("[PROFILE] Starting worker...")
	go func() {
		defer close(workerDone)
		if err := worker.Run(ctx); err != nil {
			log.Printf("[PROFILE] Worker stopped: %v", err)
		}
	}()

	log.Println("[PROFILE] Server started")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer(grpc.StatsHandler(obs.GRPCStatsHandler()), grpc.UnaryInterceptor(telemetry.UnaryServerInterceptor()))
	h := NewHandler(service)
	profilepb.RegisterProfileServer(grpcServer, h)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(lis)
	}()

	select {
	case err := <-serveErr:
		if err != nil {
			log.Printf("[PROFILE] Server stopped: %v", err)
		}
	case <-ctx.Done():
	}

	log.Println("[PROFILE] Shutting down...")
	cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("[PROFILE] Server stopped gracefully")
	case <-time.After(shutdownTimeout):
		log.Println("[PROFILE] Graceful stop timed out, forcing stop")
		grpcServer.Stop()
	}

	select {
	case <-workerDone:
		log.Println("[PROFILE] Worker stopped")
	case <-time.After(shutdownTimeout):
		log.Println("[PROFILE] Waiting for worker timed out")
	}
}
