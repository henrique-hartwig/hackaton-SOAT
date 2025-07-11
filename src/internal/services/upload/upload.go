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

func HandleVideoUpload(c *gin.Context, minioClient *storage.MinioClient, publisher *queue.Publisher) {
	file, header, err := c.Request.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, UploadResponse{
			Success: false,
			Message: "Erro ao receber arquivo: " + err.Error(),
		})
		return
	}
	defer file.Close()

	if !isValidVideoFile(header.Filename) {
		c.JSON(http.StatusBadRequest, UploadResponse{
			Success: false,
			Message: "Formato de arquivo não suportado. Use: mp4, avi, mov, mkv",
		})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, UploadResponse{
			Success: false,
			Message: "Usuário não autenticado",
		})
		return
	}

	userIDUint := uint(userID.(int))

	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("%s_%s", timestamp, header.Filename)
	objectName := fmt.Sprintf("%d/input/%s", userIDUint, fileName)

	url, err := minioClient.UploadFile(c.Request.Context(), objectName, file, header.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Erro ao fazer upload para MinIO: " + err.Error(),
		})
		return
	}

	authHeader := c.GetHeader("Authorization")
	videoID, err := createVideoInAPI(header.Filename, url, userIDUint, authHeader)
	if err != nil {
		minioClient.DeleteFile(c.Request.Context(), objectName)

		c.JSON(http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Erro ao salvar registro na API: " + err.Error(),
		})
		return
	}

	job := &models.VideoProcessingJob{
		ID:        generateJobID(),
		VideoID:   videoID,
		UserID:    userIDUint,
		VideoURL:  url,
		FileName:  header.Filename,
		Status:    models.StatusPending,
		AuthToken: authHeader,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := publisher.PublishVideoProcessingJob(job); err != nil {
		fmt.Printf("⚠️ Erro ao enviar job para processamento: %v\n", err)
	}

	c.JSON(http.StatusCreated, UploadResponse{
		Success: true,
		Message: "Vídeo enviado com sucesso e enviado para processamento!",
		VideoID: videoID,
		URL:     url,
	})
}

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

	var videoResp VideoCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&videoResp); err != nil {
		return 0, err
	}

	return videoResp.ID, nil
}
