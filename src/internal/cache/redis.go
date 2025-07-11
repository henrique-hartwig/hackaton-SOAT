package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"src/internal/config"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
	config *config.Config
}

func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("erro ao conectar Redis: %w", err)
	}

	log.Println("✅ Conectado ao Redis")
	return &RedisClient{
		client: client,
		config: cfg,
	}, nil
}

type VideoCache struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Status      string    `json:"status"`
	UserID      uint      `json:"user_id"`
	URL         string    `json:"url"`
	Duration    int       `json:"duration,omitempty"`
	Thumbnail   string    `json:"thumbnail,omitempty"`
	ProcessedAt time.Time `json:"processed_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserSession struct {
	UserID    uint      `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Roles     []string  `json:"roles"`
	LastLogin time.Time `json:"last_login"`
}

type ProcessingStatus struct {
	VideoID       uint      `json:"video_id"`
	Status        string    `json:"status"`
	Progress      int       `json:"progress"`
	Message       string    `json:"message"`
	EstimatedTime int       `json:"estimated_time"`
	UpdatedAt     time.Time `json:"updated_at"`
}

const (
	VideoKeyPrefix      = "video:"
	UserKeyPrefix       = "user:"
	ProcessingKeyPrefix = "processing:"
	SessionKeyPrefix    = "session:"
)

const (
	VideoTTL      = 1 * time.Hour
	UserTTL       = 30 * time.Minute
	ProcessingTTL = 10 * time.Minute
	SessionTTL    = 24 * time.Hour
)

func (r *RedisClient) SetVideo(ctx context.Context, video *VideoCache) error {
	key := fmt.Sprintf("%s%d", VideoKeyPrefix, video.ID)

	data, err := json.Marshal(video)
	if err != nil {
		return fmt.Errorf("erro ao serializar vídeo: %w", err)
	}

	return r.client.Set(ctx, key, data, VideoTTL).Err()
}

func (r *RedisClient) GetVideo(ctx context.Context, videoID uint) (*VideoCache, error) {
	key := fmt.Sprintf("%s%d", VideoKeyPrefix, videoID)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("erro ao buscar vídeo: %w", err)
	}

	var video VideoCache
	if err := json.Unmarshal([]byte(data), &video); err != nil {
		return nil, fmt.Errorf("erro ao deserializar vídeo: %w", err)
	}

	return &video, nil
}

func (r *RedisClient) SetUserSession(ctx context.Context, sessionID string, user *UserSession) error {
	key := fmt.Sprintf("%s%s", SessionKeyPrefix, sessionID)

	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("erro ao serializar sessão: %w", err)
	}

	return r.client.Set(ctx, key, data, SessionTTL).Err()
}

func (r *RedisClient) GetUserSession(ctx context.Context, sessionID string) (*UserSession, error) {
	key := fmt.Sprintf("%s%s", SessionKeyPrefix, sessionID)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("erro ao buscar sessão: %w", err)
	}

	var user UserSession
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, fmt.Errorf("erro ao deserializar sessão: %w", err)
	}

	return &user, nil
}

func (r *RedisClient) SetProcessingStatus(ctx context.Context, status *ProcessingStatus) error {
	key := fmt.Sprintf("%s%d", ProcessingKeyPrefix, status.VideoID)

	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("erro ao serializar status: %w", err)
	}

	return r.client.Set(ctx, key, data, ProcessingTTL).Err()
}

func (r *RedisClient) GetProcessingStatus(ctx context.Context, videoID uint) (*ProcessingStatus, error) {
	key := fmt.Sprintf("%s%d", ProcessingKeyPrefix, videoID)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("erro ao buscar status: %w", err)
	}

	var status ProcessingStatus
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return nil, fmt.Errorf("erro ao deserializar status: %w", err)
	}

	return &status, nil
}

func (r *RedisClient) InvalidateVideo(ctx context.Context, videoID uint) error {
	key := fmt.Sprintf("%s%d", VideoKeyPrefix, videoID)
	return r.client.Del(ctx, key).Err()
}

func (r *RedisClient) GetVideosByUser(ctx context.Context, userID uint) ([]VideoCache, error) {
	pattern := fmt.Sprintf("%s*", VideoKeyPrefix)

	var videos []VideoCache
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()

		data, err := r.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var video VideoCache
		if err := json.Unmarshal([]byte(data), &video); err != nil {
			continue
		}

		if video.UserID == userID {
			videos = append(videos, video)
		}
	}

	return videos, iter.Err()
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

func (r *RedisClient) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
