package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/repriest/url-shortener/internal/config"
	"github.com/repriest/url-shortener/internal/handlers"
	"github.com/repriest/url-shortener/internal/logger"
	"log"
	"net/http"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// read args
	cfg, err := config.NewConfig()
	if err != nil {
		return err
	}

	// init logger
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	// handle with chi
	h := handlers.NewHandler(cfg)
	r := chi.NewRouter()
	r.Post("/", logger.RequestLogger(h.ShortenHandler))
	r.Get("/{id}", logger.ResponseLogger(h.ExpandHandler))
	r.Post("/api/shorten", logger.RequestLogger(h.ShortenJsonHandler))

	return http.ListenAndServe(cfg.ServerAddr, r)
}
