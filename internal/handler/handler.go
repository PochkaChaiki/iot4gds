package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/pochkachaiki/iot4gds/internal/models/packet"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Handler struct {
	collection *mongo.Collection
	rabbitCh   *amqp.Channel
}

func New(collection *mongo.Collection, rabbitCh *amqp.Channel) *Handler {
	return &Handler{
		collection: collection,
		rabbitCh:   rabbitCh,
	}
}

func (h *Handler) HandlePacket(w http.ResponseWriter, r *http.Request) {
	var p packet.Packet
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		slog.Error("decode error", "err", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if p.DeviceID <= 0 || p.Timestamp == "" || p.Pressure < 0 || p.Temperature < 0 {
		slog.Error("validation error", "packet", p)
		http.Error(w, "invalid packet", http.StatusBadRequest)
		return
	}

	_, err := time.Parse(time.RFC3339, p.Timestamp)
	if err != nil {
		slog.Error("timestamp parse error", "err", err)
		http.Error(w, "invalid timestamp", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = h.collection.InsertOne(ctx, bson.M{
		"device_id":   p.DeviceID,
		"timestamp":   p.Timestamp,
		"pressure":    p.Pressure,
		"temperature": p.Temperature,
	})
	if err != nil {
		slog.Error("mongo insert error", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(p)
	if err != nil {
		slog.Error("marshal for rabbit error", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	err = h.rabbitCh.PublishWithContext(ctx, "", "packets", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
	if err != nil {
		slog.Error("rabbit publish error", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
