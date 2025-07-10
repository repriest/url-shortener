package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/repriest/url-shortener/internal/config"
	"github.com/repriest/url-shortener/internal/handlers"
	"github.com/repriest/url-shortener/internal/logger"
	"github.com/repriest/url-shortener/internal/storage"
	t "github.com/repriest/url-shortener/internal/storage/types"
	"github.com/repriest/url-shortener/internal/zipper"
	"go.uber.org/zap"
	"log"
	"net/http"
)

func initConfig() (*config.Config, error) {
	cfg, err := config.NewConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func initLogger(cfg *config.Config) error {
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}
	return nil
}

func initStorage(cfg *config.Config) (*storage.Repository, error) {
	var st t.Storage

	if cfg.DatabaseDSN != "" {
		st, err := storage.NewPostgresStorage(cfg.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("could not connect to postgres: %w", err)
		}
		return storage.NewRepository(st)

	} else if cfg.FileStoragePath != "" {
		st, err := storage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %w", cfg.FileStoragePath, err)
		}
		return storage.NewRepository(st)

	} else {
		st = storage.NewMemoryStorage()
	}

	return storage.NewRepository(st)
}

func closeStorage(st *storage.Repository) {
	if err := st.Close(); err != nil {
		logger.Log.Error("could not close storage", zap.Error(err))
	}
}

func initRouter(cfg *config.Config, st *storage.Repository) *chi.Mux {
	h := handlers.NewHandler(cfg, st)
	r := chi.NewRouter()
	r.Post("/", logger.RequestLogger(zipper.GzipMiddleware(h.ShortenHandler)))
	r.Get("/{id}", logger.ResponseLogger(zipper.GzipMiddleware(h.ExpandHandler)))
	r.Post("/api/shorten", logger.RequestLogger(zipper.GzipMiddleware(h.ShortenJSONHandler)))
	r.Get("/ping", logger.RequestLogger(zipper.GzipMiddleware(h.PingHandler)))
	r.Post("/api/shorten/batch", logger.RequestLogger(zipper.GzipMiddleware(h.ShortenBatchHandler)))
	return r
}

func main() {
	cfg, err := initConfig()
	if err != nil {
		log.Fatal(err)
	}

	err = initLogger(cfg)
	if err != nil {
		log.Fatal(err)
	}

	store, err := initStorage(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer closeStorage(store)

	r := initRouter(cfg, store)
	err = http.ListenAndServe(cfg.ServerAddr, r)
	if err != nil {
		log.Fatal(err)
	}
}
