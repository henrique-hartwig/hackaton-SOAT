package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	apiBaseURL  string
}

// NewConsumer cria um novo consumer
func NewConsumer(channel *amqp.Channel, processor *video_processing.Processor, minioClient *storage.MinioClient) *Consumer {
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "http://localhost:8000"
	}

	return &Consumer{
		channel:     channel,
		processor:   processor,
		minioClient: minioClient,
		apiBaseURL:  apiBaseURL,
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

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-msgs:
			go c.handleMessage(msg)
		}
	}
}

// handleMessage processa uma mensagem individual com retry
func (c *Consumer) handleMessage(msg amqp.Delivery) {
	var job models.VideoProcessingJob
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		log.Printf("❌ Erro ao deserializar job: %v", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("🎬 Processando job: VideoID=%d, UserID=%d", job.VideoID, job.UserID)

	// Tentar processar com retry (3 tentativas)
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("🔄 Tentativa %d/%d para VideoID=%d", attempt, maxRetries, job.VideoID)

		err := c.processAndSaveVideo(&job)
		if err == nil {
			// Sucesso - atualizar status para completed
			if updateErr := c.updateVideoStatus(job.VideoID, models.StatusCompleted, "Vídeo processado com sucesso"); updateErr != nil {
				log.Printf("⚠️ Erro ao atualizar status para completed: %v", updateErr)
			}
			msg.Ack(false)
			return
		}

		log.Printf("❌ Tentativa %d falhou para VideoID=%d: %v", attempt, job.VideoID, err)

		if attempt < maxRetries {
			// Aguardar antes da próxima tentativa (backoff exponencial)
			waitTime := time.Duration(attempt*attempt) * time.Second
			log.Printf("⏳ Aguardando %v antes da próxima tentativa...", waitTime)
			time.Sleep(waitTime)
		}
	}

	// Todas as tentativas falharam - atualizar status para failed
	log.Printf("💥 Todas as %d tentativas falharam para VideoID=%d", maxRetries, job.VideoID)
	if updateErr := c.updateVideoStatus(job.VideoID, models.StatusFailed, "Processamento falhou após 3 tentativas"); updateErr != nil {
		log.Printf("⚠️ Erro ao atualizar status para failed: %v", updateErr)
	}

	msg.Ack(false) // Não fazer requeue, já tentamos 3 vezes
}

// updateVideoStatus atualiza o status do vídeo via API
func (c *Consumer) updateVideoStatus(videoID uint, status, message string) error {
	// Primeiro, buscar o vídeo atual para obter title e url
	getURL := fmt.Sprintf("%s/api/v1/videos/%d", c.apiBaseURL, videoID)
	getReq, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return fmt.Errorf("erro ao criar requisição GET: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	getResp, err := client.Do(getReq)
	if err != nil {
		return fmt.Errorf("erro ao buscar vídeo: %w", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		return fmt.Errorf("erro ao buscar vídeo: API retornou status %d", getResp.StatusCode)
	}

	// Decodificar resposta para obter dados atuais do vídeo
	var videoData map[string]interface{}
	if err := json.NewDecoder(getResp.Body).Decode(&videoData); err != nil {
		return fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	// Mapear status do upload-service para o status da API
	var apiStatus string
	switch status {
	case models.StatusPending:
		apiStatus = "pending"
	case models.StatusProcessing:
		apiStatus = "pending" // Manter como pending durante processamento
	case models.StatusCompleted:
		apiStatus = "processed"
	case models.StatusFailed:
		apiStatus = "failed"
	default:
		apiStatus = "pending"
	}

	// Preparar dados para update (manter title e url originais, atualizar apenas status)
	updateData := map[string]interface{}{
		"title":  videoData["title"],
		"url":    videoData["url"],
		"status": apiStatus,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("erro ao serializar dados: %w", err)
	}

	// Chamar API para atualizar vídeo
	url := fmt.Sprintf("%s/api/v1/videos/%d", c.apiBaseURL, videoID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("erro ao criar requisição: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao fazer requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API retornou status %d", resp.StatusCode)
	}

	log.Printf("✅ Status atualizado para VideoID=%d: %s", videoID, apiStatus)
	return nil
}

// processAndSaveVideo processa o vídeo e salva o resultado no MinIO
func (c *Consumer) processAndSaveVideo(job *models.VideoProcessingJob) error {
	// Atualizar status para processing
	if err := c.updateVideoStatus(job.VideoID, models.StatusProcessing, "Processando vídeo..."); err != nil {
		log.Printf("⚠️ Erro ao atualizar status para processing: %v", err)
	}

	// Processar o vídeo
	result := c.processor.ProcessVideo(job)

	// Se o processamento foi bem-sucedido, salvar no MinIO
	if result.Status == models.StatusCompleted {
		return c.saveProcessedVideo(job)
	}

	return fmt.Errorf("processamento falhou: %s", result.Message)
}

// saveProcessedVideo salva o vídeo processado no MinIO
func (c *Consumer) saveProcessedVideo(job *models.VideoProcessingJob) error {
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
