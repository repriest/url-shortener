package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/repriest/url-shortener/internal/config"
	"github.com/repriest/url-shortener/internal/handlers"
	"github.com/repriest/url-shortener/internal/logger"
	"github.com/repriest/url-shortener/internal/storage"
	"github.com/repriest/url-shortener/internal/zipper"
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
	st, err := storage.NewRepository(cfg.FileStoragePath)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func initRouter(cfg *config.Config, st *storage.Repository) *chi.Mux {
	h := handlers.NewHandler(cfg, st)
	r := chi.NewRouter()
	r.Post("/", logger.RequestLogger(zipper.GzipMiddleware(h.ShortenHandler)))
	r.Get("/{id}", logger.ResponseLogger(zipper.GzipMiddleware(h.ExpandHandler)))
	r.Post("/api/shorten", logger.RequestLogger(zipper.GzipMiddleware(h.ShortenJSONHandler)))
	r.Post("/ping", logger.RequestLogger(zipper.GzipMiddleware(h.PingHandler)))
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

	r := initRouter(cfg, store)
	err = http.ListenAndServe(cfg.ServerAddr, r)
	if err != nil {
		log.Fatal(err)
	}
}
