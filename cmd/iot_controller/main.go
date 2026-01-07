package cmd

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
)

func main() {
	cfg := config.MustLoad()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("starting iot controller", "http_addr", cfg.HTTPAddr, "mongo_uri", cfg.MongoURI, "rabbit_uri", cfg.RabbitURI)

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

	err = queue.DeclareQueue(rabbitCh, "packets")
	if err != nil {
		slog.Error("declare queue error", "err", err)
		os.Exit(1)
	}

	db := mongoClient.Database("iot")
	collection := db.Collection("packets")

	h := handler.New(collection, rabbitCh)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /packets", h.HandlePacket)

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
