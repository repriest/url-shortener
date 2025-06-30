package handlers

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/repriest/url-shortener/internal/config"
	"github.com/repriest/url-shortener/internal/storage"
	"github.com/repriest/url-shortener/internal/urlservice"
	"io"
	"net/http"
)

type Handler struct {
	cfg *config.Config
	st  *storage.Storage
}

func NewHandler(cfg *config.Config, st *storage.Storage) *Handler {
	return &Handler{cfg: cfg, st: st}
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

func (h *Handler) ShortenHandler(w http.ResponseWriter, r *http.Request) {
	// read body
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, "Could not read body", http.StatusBadRequest)
		return
	}

	// check if URL is empty
	longURL := string(body)
	if longURL == "" {
		w.WriteHeader(http.StatusCreated)
		return
	}

	// shorten URL
	shortURL, err := urlservice.ShortenURL(longURL)
	if err != nil {
		http.Error(w, "Could not shorten URL", http.StatusBadRequest)
		return
	}

	// write shortened URL
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(h.cfg.BaseURL + "/" + shortURL)); err != nil {
		http.Error(w, "Could not write URL", http.StatusInternalServerError)
		return
	}

	// append entry uuid - shorturl - longurl to file
	err = h.st.AddNewEntry(shortURL, longURL)
	if err != nil {
		http.Error(w, "Could not write URL", http.StatusInternalServerError)
	}
}

func (h *Handler) ExpandHandler(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "id")
	longURL, err := urlservice.ExpandURL(shortURL)
	if err != nil {
		http.Error(w, "Could not decode URL", http.StatusBadRequest)
	}

	http.Redirect(w, r, string(longURL), http.StatusTemporaryRedirect)
}

func (h *Handler) ShortenJSONHandler(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest

	// read body
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// parse body json
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// shorten URL
	shortURL, err := urlservice.ShortenURL(req.URL)
	if err != nil {
		http.Error(w, "Could not shorten URL", http.StatusBadRequest)
		return
	}

	// write shortened URL
	resp := ShortenResponse{Result: h.cfg.BaseURL + "/" + shortURL}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	respJSON, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
		return
	}
	w.Write(respJSON)

	// append entry uuid - shorturl - longurl to file
	err = h.st.AddNewEntry(shortURL, req.URL)
	if err != nil {
		http.Error(w, "Could not write URL", http.StatusInternalServerError)
	}
}
