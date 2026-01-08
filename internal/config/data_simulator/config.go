package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	IotSystemUrl string  `yaml:"iot_system_url" env-required:"true"`
	MetricsAddr  string  `yaml:"metrics_addr" env-default:":9092"`
	DeviceNumber int     `yaml:"device_number" env-required:"true"`
	MsgPeriod    float32 `yaml:"msg_period" env-required:"true"`
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
