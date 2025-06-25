package config

import (
	"errors"
	"flag"
	"fmt"
	"github.com/caarlos0/env/v11"
	"go.uber.org/zap/zapcore"
	"net"
	"net/url"
	"strings"
)

type Config struct {
	ServerAddr string `env:"SERVER_ADDRESS"`
	BaseURL    string `env:"BASE_URL"`
	LogLevel   string `env:"LOG_LEVEL"`
}

func NewConfig() (*Config, error) {
	serverAddr := flag.String("a", "localhost:8080", "HTTP server address")
	baseURL := flag.String("b", "http://localhost:8080", "Base URL")
	logLevel := flag.String("l", "info", "Log level")
	flag.Parse()
	cfg := &Config{}

	// use env
	if err := env.Parse(cfg); err != nil {
		panic(err)
	}

	// use flags if env not set
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = *serverAddr
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = *baseURL
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = *logLevel
	}

	// use defaults
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = "localhost:8080"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:8080"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	// validate
	if err := validateServerAddr(cfg.ServerAddr); err != nil {
		return nil, err
	}
	if err := validateBaseURL(cfg.BaseURL); err != nil {
		return nil, err
	}
	if err := validateLogLevel(cfg.LogLevel); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validateLogLevel(logLevel string) error {
	logLevel = strings.ToLower(logLevel)

	var level zapcore.Level
	err := level.UnmarshalText([]byte(logLevel))
	if err != nil {
		return fmt.Errorf("invalid log level: %s", logLevel)
	}
	return nil
}

func validateServerAddr(addr string) error {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid Server Address: %w", err)
	}
	return nil
}

func validateBaseURL(baseURL string) error {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid Base URL: %w", err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return errors.New("empty Scheme or Host")
	}
	return nil
}
