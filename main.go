package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
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

	httpScheme := "http"
	if req.TLS != nil {
		httpScheme = "https"
	}

	fullShortUrl, insertErr := s.service.InsertUrl(ctx, sr, httpScheme, req.Host)

	if insertErr != nil {
		http.Error(w, "Url shortening failed", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(ShortenResponse{ShortUrl: fullShortUrl}); err != nil {
		log.Printf("response encode failed: %v", err)
	}
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
