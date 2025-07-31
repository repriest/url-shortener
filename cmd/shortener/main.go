package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/repriest/url-shortener/internal/config"
	"github.com/repriest/url-shortener/internal/handlers"
	"github.com/repriest/url-shortener/internal/middleware/auth"
	"github.com/repriest/url-shortener/internal/middleware/logger"
	"github.com/repriest/url-shortener/internal/middleware/zipper"
	"github.com/repriest/url-shortener/internal/storage/file"
	"github.com/repriest/url-shortener/internal/storage/memory"
	"github.com/repriest/url-shortener/internal/storage/postgres"
	t "github.com/repriest/url-shortener/internal/storage/types"
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

func initStorage(cfg *config.Config) (t.Storage, error) {
	if cfg.DatabaseDSN != "" {
		st, err := postgres.NewPgStorage(cfg.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("could not connect to postgres: %w", err)
		}
		return st, nil
	}

	if cfg.FileStoragePath != "" {
		st, err := file.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %w", cfg.FileStoragePath, err)
		}
		return st, nil
	}

	return memory.NewMemoryStorage()
}

func closeStorage(st t.Storage) {
	if err := st.Close(); err != nil {
		logger.Log.Error("could not close storage", zap.Error(err))
	}
}

func initRouter(cfg *config.Config, st t.Storage) *chi.Mux {
	h := handlers.NewHandler(cfg, st)
	r := chi.NewRouter()

	r.Get("/ping", h.PingHandler)
	r.Group(func(r chi.Router) {
		r.Use(logger.RequestLogger, logger.ResponseLogger, zipper.GzipMiddleware, auth.SetCookieMiddleware(cfg))
		r.Post("/", h.ShortenHandler)
		r.Get("/{id}", h.ExpandHandler)
		r.Post("/api/shorten", h.ShortenJSONHandler)
		r.Post("/api/shorten/batch", h.ShortenBatchHandler)
	})
	r.Group(func(r chi.Router) {
		r.Use(logger.RequestLogger, logger.ResponseLogger, zipper.GzipMiddleware, auth.SetCookieMiddleware(cfg), auth.AuthRequiredMiddleware(cfg))
		r.Get("/api/user/urls", h.GetUserURLsHandler)
		r.Delete("/api/user/urls", h.DeleteURLsHandler)
	})

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
