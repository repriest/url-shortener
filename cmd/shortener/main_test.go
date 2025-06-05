package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_encodeHandler(t *testing.T) {
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
			name:       "Normal url",
			method:     http.MethodPost,
			body:       "https://google.com",
			response:   myScheme + myAddr + "/" + "aHR0cHM6Ly9nb29nbGUuY29t",
			statusCode: http.StatusCreated,
		},
		{
			name:       "Malformed url",
			method:     http.MethodPost,
			body:       "badurl!@#$",
			response:   "Could not parse URI\n",
			statusCode: http.StatusBadRequest,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, myAddr, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			encodeHandler(rec, req)

			resp := rec.Result()
			respBody, err := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tc.statusCode, resp.StatusCode)
			assert.Equal(t, tc.response, string(respBody))

		})
	}
}

func TestDecodeHandler(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/{id}", decodeHandler)

	tt := []struct {
		name       string
		method     string
		path       string
		location   string
		statusCode int
	}{
		//{
		//	name:       "Empty path",
		//	method:     "GET",
		//	path:       "/",
		//	statusCode: http.StatusMethodNotAllowed,
		//},
		{
			name:       "Normal url",
			method:     "GET",
			path:       "/aHR0cHM6Ly9nb29nbGUuY29t",
			location:   "https://google.com",
			statusCode: http.StatusTemporaryRedirect,
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
