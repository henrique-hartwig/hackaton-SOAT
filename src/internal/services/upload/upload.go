package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"src/internal/models"
	"src/internal/queue"
	"src/internal/storage"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type UploadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	VideoID uint   `json:"video_id,omitempty"`
	URL     string `json:"url,omitempty"`
}

type VideoCreateRequest struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	UserID uint   `json:"id_user"`
}

type VideoCreateResponse struct {
	ID uint `json:"id"`
}

// HandleVideoUpload processa o upload de vídeo e envia para processamento
func HandleVideoUpload(c *gin.Context, minioClient *storage.MinioClient, publisher *queue.Publisher) {
	// 1. Receber arquivo
	file, header, err := c.Request.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, UploadResponse{
			Success: false,
			Message: "Erro ao receber arquivo: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// 2. Validar tipo de arquivo
	if !isValidVideoFile(header.Filename) {
		c.JSON(http.StatusBadRequest, UploadResponse{
			Success: false,
			Message: "Formato de arquivo não suportado. Use: mp4, avi, mov, mkv",
		})
		return
	}

	// 3. Obter user ID do contexto (setado pelo middleware de autenticação)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, UploadResponse{
			Success: false,
			Message: "Usuário não autenticado",
		})
		return
	}

	userIDUint := uint(userID.(int))

	// 4. Gerar nome único para o arquivo
	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("%s_%s", timestamp, header.Filename)
	objectName := fmt.Sprintf("%d/input/%s", userIDUint, fileName)

	// 5. Upload para MinIO
	url, err := minioClient.UploadFile(c.Request.Context(), objectName, file, header.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Erro ao fazer upload para MinIO: " + err.Error(),
		})
		return
	}

	// 6. Criar registro na API de vídeos
	authHeader := c.GetHeader("Authorization")
	videoID, err := createVideoInAPI(header.Filename, url, userIDUint, authHeader)
	if err != nil {
		// Se falhar, tentar deletar do MinIO
		minioClient.DeleteFile(c.Request.Context(), objectName)

		c.JSON(http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Erro ao salvar registro na API: " + err.Error(),
		})
		return
	}

	// 7. Enviar job para processamento
	job := &models.VideoProcessingJob{
		ID:        generateJobID(),
		VideoID:   videoID,
		UserID:    userIDUint,
		VideoURL:  url,
		FileName:  header.Filename,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := publisher.PublishVideoProcessingJob(job); err != nil {
		// Log do erro, mas não falha o upload
		fmt.Printf("⚠️ Erro ao enviar job para processamento: %v\n", err)
		// Você pode decidir se quer falhar o upload ou apenas logar o erro
	}

	// 8. Retornar sucesso
	c.JSON(http.StatusCreated, UploadResponse{
		Success: true,
		Message: "Vídeo enviado com sucesso e enviado para processamento!",
		VideoID: videoID,
		URL:     url,
	})
}

// generateJobID gera um ID único para o job
func generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}

func isValidVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".mp4", ".avi", ".mov", ".mkv", ".wmv", ".flv", ".webm"}

	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

func createVideoInAPI(title, url string, userID uint, authHeader string) (uint, error) {
	videoData := VideoCreateRequest{
		Title:  title,
		URL:    url,
		UserID: userID,
	}

	jsonData, err := json.Marshal(videoData)
	if err != nil {
		return 0, err
	}

	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "http://localhost:8000"
	}

	// Chamar API de vídeos
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/videos", apiBaseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API retornou status %d: %s", resp.StatusCode, string(body))
	}

	// Ler resposta
	var videoResp VideoCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&videoResp); err != nil {
		return 0, err
	}

	return videoResp.ID, nil
}
