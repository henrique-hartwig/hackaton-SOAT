package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"upload-service/internal/models"
	"upload-service/internal/services/video_processing"
	"upload-service/internal/storage"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer representa o consumer de mensagens do RabbitMQ
type Consumer struct {
	channel     *amqp.Channel
	processor   *video_processing.Processor
	minioClient *storage.MinioClient
}

// NewConsumer cria um novo consumer
func NewConsumer(channel *amqp.Channel, processor *video_processing.Processor, minioClient *storage.MinioClient) *Consumer {
	return &Consumer{
		channel:     channel,
		processor:   processor,
		minioClient: minioClient,
	}
}

func (c *Consumer) StartProcessing(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		models.InputProcessingQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Println("🎬 Consumer iniciado - aguardando jobs de processamento...")

	// Processar mensagens
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-msgs:
			go c.handleMessage(msg)
		}
	}
}

// handleMessage processa uma mensagem individual
func (c *Consumer) handleMessage(msg amqp.Delivery) {
	var job models.VideoProcessingJob
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		log.Printf("❌ Erro ao deserializar job: %v", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("🎬 Processando job: VideoID=%d, UserID=%d", job.VideoID, job.UserID)

	// Processar o vídeo e salvar resultado no MinIO
	if err := c.processAndSaveVideo(&job); err != nil {
		log.Printf("❌ Erro ao processar vídeo: %v", err)
		msg.Nack(false, true) // Requeue para tentar novamente
		return
	}

	// Acknowledgment da mensagem
	msg.Ack(false)
}

// processAndSaveVideo processa o vídeo e salva o resultado no MinIO
func (c *Consumer) processAndSaveVideo(job *models.VideoProcessingJob) error {
	// Processar o vídeo
	result := c.processor.ProcessVideo(job)

	// Se o processamento foi bem-sucedido, salvar no MinIO
	if result.Status == models.StatusCompleted {
		return c.saveProcessedVideo(job, result)
	}

	return fmt.Errorf("processamento falhou: %s", result.Message)
}

// saveProcessedVideo salva o vídeo processado no MinIO
func (c *Consumer) saveProcessedVideo(job *models.VideoProcessingJob, result *video_processing.ProcessingResult) error {
	// Gerar nome do arquivo processado
	fileName := strings.TrimSuffix(job.FileName, filepath.Ext(job.FileName))
	processedFileName := fmt.Sprintf("%s_processed.mp4", fileName)

	// Caminho no MinIO: {user_id}/outputs/{processed_file_name}
	objectName := fmt.Sprintf("%d/outputs/%s", job.UserID, processedFileName)

	// Aqui você implementaria a lógica real de processamento
	// Por enquanto, vamos simular criando um arquivo de exemplo
	processedContent := fmt.Sprintf("Processed video content for %s", job.FileName)

	// Salvar no MinIO
	err := c.minioClient.UploadString(context.Background(), objectName, processedContent)
	if err != nil {
		return fmt.Errorf("erro ao salvar vídeo processado: %w", err)
	}

	log.Printf("✅ Vídeo processado salvo: %s", objectName)
	return nil
}
