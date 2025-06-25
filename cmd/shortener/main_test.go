package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/repriest/url-shortener/internal/config"
	"github.com/repriest/url-shortener/internal/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShortenHandler(t *testing.T) {
	cfg := &config.Config{
		ServerAddr: "localhost:8080",
		BaseURL:    "http://localhost:8080",
	}
	h := handlers.NewHandler(cfg)
	tt := []struct {
		name       string
		method     string
		body       string
		response   string
		statusCode int
	}{
		{
			name:       "Empty body",
			method:     http.MethodPost,
			body:       "",
			response:   "",
			statusCode: http.StatusCreated,
		},
		{
			name:       "Valid URL",
			method:     http.MethodPost,
			body:       "https://google.com",
			response:   cfg.BaseURL + "/" + "aHR0cHM6Ly9nb29nbGUuY29t",
			statusCode: http.StatusCreated,
		},
		{
			name:       "Invalid URL",
			method:     http.MethodPost,
			body:       "badurl!@#$",
			response:   "Could not shorten URL\n",
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, cfg.ServerAddr, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			h.ShortenHandler(rec, req)

			resp := rec.Result()
			respBody, err := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tc.statusCode, resp.StatusCode)
			assert.Equal(t, tc.response, string(respBody))

		})
	}
}

func TestExpandHandler(t *testing.T) {
	cfg := &config.Config{
		ServerAddr: "localhost:8080",
		BaseURL:    "http://localhost:8080",
	}
	h := handlers.NewHandler(cfg)
	r := chi.NewRouter()
	r.Get("/{id}", h.ExpandHandler)

	tt := []struct {
		name       string
		method     string
		path       string
		location   string
		statusCode int
	}{
		{
			name:       "Valid URL",
			method:     "GET",
			path:       "/aHR0cHM6Ly9nb29nbGUuY29t",
			location:   "https://google.com",
			statusCode: http.StatusTemporaryRedirect,
		},
		{
			name:       "Invalid URL",
			method:     "GET",
			path:       "/notbase64",
			location:   "/",
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)

			r.ServeHTTP(resp, req)

			assert.Equal(t, tc.statusCode, resp.Code)
			assert.Equal(t, tc.location, resp.Header().Values("Location")[0])
		})
	}
}

func TestShortenJsonHandler(t *testing.T) {
	cfg := &config.Config{
		ServerAddr: "localhost:8080",
		BaseURL:    "http://localhost:8080",
	}
	h := handlers.NewHandler(cfg)

	tt := []struct {
		name        string
		method      string
		body        string
		response    string
		statusCode  int
		contentType string
	}{
		{
			name:        "Valid URL",
			method:      "GET",
			body:        `{"url":"https://google.com"}`,
			response:    `{"result":"http://localhost:8080/aHR0cHM6Ly9nb29nbGUuY29t"}`,
			statusCode:  http.StatusCreated,
			contentType: "application/json",
		},
		{
			name:        "Invalid JSON",
			body:        `{not: json}`,
			response:    "Invalid JSON\n",
			statusCode:  http.StatusBadRequest,
			contentType: "text/plain; charset=utf-8",
		},
		{
			name:        "No URL",
			body:        `{"foo":"bar"}`,
			response:    "URL is required\n",
			statusCode:  http.StatusBadRequest,
			contentType: "text/plain; charset=utf-8",
		},
		{
			name:        "Invalid URL",
			body:        `{"url":"badurl!@#$"}`,
			response:    "Could not shorten URL\n",
			statusCode:  http.StatusBadRequest,
			contentType: "text/plain; charset=utf-8",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			h.ShortenJsonHandler(rec, req)

			resp := rec.Result()
			respBody, err := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tc.statusCode, resp.StatusCode)
			assert.Equal(t, tc.contentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, tc.response, string(respBody))
		})
	}
}
