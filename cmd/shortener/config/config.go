package config

import (
	"flag"
)

type Config struct {
	ServerAddr string
	BaseURL    string
}

func NewConfig() *Config {
	serverAddr := flag.String("a", "localhost:8080", "HTTP server address")
	baseURL := flag.String("b", "http://localhost:8080", "Base URL")
	flag.Parse()

	return &Config{
		ServerAddr: *serverAddr,
		BaseURL:    *baseURL,
	}
}
