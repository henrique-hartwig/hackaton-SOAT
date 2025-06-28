package video

import (
	"time"
)

type Video struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Status string

const (
	StatusPending   Status = "pending"
	StatusProcessed Status = "processed"
	StatusFailed    Status = "failed"
)

func NewVideo(title, url string) *Video {
	return &Video{
		Title:     title,
		URL:       url,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
