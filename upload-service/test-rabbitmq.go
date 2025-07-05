package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
	"upload-service/internal/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	// Conectar ao RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal("Erro ao conectar RabbitMQ:", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Erro ao abrir canal:", err)
	}
	defer ch.Close()

	// Criar job de teste
	testJob := &models.VideoProcessingJob{
		ID:        fmt.Sprintf("test_%d", time.Now().Unix()),
		VideoID:   123,
		UserID:    456,
		VideoURL:  "http://test.com/video.mp4",
		FileName:  "test_video.mp4",
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Serializar job
	jobBytes, err := json.Marshal(testJob)
	if err != nil {
		log.Fatal("Erro ao serializar job:", err)
	}

	// Enviar para fila
	err = ch.Publish(
		"",                          // exchange
		models.InputProcessingQueue, // routing key
		false,                       // mandatory
		false,                       // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jobBytes,
		},
	)

	if err != nil {
		log.Fatal("Erro ao publicar mensagem:", err)
	}

	fmt.Printf("✅ Mensagem de teste enviada para fila '%s'\n", models.InputProcessingQueue)
	fmt.Printf("📋 Job ID: %s\n", testJob.ID)
	fmt.Printf("🎬 Video ID: %d\n", testJob.VideoID)
	fmt.Printf("👤 User ID: %d\n", testJob.UserID)

	// Verificar quantas mensagens estão na fila
	queue, err := ch.QueueInspect(models.InputProcessingQueue)
	if err != nil {
		log.Printf("⚠️ Erro ao inspecionar fila: %v", err)
	} else {
		fmt.Printf("📊 Mensagens na fila '%s': %d\n", models.InputProcessingQueue, queue.Messages)
	}
}
