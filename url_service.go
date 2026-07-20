package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"

	"github.com/lib/pq"
)

type UrlService struct {
	repo UrlRepository
}

func NewUrlService(repo UrlRepository) *UrlService {
	return &UrlService{repo: repo}
}

func (s *UrlService) InsertUrl(ctx context.Context, sr ShortenRequest, httpScheme string, host string) (string, error) {
	const maxRetries = 3

	//TODO: cover with tests
	for attempt := range maxRetries {
		shortCode, err := generateShortCode()
		if err != nil {
			log.Printf("generation of short code failed: %v", err)
			return "", err
		}

		if err := s.repo.Insert(ctx, shortCode, sr.OriginalUrl, sr.Ttl); err != nil {
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				log.Printf("short code collision, retry %d/%d", attempt+1, maxRetries)
				continue
			}
			log.Printf("insert failed: %v", err)
			return "", err
		}

		fullShortURL := fmt.Sprintf("%s://%s/%s", httpScheme, host, shortCode)

		return fullShortURL, nil
	}

	log.Printf("exhausted %d retries generating a unique short code", maxRetries)
	return "", fmt.Errorf("exhausted %d retries generating a unique short code", maxRetries)
}

func (s *UrlService) GetOriginalUrlByShortCode(ctx context.Context, shortCode string) (UrlRecord, error) {
	return s.repo.GetByShortCode(ctx, shortCode)
}

func (s *UrlService) InrementVisitCount(ctx context.Context, id int64) error {
	return s.repo.IncrementVisits(ctx, id)
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
