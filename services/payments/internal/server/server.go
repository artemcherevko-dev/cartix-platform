package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	natsclient "nats"
	paymentspb "proto/payments"

	"github.com/stripe/stripe-go/v85"
	"google.golang.org/grpc"

	"payments/internal/checkout"
	"payments/internal/config"
	"payments/internal/db"
	paymentsnats "payments/internal/nats"
	"payments/internal/webhook"
)

const shutdownTimeout = 10 * time.Second

func Run(cfg *config.Config) error {
	gormDB, err := db.ConnectDB(cfg.DSNPayments)
	if err != nil {
		return err
	}

	repo := db.NewRepo(gormDB)

	log.Println("[PAYMENTS] Connecting NATS...")
	natsClient, err := natsclient.New(cfg.NATSUrl)
	if err != nil {
		return err
	}
	defer natsClient.Close()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	if err := paymentsnats.InitStreams(ctx, natsClient); err != nil {
		return err
	}

	publish := func(ctx context.Context, event paymentsnats.PaymentSucceeded) error {
		return paymentsnats.PublishPaymentSucceeded(ctx, natsClient, event)
	}

	webhookHandler := webhook.NewHandler(
		cfg.StripeWebhookSecret,
		repo,
		publish,
	)

	var stripeClient *stripe.Client
	if apiURL := cfg.StripeAPIURL; apiURL != "" {
		// Test/development hook: point the Stripe backend at a mock server.
		backends := stripe.NewBackendsWithConfig(&stripe.BackendConfig{
			URL: stripe.String(apiURL),
		})
		stripeClient = stripe.NewClient(cfg.StripeSecretKey, stripe.WithBackends(backends))
	} else {
		stripeClient = stripe.NewClient(cfg.StripeSecretKey)
	}
	creator := checkout.NewCreator(stripeClient.V1CheckoutSessions, cfg.AppURL)
	paymentsHandler := NewHandler(repo, creator)

	mux := http.NewServeMux()
	mux.Handle("POST /v1/payments/webhook", webhookHandler)

	httpServer := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	paymentspb.RegisterPaymentsServer(grpcServer, paymentsHandler)

	httpErr := make(chan error, 1)
	go func() {
		log.Printf("[PAYMENTS] HTTP server listening on :%s", cfg.HTTPPort)
		httpErr <- httpServer.ListenAndServe()
	}()

	grpcErr := make(chan error, 1)
	go func() {
		log.Printf("[PAYMENTS] gRPC server listening on :%s", cfg.GRPCPort)
		grpcErr <- grpcServer.Serve(lis)
	}()

	select {
	case err := <-httpErr:
		if !errors.Is(err, http.ErrServerClosed) && ctx.Err() == nil {
			return err
		}
	case err := <-grpcErr:
		if ctx.Err() == nil {
			return err
		}
	case <-ctx.Done():
	}

	log.Println("[PAYMENTS] Shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	var firstErr error
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		firstErr = err
	}

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(shutdownTimeout):
		grpcServer.Stop()
	}

	if firstErr != nil {
		return firstErr
	}

	log.Println("[PAYMENTS] Stopped gracefully")

	return nil
}
