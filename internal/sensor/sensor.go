package sensor

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	config "github.com/pochkachaiki/iot4gds/internal/config/data_simulator"
	"github.com/pochkachaiki/iot4gds/internal/models/packet"
	"github.com/pochkachaiki/iot4gds/internal/sender"
)

const (
	defaultMeanPressure    = 0.05 // МПа
	defaultMeanTemperature = 24.0 //
	anomalyProbability     = 0.03
	anomalyDuration        = 10
	deltaPa                = 19.6133 // Паскали
	deltaMPa               = deltaPa / 1e6
)

func Run(ctx context.Context, cfg *config.Config, deviceID int) {
	ticker := time.NewTicker(time.Duration(cfg.MsgFrequency) * time.Second)
	defer ticker.Stop()

	slog.InfoContext(ctx, "device started", "device_id", deviceID, "msg_frequency", cfg.MsgFrequency)

	anomalyCounter := 0
	anomalyDirection := 0                                              // 1: up, -1: down
	anomalyIncrement := float32(deltaMPa / float64(anomalyDuration-1)) // приращение на пакет для достижения delta за 10 пакетов
	currentMeanPressure := float32(defaultMeanPressure)

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "device stopped", "device_id", deviceID)
			return
		case <-ticker.C:
			if anomalyCounter == 0 {
				if rand.Float32() < anomalyProbability {
					anomalyCounter = anomalyDuration
					if rand.Float32() < 0.5 {
						anomalyDirection = 1
					} else {
						anomalyDirection = -1
					}
					slog.InfoContext(ctx, "starting anomaly", "device_id", deviceID, "anomaly_direction", anomalyDirection)
				}
			}

			if anomalyCounter > 0 {
				currentMeanPressure += float32(anomalyDirection) * anomalyIncrement
				anomalyCounter--
				if anomalyCounter == 0 {
					currentMeanPressure = defaultMeanPressure // сброс после аномалии
					slog.InfoContext(ctx, "anomaly ended", "device_id", deviceID)
				}
			}

			p := packet.Generate(deviceID, currentMeanPressure, defaultMeanTemperature)
			err := sender.Send(cfg.IotSystemUrl, p)
			if err != nil {
				slog.ErrorContext(ctx, "send error", "device_id", deviceID, "err", err)
			} else {
				slog.InfoContext(ctx, "sent packet", "device_id", deviceID, "pressure", p.Pressure, "temperature", p.Temperature)
			}
		}
	}
}
