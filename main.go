package main

import (
	"encoding/json"
	"net/http"
)

func shortenUrl(w http.ResponseWriter, req *http.Request) {
	type ShortenRequest struct {
		ShortUrl string
		Ttl      int64
	}

	var sr ShortenRequest
	err := json.NewDecoder(req.Body).Decode(&sr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /url", shortenUrl)
	http.ListenAndServe(":8080", mux)
}
