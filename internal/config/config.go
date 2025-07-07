package config

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"github.com/caarlos0/env/v11"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap/zapcore"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	ServerAddr      string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	LogLevel        string `env:"LOG_LEVEL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	defaults := &Config{
		ServerAddr:      "localhost:8080",
		BaseURL:         "http://localhost:8080",
		LogLevel:        "info",
		FileStoragePath: "", // url_store.json
		DatabaseDSN:     "", // postgres://postgres:admin@localhost:5432/postgres?sslmode=disable
	}

	flag.StringVar(&cfg.ServerAddr, "a", defaults.ServerAddr, "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", defaults.BaseURL, "Base URL")
	flag.StringVar(&cfg.LogLevel, "l", defaults.LogLevel, "Log level")
	flag.StringVar(&cfg.FileStoragePath, "f", defaults.FileStoragePath, "File storage path")
	flag.StringVar(&cfg.DatabaseDSN, "d", defaults.DatabaseDSN, "Database DSN")
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
	if cfg.FileStoragePath != "" {
		if err := validateFileStoragePath(cfg.FileStoragePath); err != nil {
			return nil, err
		}
	}
	if cfg.DatabaseDSN != "" {
		if err := validateDatabaseDSN(cfg.DatabaseDSN); err != nil {
			return nil, err
		}
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
	// check if the path is a dir
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("file storage path cannot be a directory: %s", path)
		}
	}
	return nil
}

func validateDatabaseDSN(dsn string) error {
	_, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("invalid Database DSN: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("invalid Database DSN: %w", err)
	}
	defer db.Close()

	return db.PingContext(ctx)
}
