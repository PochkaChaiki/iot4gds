package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	config "github.com/pochkachaiki/iot4gds/internal/config/data_simulator"
	"github.com/pochkachaiki/iot4gds/internal/sensor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func setupLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

func main() {
	logger := setupLogger()
	slog.SetDefault(logger)

	cfg := config.MustLoad()

	slog.Info("starting simulator",
		"device_number", cfg.DeviceNumber,
		"msg_frequency", cfg.MsgFrequency,
		"iot_system_url", cfg.IotSystemUrl,
		"metrics_addr", cfg.MetricsAddr)

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(cfg.MetricsAddr, nil); err != nil {
			slog.Error("metrics server error", "err", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	for id := 1; id <= cfg.DeviceNumber; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sensor.Run(ctx, cfg, id)
		}(id)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutdown signal received")
	cancel()
	wg.Wait()

	slog.Info("simulator stopped")
}
