package config

import (
	"flag"
	"github.com/caarlos0/env/v11"
)

type Config struct {
	ServerAddr string `env:"SERVER_ADDRESS"`
	BaseURL    string `env:"BASE_URL"`
}

func NewConfig() *Config {
	serverAddr := flag.String("a", "localhost:8080", "HTTP server address")
	baseURL := flag.String("b", "http://localhost:8080", "Base URL")
	flag.Parse()
	cfg := &Config{}

	// env
	if err := env.Parse(cfg); err != nil {
		panic(err)
	}

	// flags fallback
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = *serverAddr
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = *baseURL
	}

	// defaults fallback
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = "localhost:8080"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:8080"
	}

	return cfg
}
