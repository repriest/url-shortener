package config

import (
	"errors"
	"flag"
	"fmt"
	"github.com/caarlos0/env/v11"
	"go.uber.org/zap/zapcore"
	"net"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	ServerAddr      string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	LogLevel        string `env:"LOG_LEVEL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	defaults := &Config{
		ServerAddr:      "localhost:8080",
		BaseURL:         "http://localhost:8080",
		LogLevel:        "info",
		FileStoragePath: "url_store.json",
	}

	flag.StringVar(&cfg.ServerAddr, "server_addr", defaults.ServerAddr, "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "base_url", defaults.BaseURL, "Base URL")
	flag.StringVar(&cfg.LogLevel, "log_level", defaults.LogLevel, "Log level")
	flag.StringVar(&cfg.FileStoragePath, "file_storage_path", defaults.FileStoragePath, "File storage path")
	flag.Parse()

	// use env
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env: %w", err)
	}

	// use defaults
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = defaults.ServerAddr
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaults.BaseURL
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = defaults.LogLevel
	}
	if cfg.FileStoragePath == "" {
		cfg.FileStoragePath = defaults.FileStoragePath
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
	if err := validateFileStoragePath(cfg.FileStoragePath); err != nil {
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

func validateFileStoragePath(path string) error {
	// check if the path is empty
	if path == "" {
		return errors.New("empty file storage path")
	}

	// check if the path is a dir
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("file storage path cannot be a directory: %s", path)
		}
	}
	return nil
}
