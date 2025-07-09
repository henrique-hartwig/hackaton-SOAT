package config

import "os"

// Config centraliza todas as configurações da aplicação
type Config struct {
	// MinIO
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string

	// RabbitMQ
	RabbitMQURL string

	// Redis
	RedisURL string

	// API
	APIBaseURL string

	// Server
	ServerPort string
}

// LoadConfig carrega as configurações das variáveis de ambiente
func LoadConfig() *Config {
	return &Config{
		// MinIO
		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "minio:9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:    getEnv("MINIO_BUCKET", "videos"),

		// RabbitMQ
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),

		// Redis
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),

		// API
		APIBaseURL: getEnv("API_BASE_URL", "http://localhost:8080"),

		// Server
		ServerPort: getEnv("SERVER_PORT", "8081"),
	}
}

// getEnv retorna o valor da variável de ambiente ou o valor padrão
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
