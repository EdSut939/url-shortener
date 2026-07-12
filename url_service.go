package main

import "context"

type UrlService struct {
	repo UrlRepository
}

func NewUrlService(repo UrlRepository) *UrlService {
	return &UrlService{repo: repo}
}

func (s *UrlService) InsertUrl(ctx context.Context, shortCode, originalUrl string, ttl *int64) error {
	return s.repo.Insert(ctx, shortCode, originalUrl, ttl)
}

func (s *UrlService) GetOriginalUrlByShortCode(ctx context.Context, shortCode string) (UrlRecord, error) {
	return s.repo.GetByShortCode(ctx, shortCode)
}

func (s *UrlService) InrementVisitCount(ctx context.Context, id int64) error {
	return s.repo.IncrementVisits(ctx, id)
}
