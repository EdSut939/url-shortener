package main

import "time"

type ShortenRequest struct {
	OriginalUrl string `json:"originalUrl"`
	Ttl         *int64 `json:"ttl,omitempty"`
}

type ShortenResponse struct {
	ShortUrl string `json:"short_url"`
}

type UrlRecord struct {
	Id          int64
	OriginalUrl string
	Ttl         *int64
	CreatedAt   time.Time
	Visits      int64
}
