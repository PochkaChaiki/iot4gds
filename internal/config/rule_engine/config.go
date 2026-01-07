package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	MongoURI         string  `yaml:"mongo_uri" env-required:"true"`
	RabbitURI        string  `yaml:"rabbit_uri" env-required:"true"`
	QueueName        string  `yaml:"queue_name" env-default:"packets"`
	DBName           string  `yaml:"db_name" env-default:"iot"`
	PacketCollection string  `yaml:"packet_collection" env-default:"packets"`
	AlertCollection  string  `yaml:"alert_collection" env-default:"alerts"`
	SustainedCount   int     `yaml:"sustained_count" env-default:"10"`
	DeltaPressure    float32 `yaml:"delta_pressure" env-default:"0.0000196133"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		panic("CONFIG_PATH environment variable is not set")
	}
	if _, err := os.Stat(configPath); err != nil {
		panic(fmt.Errorf("error opening config file: %s", err))
	}
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic(fmt.Errorf("error reading config file: %s", err))
	}
	return &cfg
}
