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

	natsclient "nats"
	catalogpb "proto/catalog"
	orderpb "proto/order"
	paymentspb "proto/payments"
	telemetry "shared/metrics"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"order/internal/config"
	"order/internal/db"
	ordernats "order/internal/nats"
)

const shutdownTimeout = 10 * time.Second

func parseUUID(raw string) (uuid.UUID, error) {
	return uuid.Parse(raw)
}

func Run(cfg *config.Config) error {
	gormDB, err := db.ConnectDB(cfg.DSNOrders)
	if err != nil {
		return err
	}
	defer func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			sqlDB.Close()
		}
	}()

	log.Println("[ORDER] Connecting gRPC dependencies...")
	catalogConn, err := grpc.NewClient(
		cfg.CatalogURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer catalogConn.Close()

	paymentsConn, err := grpc.NewClient(
		cfg.PaymentsURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer paymentsConn.Close()

	repo := db.NewRepo(gormDB)
	service := NewService(
		repo,
		NewCatalogGRPC(catalogpb.NewCatalogClient(catalogConn)),
		NewPaymentsGRPC(paymentspb.NewPaymentsClient(paymentsConn)),
	)

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	obs, err := telemetry.New(ctx, telemetry.ConfigFromEnv("order-service"))
	if err != nil {
		return err
	}
	if err := obs.Start(); err != nil {
		return err
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()
		if err := obs.Shutdown(shutdownCtx); err != nil {
			log.Printf("[ORDER] telemetry shutdown: %v", err)
		}
	}()

	natsClient, err := natsclient.New(cfg.NATSUrl)
	if err != nil {
		return err
	}
	defer natsClient.Close()

	if err := ordernats.InitStreams(ctx, natsClient); err != nil {
		return err
	}

	service.SetStatusPublisher(func(ctx context.Context, event OrderStatusEvent) error {
		items := make([]ordernats.OrderStatusItem, 0, len(event.Items))
		for _, item := range event.Items {
			items = append(items, ordernats.OrderStatusItem{
				Name:        item.Name,
				Quantity:    item.Quantity,
				AmountMinor: item.AmountMinor,
			})
		}

		return ordernats.PublishOrderStatusUpdated(ctx, natsClient, ordernats.OrderStatusUpdated{
			OrderID:        event.OrderID,
			UserID:         event.UserID,
			Email:          event.Email,
			Status:         event.Status,
			PreviousStatus: event.PreviousStatus,
			TotalMinor:     event.TotalMinor,
			Currency:       event.Currency,
			Items:          items,
			UpdatedAt:      event.UpdatedAt,
		})
	})

	consumer, err := ordernats.InitConsumer(ctx, natsClient)
	if err != nil {
		return err
	}

	workerDone := make(chan error, 1)
	go func() {
		worker := ordernats.NewWorker(consumer, service)
		workerDone <- worker.Run(ctx)
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(grpc.StatsHandler(obs.GRPCStatsHandler()), grpc.UnaryInterceptor(telemetry.UnaryServerInterceptor()))
	orderpb.RegisterOrderServer(grpcServer, NewHandler(service))

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("[ORDER] gRPC server listening on :%s", cfg.GRPCPort)
		serveErr <- grpcServer.Serve(lis)
	}()

	select {
	case err := <-serveErr:
		if err != nil && ctx.Err() == nil {
			return err
		}
	case <-ctx.Done():
	case err := <-workerDone:
		if err != nil && ctx.Err() == nil {
			return err
		}
	}

	log.Println("[ORDER] Shutting down...")
	cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("[ORDER] Server stopped gracefully")
	case <-time.After(shutdownTimeout):
		log.Println("[ORDER] Graceful stop timed out, forcing stop")
		grpcServer.Stop()
	}

	select {
	case err := <-workerDone:
		if err != nil {
			log.Printf("[ORDER] Worker stopped: %v", err)
		} else {
			log.Println("[ORDER] Worker stopped gracefully")
		}
	case <-time.After(shutdownTimeout):
		log.Println("[ORDER] Waiting for worker timed out")
	}

	return nil
}
