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
	mux.HandleFunc("POST /", encodeHandler)
	mux.HandleFunc("GET /{id}", decodeHandler)
	return http.ListenAndServe("localhost:8080", mux)
}

func encodeHandler(w http.ResponseWriter, r *http.Request) {
	//if r.Header.Get("Content-Type") != "text/plain" {
	//	log.Println("Invalid content-type, expected string")
	//	w.WriteHeader(http.StatusBadRequest)
	//	return
	//}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading body:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println("New request received:", string(body))

	if _, err = url.ParseRequestURI(string(body[:])); err != nil {
		if len(body) == 0 {
			w.WriteHeader(http.StatusCreated)
			return
		}
		log.Println("Could not parse URI: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	shortURI := base64.StdEncoding.EncodeToString(body)
	w.WriteHeader(http.StatusCreated)

	if _, err := w.Write([]byte("http://" + r.Host + "/" + shortURI)); err != nil {
		log.Println("Could not write URI: ", shortURI)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func decodeHandler(w http.ResponseWriter, r *http.Request) {
	//if r.Header.Get("Content-Type") != "text/plain" {
	//	log.Println("Invalid content-type, expected string")
	//	w.WriteHeader(http.StatusBadRequest)
	//	return
	//}

	shortURI := r.PathValue("id")
	longURI, err := base64.StdEncoding.DecodeString(shortURI)

	if err != nil {
		log.Println("Could not decode URI: ", err)
	}
	w.Header().Set("Location", string(longURI))
	w.WriteHeader(http.StatusTemporaryRedirect)
}
