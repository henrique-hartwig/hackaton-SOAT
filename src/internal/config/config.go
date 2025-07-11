package config

import "os"

type Config struct {
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string

	RabbitMQURL string

	RedisURL string

	APIBaseURL string

	ServerPort string
}

func LoadConfig() *Config {
	return &Config{
		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "minio:9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:    getEnv("MINIO_BUCKET", "videos"),

		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),

		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),

		APIBaseURL: getEnv("API_BASE_URL", "http://localhost:8080"),

		ServerPort: getEnv("SERVER_PORT", "8081"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
