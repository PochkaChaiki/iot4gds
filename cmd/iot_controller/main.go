package main

import (
	"context"
	"fmt"
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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		path := r.URL.Path
		method := r.Method
		status := ww.status

		httpRequestsTotal.WithLabelValues(method, path, http.StatusText(status)).Inc()
		httpRequestDuration.WithLabelValues(method, path).Observe(duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func setupLogger() *slog.Logger {
	f, err := os.OpenFile("/app/logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("open log file: %v", err))
	}
	return slog.New(slog.NewJSONHandler(f, nil))
}

func main() {
	logger := setupLogger()
	slog.SetDefault(logger)

	cfg := config.MustLoad()

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

	instrumentedMux := metricsMiddleware(mux)

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: instrumentedMux,
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
