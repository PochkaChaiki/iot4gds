package sender

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pochkachaiki/iot4gds/internal/models/packet"
)

const (
	contentType = "application/json"
	timeout     = 5 * time.Second
)

// Send отправляет один пакет в IoT контроллер по HTTP POST
func Send(url string, p packet.Packet) error {
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal packet: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
