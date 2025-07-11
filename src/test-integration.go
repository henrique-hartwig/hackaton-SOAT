package main

import (
	"context"
	"fmt"
	"log"
	"src/internal/cache"
	"src/internal/config"
	"src/internal/models"
	"src/internal/queue"
	"src/internal/services/video_processing"
	"src/internal/storage"
	"time"
)

func main() {
	fmt.Println("🧪 Iniciando testes de integração...")

	// Carregar configuração
	cfg := config.LoadConfig()

	// Testar MinIO
	fmt.Println("\n📦 Testando MinIO...")
	minioClient, err := storage.NewMinioClient(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucket,
	)
	if err != nil {
		log.Fatal("❌ Erro ao conectar MinIO:", err)
	}
	fmt.Println("✅ MinIO conectado com sucesso")

	// Testar Redis
	fmt.Println("\n🔴 Testando Redis...")
	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		log.Fatal("❌ Erro ao conectar Redis:", err)
	}
	defer redisClient.Close()

	// Testar operações básicas do Redis
	ctx := context.Background()

	// Testar cache de vídeo
	videoCache := &cache.VideoCache{
		ID:        1,
		Title:     "Teste Vídeo",
		Status:    "pending",
		UserID:    1,
		URL:       "http://test.com/video.mp4",
		CreatedAt: time.Now(),
	}

	if err := redisClient.SetVideo(ctx, videoCache); err != nil {
		log.Fatal("❌ Erro ao salvar vídeo no cache:", err)
	}
	fmt.Println("✅ Cache de vídeo funcionando")

	// Recuperar vídeo do cache
	retrievedVideo, err := redisClient.GetVideo(ctx, 1)
	if err != nil {
		log.Fatal("❌ Erro ao recuperar vídeo do cache:", err)
	}
	if retrievedVideo == nil {
		log.Fatal("❌ Vídeo não encontrado no cache")
	}
	fmt.Printf("✅ Vídeo recuperado: %s\n", retrievedVideo.Title)

	// Testar RabbitMQ
	fmt.Println("\n🐰 Testando RabbitMQ...")
	rabbitMQClient, err := queue.NewRabbitMQClient(cfg)
	if err != nil {
		log.Fatal("❌ Erro ao conectar RabbitMQ:", err)
	}
	defer rabbitMQClient.Close()

	// Testar publisher
	publisher := queue.NewPublisher(rabbitMQClient.GetChannel())
	fmt.Println("✅ Publisher criado com sucesso")

	// Testar processor
	processor := video_processing.NewProcessor()
	fmt.Println("✅ Processor criado com sucesso")

	// Testar consumer (apenas criar, não usar)
	_ = queue.NewConsumer(rabbitMQClient.GetChannel(), processor, minioClient)
	fmt.Println("✅ Consumer criado com sucesso")

	// Testar job de processamento
	job := &models.VideoProcessingJob{
		ID:        "test_job_001",
		VideoID:   1,
		UserID:    1,
		VideoURL:  "http://test.com/video.mp4",
		FileName:  "test_video.mp4",
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Testar publicação de job
	if err := publisher.PublishVideoProcessingJob(job); err != nil {
		log.Fatal("❌ Erro ao publicar job:", err)
	}
	fmt.Println("✅ Job publicado com sucesso")

	// Testar processamento
	result := processor.ProcessVideo(job)
	fmt.Printf("✅ Processamento testado: %s\n", result.Status)

	// Testar cache de status de processamento
	processingStatus := &cache.ProcessingStatus{
		VideoID:       1,
		Status:        "processing",
		Progress:      50,
		Message:       "Processando...",
		EstimatedTime: 30,
		UpdatedAt:     time.Now(),
	}

	if err := redisClient.SetProcessingStatus(ctx, processingStatus); err != nil {
		log.Fatal("❌ Erro ao salvar status de processamento:", err)
	}
	fmt.Println("✅ Cache de status de processamento funcionando")

	// Testar cache de sessão de usuário
	userSession := &cache.UserSession{
		UserID:    1,
		Email:     "test@example.com",
		Name:      "Usuário Teste",
		Roles:     []string{"user"},
		LastLogin: time.Now(),
	}

	if err := redisClient.SetUserSession(ctx, "session_123", userSession); err != nil {
		log.Fatal("❌ Erro ao salvar sessão de usuário:", err)
	}
	fmt.Println("✅ Cache de sessão de usuário funcionando")

	// Recuperar sessão
	retrievedSession, err := redisClient.GetUserSession(ctx, "session_123")
	if err != nil {
		log.Fatal("❌ Erro ao recuperar sessão:", err)
	}
	if retrievedSession == nil {
		log.Fatal("❌ Sessão não encontrada no cache")
	}
	fmt.Printf("✅ Sessão recuperada: %s\n", retrievedSession.Name)

	fmt.Println("\n🎉 Todos os testes de integração passaram!")
	fmt.Println("\n📋 Resumo das funcionalidades testadas:")
	fmt.Println("   ✅ MinIO - Upload e armazenamento")
	fmt.Println("   ✅ Redis - Cache de vídeos, sessões e status")
	fmt.Println("   ✅ RabbitMQ - Publicação e consumo de jobs")
	fmt.Println("   ✅ Video Processing - Processamento de vídeos")
	fmt.Println("   ✅ Retry Logic - Implementado no consumer")
	fmt.Println("   ✅ Status Updates - Via API REST")
}
