package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type mockRepo struct {
	records map[string]UrlRecord
	nextID  int64
}

var _ UrlRepository = (*mockRepo)(nil)

func newMockRepo() *mockRepo {
	return &mockRepo{
		records: make(map[string]UrlRecord),
		nextID:  1,
	}
}

func (m *mockRepo) Insert(ctx context.Context, shortCode, originalUrl string, ttl *int64) error {
	m.records[shortCode] = UrlRecord{
		Id:          m.nextID,
		OriginalUrl: originalUrl,
		Ttl:         ttl,
		CreatedAt:   time.Now(),
		Visits:      0,
	}
	m.nextID++
	return nil
}

func (m *mockRepo) GetByShortCode(ctx context.Context, shortCode string) (UrlRecord, error) {
	rec, ok := m.records[shortCode]
	if !ok {
		return UrlRecord{}, sql.ErrNoRows
	}
	return rec, nil
}

func (m *mockRepo) IncrementVisits(ctx context.Context, id int64) error {
	for code, rec := range m.records {
		if rec.Id == id {
			rec.Visits++
			m.records[code] = rec
			return nil
		}
	}
	return nil
}

func newTestServer() (*Server, *mockRepo) {
	repo := newMockRepo()
	svc := NewUrlService(repo)
	return &Server{service: svc}, repo
}

func TestShortenUrl_ValidRequest(t *testing.T) {
	server, _ := newTestServer()

	body := ShortenRequest{OriginalUrl: "https://example.com"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/url", bytes.NewReader(b))
	w := httptest.NewRecorder()

	server.shortenUrl(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var sr ShortenResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !strings.HasPrefix(sr.ShortUrl, "http://") {
		t.Errorf("expected http:// prefix, got %s", sr.ShortUrl)
	}
}

func TestShortenUrl_EmptyBody(t *testing.T) {
	server, _ := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/url", bytes.NewReader([]byte("")))
	w := httptest.NewRecorder()

	server.shortenUrl(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestShortenUrl_InvalidJSON(t *testing.T) {
	server, _ := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/url", bytes.NewReader([]byte("{bad")))
	w := httptest.NewRecorder()

	server.shortenUrl(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestResolveUrl_Found(t *testing.T) {
	server, repo := newTestServer()

	shortCode := "abc123"
	repo.Insert(context.Background(), shortCode, "https://example.com", ptr(int64(86400)))

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	req.SetPathValue("shortCode", "abc123")
	w := httptest.NewRecorder()

	server.resolveUrl(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}

	urlRecord, _ := repo.GetByShortCode(context.Background(), shortCode)
	if urlRecord.Visits != 1 {
		t.Fatalf("expected 1 visit count, got %d", urlRecord.Visits)
	}

}

func TestResolveUrl_NotFound(t *testing.T) {
	server, _ := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	req.SetPathValue("shortCode", "nonexistent")
	w := httptest.NewRecorder()

	server.resolveUrl(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestResolveUrl_Expired(t *testing.T) {
	server, repo := newTestServer()

	repo.Insert(context.Background(), "exp123", "https://example.com", ptr(int64(0)))

	req := httptest.NewRequest(http.MethodGet, "/exp123", nil)
	req.SetPathValue("shortCode", "exp123")
	w := httptest.NewRecorder()

	server.resolveUrl(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGenerateShortCode(t *testing.T) {
	code, err := generateShortCode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("expected length 6, got %d", len(code))
	}
	for _, c := range code {
		if !('a' <= c && c <= 'z') && !('A' <= c && c <= 'Z') && !('0' <= c && c <= '9') {
			t.Errorf("unexpected char %c in code", c)
		}
	}
}

func ptr(v int64) *int64 { return &v }
