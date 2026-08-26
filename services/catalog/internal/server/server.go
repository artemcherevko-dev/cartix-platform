package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"catalog/internal/config"
	"catalog/internal/db"

	catalogpb "proto/catalog"

	"google.golang.org/grpc"
)

const shutdownTimeout = 10 * time.Second

func Run(cfg *config.Config) error {
	gormDB, err := db.ConnectDB(cfg.DSNCatalog)
	if err != nil {
		return err
	}
	defer func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			sqlDB.Close()
		}
	}()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	repo := db.NewRepo(gormDB)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	catalogpb.RegisterCatalogServer(grpcServer, NewHandler(repo))

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("[CATALOG] gRPC server listening on :%s", cfg.GRPCPort)
		serveErr <- grpcServer.Serve(lis)
	}()

	select {
	case err := <-serveErr:
		if err != nil {
			return err
		}
	case <-ctx.Done():
	}

	log.Println("[CATALOG] Shutting down...")

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("[CATALOG] Stopped gracefully")
	case <-time.After(shutdownTimeout):
		log.Println("[CATALOG] Graceful stop timed out, forcing stop")
		grpcServer.Stop()
	}

	return nil
}
