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
		OriginalUrl string
		Ttl         int64
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

	_, insertErr := s.db.Exec(
		"INSERT INTO urls (short_code, original_url, ttl) values ($1, $2, $3)",
		shortCode, sr.OriginalUrl, sr.Ttl,
	)

	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	fullShortURL := fmt.Sprintf("%s://%s/%s", scheme, req.Host, shortCode)
	fmt.Println(fullShortURL)

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

func (s *Server) resolveUrl(w http.ResponseWriter, req *http.Request) {
	shortCode := req.PathValue("shortCode")

	var original_url string
	// Need to implement checking ttl
	err := s.db.QueryRow("SELECT original_url FROM urls WHERE short_code = $1", shortCode).Scan(&original_url)
	if err != nil {
		log.Printf("failed to fetch original url: %v", err)
		http.Error(w, "resolving failed", http.StatusInternalServerError)
	}

	http.Redirect(w, req, original_url, 302)
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
	mux.HandleFunc("GET /{shortCode}", server.resolveUrl)
	http.ListenAndServe(":8080", mux)
}
