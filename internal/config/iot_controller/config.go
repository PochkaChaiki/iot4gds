package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HTTPAddr  string `yaml:"http_addr" env-required:"true"`
	MongoURI  string `yaml:"mongo_uri" env-required:"true"`
	RabbitURI string `yaml:"rabbit_uri" env-required:"true"`
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
