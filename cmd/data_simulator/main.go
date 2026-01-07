package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	config "github.com/pochkachaiki/iot4gds/internal/config/data_simulator" // замените
	"github.com/pochkachaiki/iot4gds/internal/sensor"
)

func main() {
	cfg := config.MustLoad()

	slog.Info("starting simulator",
		"device_number", cfg.DeviceNumber,
		"msg_frequency", cfg.MsgFrequency,
		"iot_system_url", cfg.IotSystemUrl)

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
