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
	"time"

	"github.com/lib/pq"
)

type Server struct {
	service *UrlService
}

func (s *Server) shortenUrl(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var sr ShortenRequest

	err := json.NewDecoder(req.Body).Decode(&sr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	const maxRetries = 3

	//TODO: cover with tests
	for attempt := range maxRetries {
		shortCode, err := generateShortCode()
		if err != nil {
			log.Printf("generation of short code failed: %v", err)
			http.Error(w, "generation failed", http.StatusInternalServerError)
			return
		}

		if err := s.service.InsertUrl(ctx, shortCode, sr.OriginalUrl, sr.Ttl); err != nil {
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				log.Printf("short code collision, retry %d/%d", attempt+1, maxRetries)
				continue
			}
			log.Printf("insert failed: %v", err)
			http.Error(w, "insert failed", http.StatusInternalServerError)
			return
		}

		scheme := "http"
		if req.TLS != nil {
			scheme = "https"
		}
		fullShortURL := fmt.Sprintf("%s://%s/%s", scheme, req.Host, shortCode)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(ShortenResponse{ShortUrl: fullShortURL}); err != nil {
			log.Printf("response encode failed: %v", err)
		}
		return
	}

	log.Printf("exhausted %d retries generating a unique short code", maxRetries)
	http.Error(w, "insert failed", http.StatusInternalServerError)
}

func (s *Server) resolveUrl(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	shortCode := req.PathValue("shortCode")

	record, err := s.service.GetOriginalUrlByShortCode(ctx, shortCode)
	if err != nil {
		log.Printf("failed to fetch original url: %v", err)
		http.Error(w, "resolving failed", http.StatusInternalServerError)
		return
	}

	if record.Ttl != nil && time.Now().After(record.CreatedAt.Add(time.Second*time.Duration(*record.Ttl))) {
		log.Printf("Link has expired")
		http.Error(w, "Link has expired", http.StatusBadRequest)
		return
	}

	if err := s.service.InrementVisitCount(ctx, record.Id); err != nil {
		log.Printf("failed to update visits: %v", err)
		http.Error(w, "something went wrong, try again", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, record.OriginalUrl, 302)
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

	repo := NewPostgresUrlRepository(db)
	service := NewUrlService(repo)
	server := &Server{service: service}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /url", server.shortenUrl)
	mux.HandleFunc("GET /{shortCode}", server.resolveUrl)
	log.Fatal(http.ListenAndServe(":8080", mux))
}
