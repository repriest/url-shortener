package main

import (
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"net/url"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", shortenHandler)
	return http.ListenAndServe("localhost:8080", mux)
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "text/plain" {
		log.Println("Invalid content-type, expected string")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if body, err := io.ReadAll(r.Body); err != nil {
		log.Println("Error reading body:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		log.Println("New request received:", string(body))

		if _, err = url.ParseRequestURI(string(body[:])); err != nil {
			log.Println("Could not parse URI: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		shortURI := base64.StdEncoding.EncodeToString(body)
		w.WriteHeader(http.StatusCreated)

		if _, err := w.Write([]byte(r.Host + "/" + shortURI)); err != nil {
			log.Println("Could not write URI: ", shortURI)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	return
}
