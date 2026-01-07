package engine

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	config "github.com/pochkachaiki/iot4gds/internal/config/rule_engine"
	"github.com/pochkachaiki/iot4gds/internal/models/packet"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	lowPressure  = 0.03
	highPressure = 0.07
	lowTemp      = 5.0
	highTemp     = 40.0
)

type RecentPacket struct {
	Timestamp string  `bson:"timestamp"`
	Pressure  float32 `bson:"pressure"`
}

// type Engine struct {
// 	cfg        *config.Config
// 	packetColl *mongo.Collection
// 	alertColl  *mongo.Collection
// }

type Engine struct {
	cfg         *config.Config
	packetColl  *mongo.Collection
	alertColl   *mongo.Collection
	mu          sync.RWMutex
	recentCache map[int][]packet.Packet
}

func New(cfg *config.Config, packetColl, alertColl *mongo.Collection) *Engine {
	return &Engine{
		cfg:         cfg,
		packetColl:  packetColl,
		alertColl:   alertColl,
		recentCache: make(map[int][]packet.Packet),
	}
}

func (e *Engine) updateCache(p packet.Packet) {
	e.mu.Lock()
	defer e.mu.Unlock()

	queue := e.recentCache[p.DeviceID]
	if len(queue) >= e.cfg.SustainedCount {
		queue = queue[1:]
	}
	queue = append(queue, p)
	e.recentCache[p.DeviceID] = queue
}

func (e *Engine) getRecentPressures(deviceID int) []float32 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	packets := e.recentCache[deviceID]
	if len(packets) < e.cfg.SustainedCount {
		return nil
	}

	pressures := make([]float32, len(packets))
	for i, pkt := range packets {
		pressures[i] = pkt.Pressure
	}
	return pressures
}

// func New(cfg *config.Config, packetColl, alertColl *mongo.Collection) *Engine {
// 	return &Engine{
// 		cfg:        cfg,
// 		packetColl: packetColl,
// 		alertColl:  alertColl,
// 	}
// }

func (e *Engine) Run(ctx context.Context, ch *amqp.Channel, queue string) {
	msgs, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		slog.Error("consume error", "err", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				slog.Error("channel closed")
				return
			}
			if err := e.processMessage(ctx, msg.Body); err != nil {
				slog.Error("process message error", "err", err)
			}
			if err := msg.Ack(false); err != nil {
				slog.Error("ack error", "err", err)
			}
		}
	}
}

func (e *Engine) processMessage(parentCtx context.Context, body []byte) error {
	var p packet.Packet
	if err := json.Unmarshal(body, &p); err != nil {
		return err
	}

	e.updateCache(p) // всегда обновляем кэш

	ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
	defer cancel()

	reason := ""
	if p.Pressure < lowPressure {
		reason = "pressure low"
	} else if p.Pressure > highPressure {
		reason = "pressure high"
	} else if p.Temperature <= lowTemp {
		reason = "temperature low"
	} else if p.Temperature > highTemp {
		reason = "temperature high"
	}
	if reason != "" {
		_, err := e.alertColl.InsertOne(ctx, bson.M{
			"type":        "instant",
			"device_id":   p.DeviceID,
			"timestamp":   p.Timestamp,
			"reason":      reason,
			"pressure":    p.Pressure,
			"temperature": p.Temperature,
		})
		if err != nil {
			return err
		}
		slog.Info("instant alert", "device_id", p.DeviceID, "reason", reason)
	}

	var pressures []float32

	// Sustained rule — сначала проверяем кэш. Если в кэше нет - идём в бд
	if pressures = e.getRecentPressures(p.DeviceID); pressures == nil {
		opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(int64(e.cfg.SustainedCount))
		cursor, err := e.packetColl.Find(ctx, bson.M{"device_id": p.DeviceID}, opts)
		if err != nil {
			return err
		}
		defer cursor.Close(ctx)

		var recents []RecentPacket
		if err := cursor.All(ctx, &recents); err != nil {
			return err
		}

		if len(recents) < e.cfg.SustainedCount {
			return nil
		}

		sort.Slice(recents, func(i, j int) bool {
			ti, _ := time.Parse(time.RFC3339, recents[i].Timestamp)
			tj, _ := time.Parse(time.RFC3339, recents[j].Timestamp)
			return ti.Before(tj)
		})

		for _, r := range recents {
			pressures = append(pressures, r.Pressure)
		}
	}

	change := pressures[len(pressures)-1] - pressures[0]
	if math.Abs(float64(change)) >= float64(e.cfg.DeltaPressure) {
		dir := "increase"
		if change < 0 {
			dir = "decrease"
		}
		reason = "rapid pressure " + dir
		_, err := e.alertColl.InsertOne(ctx, bson.M{
			"type":      "sustained",
			"device_id": p.DeviceID,
			"timestamp": p.Timestamp,
			"reason":    reason,
			"change":    change,
		})
		if err != nil {
			return err
		}
		slog.Info("sustained alert", "device_id", p.DeviceID, "reason", reason, "change", change)
	}

	return nil
}
