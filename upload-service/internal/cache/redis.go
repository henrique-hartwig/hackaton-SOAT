package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"upload-service/internal/config"

	"github.com/redis/go-redis/v9"
)

// RedisClient wrapper para operações Redis
type RedisClient struct {
	client *redis.Client
	config *config.Config
}

// NewRedisClient cria nova conexão Redis
func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Testar conexão
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

// VideoCache estrutura para cache de vídeos
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

// UserSession estrutura para cache de sessão
type UserSession struct {
	UserID    uint      `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Roles     []string  `json:"roles"`
	LastLogin time.Time `json:"last_login"`
}

// ProcessingStatus estrutura para cache de status de processamento
type ProcessingStatus struct {
	VideoID       uint      `json:"video_id"`
	Status        string    `json:"status"`
	Progress      int       `json:"progress"`
	Message       string    `json:"message"`
	EstimatedTime int       `json:"estimated_time"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Cache keys constants
const (
	VideoKeyPrefix      = "video:"
	UserKeyPrefix       = "user:"
	ProcessingKeyPrefix = "processing:"
	SessionKeyPrefix    = "session:"
)

// TTL constants
const (
	VideoTTL      = 1 * time.Hour    // Metadados de vídeo
	UserTTL       = 30 * time.Minute // Dados de usuário
	ProcessingTTL = 10 * time.Minute // Status de processamento
	SessionTTL    = 24 * time.Hour   // Sessões de usuário
)

// SetVideo armazena dados de vídeo no cache
func (r *RedisClient) SetVideo(ctx context.Context, video *VideoCache) error {
	key := fmt.Sprintf("%s%d", VideoKeyPrefix, video.ID)

	data, err := json.Marshal(video)
	if err != nil {
		return fmt.Errorf("erro ao serializar vídeo: %w", err)
	}

	return r.client.Set(ctx, key, data, VideoTTL).Err()
}

// GetVideo recupera dados de vídeo do cache
func (r *RedisClient) GetVideo(ctx context.Context, videoID uint) (*VideoCache, error) {
	key := fmt.Sprintf("%s%d", VideoKeyPrefix, videoID)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("erro ao buscar vídeo: %w", err)
	}

	var video VideoCache
	if err := json.Unmarshal([]byte(data), &video); err != nil {
		return nil, fmt.Errorf("erro ao deserializar vídeo: %w", err)
	}

	return &video, nil
}

// SetUserSession armazena sessão de usuário
func (r *RedisClient) SetUserSession(ctx context.Context, sessionID string, user *UserSession) error {
	key := fmt.Sprintf("%s%s", SessionKeyPrefix, sessionID)

	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("erro ao serializar sessão: %w", err)
	}

	return r.client.Set(ctx, key, data, SessionTTL).Err()
}

// GetUserSession recupera sessão de usuário
func (r *RedisClient) GetUserSession(ctx context.Context, sessionID string) (*UserSession, error) {
	key := fmt.Sprintf("%s%s", SessionKeyPrefix, sessionID)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Sessão não encontrada
		}
		return nil, fmt.Errorf("erro ao buscar sessão: %w", err)
	}

	var user UserSession
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, fmt.Errorf("erro ao deserializar sessão: %w", err)
	}

	return &user, nil
}

// SetProcessingStatus armazena status de processamento
func (r *RedisClient) SetProcessingStatus(ctx context.Context, status *ProcessingStatus) error {
	key := fmt.Sprintf("%s%d", ProcessingKeyPrefix, status.VideoID)

	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("erro ao serializar status: %w", err)
	}

	return r.client.Set(ctx, key, data, ProcessingTTL).Err()
}

// GetProcessingStatus recupera status de processamento
func (r *RedisClient) GetProcessingStatus(ctx context.Context, videoID uint) (*ProcessingStatus, error) {
	key := fmt.Sprintf("%s%d", ProcessingKeyPrefix, videoID)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Status não encontrado
		}
		return nil, fmt.Errorf("erro ao buscar status: %w", err)
	}

	var status ProcessingStatus
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return nil, fmt.Errorf("erro ao deserializar status: %w", err)
	}

	return &status, nil
}

// InvalidateVideo remove vídeo do cache
func (r *RedisClient) InvalidateVideo(ctx context.Context, videoID uint) error {
	key := fmt.Sprintf("%s%d", VideoKeyPrefix, videoID)
	return r.client.Del(ctx, key).Err()
}

// GetVideosByUser busca vídeos de um usuário (usando padrão)
func (r *RedisClient) GetVideosByUser(ctx context.Context, userID uint) ([]VideoCache, error) {
	// Usar SCAN para buscar por padrão (mais eficiente que KEYS)
	pattern := fmt.Sprintf("%s*", VideoKeyPrefix)

	var videos []VideoCache
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()

		data, err := r.client.Get(ctx, key).Result()
		if err != nil {
			continue // Pular erros
		}

		var video VideoCache
		if err := json.Unmarshal([]byte(data), &video); err != nil {
			continue // Pular erros de desserialização
		}

		if video.UserID == userID {
			videos = append(videos, video)
		}
	}

	return videos, iter.Err()
}

// Close fecha a conexão Redis
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Health verifica se Redis está funcionando
func (r *RedisClient) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
