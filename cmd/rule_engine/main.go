// cmd/main.go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	config "github.com/pochkachaiki/iot4gds/internal/config/rule_engine"
	"github.com/pochkachaiki/iot4gds/internal/engine"
	"github.com/pochkachaiki/iot4gds/internal/queue"
	"github.com/pochkachaiki/iot4gds/internal/storage"
)

func main() {
	cfg := config.MustLoad()

	slog.Info("starting rule engine", "mongo_uri", cfg.MongoURI, "rabbit_uri", cfg.RabbitURI, "queue", cfg.QueueName,
		"sustained_count", cfg.SustainedCount, "delta_pressure", cfg.DeltaPressure)

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
	packetColl := db.Collection(cfg.PacketCollection)
	alertColl := db.Collection(cfg.AlertCollection)

	e := engine.New(cfg, packetColl, alertColl)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		e.Run(ctx, rabbitCh, cfg.QueueName)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutdown signal received")
	cancel()

	slog.Info("rule engine stopped")
}
