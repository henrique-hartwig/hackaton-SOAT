package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"upload-service/internal/config"
	"upload-service/internal/middleware"
	"upload-service/internal/queue"
	"upload-service/internal/services/upload"
	"upload-service/internal/services/video_processing"
	"upload-service/internal/storage"

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

	rabbitMQClient, err := queue.NewRabbitMQClient(cfg)
	if err != nil {
		log.Fatal("Erro ao conectar RabbitMQ:", err)
	}
	defer rabbitMQClient.Close()

	// Criar publisher
	publisher := queue.NewPublisher(rabbitMQClient.GetChannel())

	// Criar processor
	processor := video_processing.NewProcessor()

	// Criar consumer
	consumer := queue.NewConsumer(rabbitMQClient.GetChannel(), processor, minioClient)

	// Contexto para graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Iniciar consumer em background
	go func() {
		if err := consumer.StartProcessing(ctx); err != nil {
			log.Printf("Erro no consumer de processamento: %v", err)
		}
	}()

	// Configurar graceful shutdown
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

	// Endpoint de upload (protegido por autenticação)
	router.POST("/upload/video", middleware.AuthMiddleware(), func(c *gin.Context) {
		upload.HandleVideoUpload(c, minioClient, publisher)
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Printf("🚀 Serviço de Upload iniciado na porta %s", cfg.ServerPort)
	log.Fatal(router.Run(":" + cfg.ServerPort))
}
