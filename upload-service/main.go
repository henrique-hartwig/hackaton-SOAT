package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"upload-service/internal/storage"

	"github.com/gin-gonic/gin"
)

type UploadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	VideoID uint   `json:"video_id,omitempty"`
	URL     string `json:"url,omitempty"`
}

type VideoCreateRequest struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type VideoCreateResponse struct {
	ID uint `json:"id"`
}

func main() {
	// Conectar MinIO
	minioClient, err := storage.NewMinioClient(
		"minio:9000",
		"minioadmin",
		"minioadmin",
		"videos",
	)
	if err != nil {
		log.Fatal("Erro ao conectar MinIO:", err)
	}

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

	// Endpoint de upload
	router.POST("/upload/video", func(c *gin.Context) {
		handleVideoUpload(c, minioClient)
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Println("🚀 Serviço de Upload iniciado na porta 8081")
	log.Fatal(router.Run(":8081"))
}

func handleVideoUpload(c *gin.Context, minioClient *storage.MinioClient) {
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

	// 3. Gerar nome único para o arquivo
	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("%s_%s", timestamp, header.Filename)
	objectName := fmt.Sprintf("videos/%s", fileName)

	// 4. Upload para MinIO
	url, err := minioClient.UploadFile(c.Request.Context(), objectName, file, header.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Erro ao fazer upload para MinIO: " + err.Error(),
		})
		return
	}

	// 5. Criar registro na API de vídeos
	videoID, err := createVideoInAPI(header.Filename, url)
	if err != nil {
		// Se falhar, tentar deletar do MinIO
		minioClient.DeleteFile(c.Request.Context(), objectName)

		c.JSON(http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Erro ao salvar registro na API: " + err.Error(),
		})
		return
	}

	// 6. Retornar sucesso
	c.JSON(http.StatusCreated, UploadResponse{
		Success: true,
		Message: "Vídeo enviado com sucesso!",
		VideoID: videoID,
		URL:     url,
	})
}

func createVideoInAPI(title, url string) (uint, error) {
	// Preparar dados para a API
	videoData := VideoCreateRequest{
		Title: title,
		URL:   url,
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
	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/videos", apiBaseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
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
