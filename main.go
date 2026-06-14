package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

type Server struct {
	db *sql.DB
}

func (s *Server) shortenUrl(w http.ResponseWriter, req *http.Request) {
	type ShortenRequest struct {
		LongUrl string
		Ttl     int64
	}

	var sr ShortenRequest
	err := json.NewDecoder(req.Body).Decode(&sr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	shortCode, err := generateShortCode()
	if err != nil {
		log.Printf("generation of short code failed: %v", err)
		http.Error(w, "generation failed", http.StatusInternalServerError)
		return
	}

	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	fullShortURL := fmt.Sprintf("%s://%s/%s", scheme, req.Host, shortCode)
	fmt.Println(fullShortURL)

	_, insertErr := s.db.Exec(
		"INSERT INTO urls (short_url, long_url, ttl) values ($1, $2, $3)",
		fullShortURL, sr.LongUrl, sr.Ttl,
	)

	fmt.Println(insertErr)

	if insertErr != nil {
		log.Printf("insert failed: %v", insertErr)
		http.Error(w, "insert failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	type ShortenResponse struct {
		ShortUrl string `json:"short_url"`
	}

	if err := json.NewEncoder(w).Encode(ShortenResponse{ShortUrl: fullShortURL}); err != nil {
		log.Printf("response encode failed: %v", err)
	}
}

func generateShortCode() (string, error) {
	const (
		alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		codeLen  = 6
	)
	result := make([]byte, codeLen)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		result[i] = alphabet[n.Int64()]
	}
	return string(result), nil
}

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	server := &Server{db: db}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /url", server.shortenUrl)
	http.ListenAndServe(":8080", mux)
}
