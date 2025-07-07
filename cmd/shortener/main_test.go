package main

import (
	"bytes"
	"compress/gzip"
	"github.com/go-chi/chi/v5"
	"github.com/repriest/url-shortener/internal/config"
	"github.com/repriest/url-shortener/internal/handlers"
	"github.com/repriest/url-shortener/internal/storage"
	"github.com/repriest/url-shortener/internal/zipper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var cfg *config.Config

func TestMain(m *testing.M) {
	var err error
	cfg, err = config.NewConfig()
	if err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestShortenHandler(t *testing.T) {
	st, err := storage.NewRepository(storage.NewMemoryStorage())
	require.NoError(t, err)
	defer st.Close()
	h := handlers.NewHandler(cfg, st)

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
			response:   cfg.BaseURL + "/aHR0cHM6Ly9nb29nbGUuY29t",
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
			req := httptest.NewRequest(tc.method, cfg.ServerAddr, strings.NewReader(tc.body))
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
	st, err := storage.NewRepository(storage.NewMemoryStorage())
	require.NoError(t, err)
	defer st.Close()
	h := handlers.NewHandler(cfg, st)
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
			method:     http.MethodGet,
			path:       "/aHR0cHM6Ly9nb29nbGUuY29t",
			location:   "https://google.com",
			statusCode: http.StatusTemporaryRedirect,
		},
		{
			name:       "Invalid URL",
			method:     http.MethodGet,
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
			assert.Equal(t, tc.location, resp.Header().Get("Location"))
		})
	}
}

func TestShortenJSONHandler(t *testing.T) {
	st, err := storage.NewRepository(storage.NewMemoryStorage())
	require.NoError(t, err)
	defer st.Close()
	h := handlers.NewHandler(cfg, st)

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
			method:      http.MethodPost,
			body:        `{"url":"https://google.com"}`,
			response:    `{"result":"` + cfg.BaseURL + `/aHR0cHM6Ly9nb29nbGUuY29t"}`,
			statusCode:  http.StatusCreated,
			contentType: "application/json",
		},
		{
			name:        "Invalid JSON",
			method:      http.MethodPost,
			body:        `{not: json}`,
			response:    "Invalid JSON\n",
			statusCode:  http.StatusBadRequest,
			contentType: "text/plain; charset=utf-8",
		},
		{
			name:        "No URL",
			method:      http.MethodPost,
			body:        `{"foo":"bar"}`,
			response:    "URL is required\n",
			statusCode:  http.StatusBadRequest,
			contentType: "text/plain; charset=utf-8",
		},
		{
			name:        "Invalid URL",
			method:      http.MethodPost,
			body:        `{"url":"badurl!@#$"}`,
			response:    "Could not shorten URL\n",
			statusCode:  http.StatusBadRequest,
			contentType: "text/plain; charset=utf-8",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/shorten", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			h.ShortenJSONHandler(rec, req)

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

func TestGzipCompression(t *testing.T) {
	st, err := storage.NewRepository(storage.NewMemoryStorage())
	require.NoError(t, err)
	defer st.Close()
	h := handlers.NewHandler(cfg, st)

	handler := zipper.GzipMiddleware(h.ShortenHandler)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	requestBody := "https://google.com"
	successBody := cfg.BaseURL + "/aHR0cHM6Ly9nb29nbGUuY29t"

	// compress test
	t.Run("sends_gzip", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		zb := gzip.NewWriter(buf)
		_, err := zb.Write([]byte(requestBody))
		require.NoError(t, err)
		err = zb.Close()
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodPost, srv.URL, buf)
		r.RequestURI = ""
		r.Header.Set("Content-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, string(b), successBody)
	})

	// decompress test
	t.Run("accepts_gzip", func(t *testing.T) {
		buf := bytes.NewBufferString(requestBody)
		r := httptest.NewRequest(http.MethodPost, srv.URL, buf)
		r.RequestURI = ""
		r.Header.Set("Accept-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		zr, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)

		b, err := io.ReadAll(zr)
		require.NoError(t, err)

		require.Equal(t, string(b), successBody)
	})
}
