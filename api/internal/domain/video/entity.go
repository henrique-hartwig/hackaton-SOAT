package video

import "time"

type Video struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Status    Status    `json:"status"`
	UserID    uint      `json:"id_user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Status string

const (
	StatusPending   Status = "pending"
	StatusProcessed Status = "processed"
	StatusFailed    Status = "failed"
)

func NewVideo(title, url string, userID uint) *Video {
	return &Video{
		Title:     title,
		URL:       url,
		Status:    StatusPending,
		UserID:    userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
