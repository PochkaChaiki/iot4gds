package packet

import (
	"math"
	"math/rand"
	"time"
)

type Packet struct {
	Timestamp   string  `json:"timestamp"`
	DeviceID    int     `json:"device_id"`
	Pressure    float32 `json:"pressure"`
	Temperature float32 `json:"temperature"`
}

// Генерация реалистичных значений для газораспределительной станции
// Нормальные значения:
//
//	Pressure:  среднее - 0.05 ; предупредит пороги: 0.04 - 0.06 МПа
//	Temperature: 8 - 40 град
//
// Критические пороги (для мгновенного правила):
//
//	Pressure:    < 0.03 или > 0.07
//	Temperature: <= 5 или > 40
//
// Иногда генерируем значения за пределами нормы, чтобы правила срабатывали.
func Generate(deviceID int, meanPressure, meanTemperature float32) Packet {
	var pressure, temperature float32

	if rand.Float32() < 0.95 { // 95% — нормальные значения
		pressure = meanPressure + float32(rand.NormFloat64()*0.005)     // 0.005 МПа
		temperature = meanTemperature + float32(rand.NormFloat64()*5.0) // 5 °C

		pressure = float32(math.Max(0.04, math.Min(0.06, float64(pressure))))
		temperature = float32(math.Max(8.0, math.Min(40.0, float64(temperature))))
	} else { // 5% — критические выбросы
		if rand.Float32() < 0.5 {
			pressure = meanPressure - 0.03 // < 0.03
			if pressure < 0 {
				pressure = 0
			}
		} else {
			pressure = meanPressure + 0.03 // > 0.07
		}

		if rand.Float32() < 0.5 {
			temperature = meanTemperature - 10 // <= 5
			if temperature < 0 {
				temperature = 0
			}
		} else {
			temperature = meanTemperature + 10 // > 40
		}
	}

	return Packet{
		DeviceID:    deviceID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Pressure:    pressure,
		Temperature: temperature,
	}
}
