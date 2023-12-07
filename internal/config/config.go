package config

import (
	"fmt"

	"github.com/caarlos0/env/v9"
)

type Config struct {
	Port     int    `env:"PORT" envDefault:"50000"`
	Local    bool   `env:"LOCAL" envDefault:"false"`
	LogLevel string `env:"LOC_LEVEL" envDefault:"info"`
	RedisURL string `env:"REDIS_URL" envDefault:"localhost:6379"`
}

func NewConfig() (*Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}
