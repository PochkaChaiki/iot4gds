package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "github.com/pochkachaiki/iot4gds/internal/config/iot_controller"
	"github.com/pochkachaiki/iot4gds/internal/handler"
	"github.com/pochkachaiki/iot4gds/internal/queue"
	"github.com/pochkachaiki/iot4gds/internal/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func setupLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

func main() {
	logger := setupLogger()
	slog.SetDefault(logger)

	cfg := config.MustLoad()

	slog.Info("starting rule engine", "mongo_uri", cfg.MongoURI, "rabbit_uri", cfg.RabbitURI,
		"queue", cfg.QueueName)

	mongoClient, err := storage.NewMongoClient(cfg.MongoURI)
	if err != nil {
		slog.Error("mongo connect error", "err", err)
		os.Exit(1)
	}
	defer mongoClient.Disconnect(context.Background())

	rabbitConn, err := queue.NewRabbitConnection(cfg.RabbitURI)
	if err != nil {
		slog.Error("rabbitmq connect error", "err", err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	rabbitCh, err := rabbitConn.Channel()
	if err != nil {
		slog.Error("rabbitmq channel error", "err", err)
		os.Exit(1)
	}
	defer rabbitCh.Close()

	err = queue.DeclareQueue(rabbitCh, cfg.QueueName)
	if err != nil {
		slog.Error("declare queue error", "err", err)
		os.Exit(1)
	}

	db := mongoClient.Database(cfg.DBName)
	collection := db.Collection(cfg.PacketCollection)

	h := handler.New(collection, rabbitCh, cfg.QueueName)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /packets", h.HandlePacket)
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "err", err)
	}

	slog.Info("iot controller stopped")
}
