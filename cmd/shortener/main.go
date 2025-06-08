package main

import (
	"encoding/base64"
	"github.com/repriest/url-shortener/cmd/shortener/config"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

var myScheme = "http://"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.NewConfig()
	r := chi.NewRouter()
	r.Post("/", encodeHandler(cfg))
	r.Get("/{id}", decodeHandler(cfg))
	return http.ListenAndServe(cfg.ServerAddr, r)
}

func encodeHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		defer r.Body.Close()

		if err != nil {
			log.Println("Error reading body:", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}

		if _, err = url.ParseRequestURI(string(body[:])); err != nil {
			if len(body) == 0 {
				w.WriteHeader(http.StatusCreated)
				return
			}
			log.Println("Could not parse URI: ", err)
			http.Error(w, "Could not parse URI", http.StatusBadRequest)
			return
		}

		shortURI := base64.StdEncoding.EncodeToString(body)
		w.WriteHeader(http.StatusCreated)

		if _, err := w.Write([]byte(cfg.BaseURL + "/" + shortURI)); err != nil {
			log.Println("Could not write URI: ", shortURI)
			http.Error(w, "Could not write URI", http.StatusInternalServerError)
			return
		}
	}
}

func decodeHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortURI := chi.URLParam(r, "id")
		longURI, err := base64.StdEncoding.DecodeString(shortURI)
		if err != nil {
			log.Println("Could not decode URI: ", err)
			http.Error(w, "Could not decode URI", http.StatusBadRequest)
		}
		http.Redirect(w, r, string(longURI), http.StatusTemporaryRedirect)
	}
}
