package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/repriest/url-shortener/cmd/shortener/config"
	"github.com/repriest/url-shortener/internal/app/logger"
	baseurl "github.com/repriest/url-shortener/internal/app/url"
	"io"
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

	// chi router cfg
	r := chi.NewRouter()
	r.Post("/", logger.RequestLogger(shortenHandler(cfg)))
	r.Get("/{id}", logger.ResponseLogger(expandHandler(cfg)))

	return http.ListenAndServe(cfg.ServerAddr, r)
}

func shortenHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// read body
		body, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			log.Println("Error reading body:", err)
			http.Error(w, "Could not read body", http.StatusBadRequest)
			return
		}

		// check if url is empty
		longURL := string(body)
		if longURL == "" {
			w.WriteHeader(http.StatusCreated)
			return
		}

		// shorten URL
		shortURL, err := baseurl.ShortenURL(longURL)
		if err != nil {
			log.Println("Could not shorten URL:", err)
			http.Error(w, "Could not shorten URL", http.StatusBadRequest)
			return
		}

		// write shortened URL
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write([]byte(cfg.BaseURL + "/" + shortURL)); err != nil {
			log.Println("Could not write URL: ", shortURL)
			http.Error(w, "Could not write URL", http.StatusInternalServerError)
			return
		}
	}
}

func expandHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortURL := chi.URLParam(r, "id")
		longURL, err := baseurl.ExpandURL(shortURL)
		if err != nil {
			log.Println("Could not decode URL: ", err)
			http.Error(w, "Could not decode URL", http.StatusBadRequest)
		}
		http.Redirect(w, r, string(longURL), http.StatusTemporaryRedirect)
	}
}
