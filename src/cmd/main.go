package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"src/internal/cache"
	"src/internal/config"
	"src/internal/middleware"
	"src/internal/queue"
	"src/internal/services/upload"
	"src/internal/services/video_processing"
	"src/internal/storage"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	minioClient, err := storage.NewMinioClient(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucket,
	)
	if err != nil {
		log.Fatal("Erro ao conectar MinIO:", err)
	}

	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		log.Fatal("Erro ao conectar Redis:", err)
	}
	defer redisClient.Close()

	rabbitMQClient, err := queue.NewRabbitMQClient(cfg)
	if err != nil {
		log.Fatal("Erro ao conectar RabbitMQ:", err)
	}
	defer rabbitMQClient.Close()

	publisher := queue.NewPublisher(rabbitMQClient.GetChannel())

	processor := video_processing.NewProcessorWithMinIO(minioClient)

	consumer := queue.NewConsumer(rabbitMQClient.GetChannel(), processor, minioClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := consumer.StartProcessing(ctx); err != nil {
			log.Printf("Erro no consumer de processamento: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("🛑 Recebido sinal de shutdown, encerrando...")
		cancel()
	}()

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	router.POST("/upload/video", middleware.AuthMiddleware(), func(c *gin.Context) {
		upload.HandleVideoUpload(c, minioClient, publisher)
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Printf("🚀 Serviço de Upload iniciado na porta %s", cfg.ServerPort)
	log.Fatal(router.Run(":" + cfg.ServerPort))
}
