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
	"src/internal/models"
	"src/internal/services/video_processing"
	"src/internal/storage"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	channel     *amqp.Channel
	processor   *video_processing.Processor
	minioClient *storage.MinioClient
	apiBaseURL  string
}

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

func (c *Consumer) handleMessage(msg amqp.Delivery) {
	var job models.VideoProcessingJob
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		log.Printf("Erro ao deserializar job: %v", err)
		if nackErr := msg.Nack(false, false); nackErr != nil {
			log.Printf("Erro ao fazer Nack: %v", nackErr)
		}
		return
	}

	log.Printf("🎬 Processando job: VideoID=%d, UserID=%d", job.VideoID, job.UserID)

	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("🔄 Tentativa %d/%d para VideoID=%d", attempt, maxRetries, job.VideoID)

		err := c.processAndSaveVideo(&job)
		if err == nil {
			if updateErr := c.updateVideoStatus(job.VideoID, models.StatusCompleted, job.AuthToken); updateErr != nil {
				log.Printf("Erro ao atualizar status para completed: %v", updateErr)
			}
			if ackErr := msg.Ack(false); ackErr != nil {
				log.Printf("Erro ao fazer Ack: %v", ackErr)
			}
			return
		}

		log.Printf("Tentativa %d falhou para VideoID=%d: %v", attempt, job.VideoID, err)

		if attempt < maxRetries {
			waitTime := time.Duration(attempt*attempt) * time.Second
			log.Printf("Aguardando %v antes da próxima tentativa...", waitTime)
			time.Sleep(waitTime)
		}
	}

	log.Printf("Todas as %d tentativas falharam para VideoID=%d", maxRetries, job.VideoID)
	if updateErr := c.updateVideoStatus(job.VideoID, models.StatusFailed, job.AuthToken); updateErr != nil {
		log.Printf("Erro ao atualizar status para failed: %v", updateErr)
	}

	if ackErr := msg.Ack(false); ackErr != nil {
		log.Printf("Erro ao fazer Ack: %v", ackErr)
	}
}

func (c *Consumer) updateVideoStatus(videoID uint, status string, authToken string) error {
	getURL := fmt.Sprintf("%s/api/v1/videos/%d", c.apiBaseURL, videoID)
	getReq, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return fmt.Errorf("erro ao criar requisição GET: %w", err)
	}

	if authToken != "" {
		getReq.Header.Set("Authorization", authToken)
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

	var videoData map[string]any
	if err := json.NewDecoder(getResp.Body).Decode(&videoData); err != nil {
		return fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	var apiStatus string
	switch status {
	case models.StatusPending:
		apiStatus = "pending"
	case models.StatusProcessing:
		apiStatus = "processing"
	case models.StatusCompleted:
		apiStatus = "processed"
	case models.StatusFailed:
		apiStatus = "failed"
	default:
		apiStatus = "pending"
	}

	updateData := map[string]any{
		"title":  videoData["title"],
		"url":    videoData["url"],
		"status": apiStatus,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("erro ao serializar dados: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/videos/%d", c.apiBaseURL, videoID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("erro ao criar requisição: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

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

func (c *Consumer) processAndSaveVideo(job *models.VideoProcessingJob) error {
	if err := c.updateVideoStatus(job.VideoID, models.StatusProcessing, job.AuthToken); err != nil {
		log.Printf("Erro ao atualizar status para processing: %v", err)
	}

	result := c.processor.ProcessVideo(job)
	log.Printf("🔎 Resultado do processamento: Status=%s, Message=%s, ProcessedAt=%s, ZipPath=%s, FrameCount=%d, Images=%v",
		result.Status, result.Message, result.ProcessedAt.Format("2006-01-02 15:04:05"), result.ZipPath, result.FrameCount, result.Images)

	if result.Status == models.StatusCompleted {
		return c.saveProcessedVideo(job, result)
	}

	return fmt.Errorf("processamento falhou: %s", result.Message)
}

func (c *Consumer) saveProcessedVideo(job *models.VideoProcessingJob, result *video_processing.ProcessingResult) error {
	if result.ZipPath != "" {
		zipFilePath := filepath.Join("outputs", result.ZipPath)

		if _, err := os.Stat(zipFilePath); os.IsNotExist(err) {
			return fmt.Errorf("arquivo ZIP não encontrado: %s", zipFilePath)
		}

		zipFile, err := os.Open(zipFilePath)
		if err != nil {
			return fmt.Errorf("erro ao abrir arquivo ZIP: %w", err)
		}
		defer zipFile.Close()

		fileInfo, err := zipFile.Stat()
		if err != nil {
			return fmt.Errorf("erro ao obter informações do arquivo: %w", err)
		}

		objectName := fmt.Sprintf("%d/outputs/%s", job.UserID, result.ZipPath)

		_, err = c.minioClient.UploadFile(context.Background(), objectName, zipFile, fileInfo.Size())
		if err != nil {
			return fmt.Errorf("erro ao salvar arquivo ZIP no MinIO: %w", err)
		}

		log.Printf("✅ Arquivo ZIP salvo no MinIO: %s (frames: %d)", objectName, result.FrameCount)

		os.Remove(zipFilePath)

		return nil
	}

	fileName := strings.TrimSuffix(job.FileName, filepath.Ext(job.FileName))
	processedFileName := fmt.Sprintf("%s_processed.txt", fileName)
	objectName := fmt.Sprintf("%d/outputs/%s", job.UserID, processedFileName)

	processedContent := fmt.Sprintf("Processed video content for %s\nFrames extracted: %d", job.FileName, result.FrameCount)

	err := c.minioClient.UploadString(context.Background(), objectName, processedContent)
	if err != nil {
		return fmt.Errorf("erro ao salvar vídeo processado: %w", err)
	}

	log.Printf("✅ Vídeo processado salvo: %s", objectName)
	return nil
}
