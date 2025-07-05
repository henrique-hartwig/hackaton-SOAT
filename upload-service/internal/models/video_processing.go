package models

import "time"

type VideoProcessingJob struct {
	ID        string    `json:"id"`
	VideoID   uint      `json:"video_id"`
	UserID    uint      `json:"user_id"`
	VideoURL  string    `json:"video_url"`
	FileName  string    `json:"file_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type VideoProcessingResult struct {
	JobID       string    `json:"job_id"`
	VideoID     uint      `json:"video_id"`
	Status      string    `json:"status"`
	Message     string    `json:"message,omitempty"`
	ProcessedAt time.Time `json:"processed_at"`
}

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

const (
	InputProcessingQueue  = "input_processing_queue"
	ProcessingResultQueue = "processing_result_queue"
)
