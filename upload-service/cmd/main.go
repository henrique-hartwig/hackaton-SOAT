package main

import (
	"log"
	"os"
	"upload-service/internal/middleware"
	"upload-service/internal/services/upload"
	"upload-service/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	minioClient, err := storage.NewMinioClient(
		os.Getenv("MINIO_ENDPOINT"),
		os.Getenv("MINIO_ACCESS_KEY"),
		os.Getenv("MINIO_SECRET_KEY"),
		os.Getenv("MINIO_BUCKET"),
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

	// Endpoint de upload (protegido por autenticação)
	router.POST("/upload/video", middleware.AuthMiddleware(), func(c *gin.Context) {
		upload.HandleVideoUpload(c, minioClient)
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Println("🚀 Serviço de Upload iniciado na porta 8081")
	log.Fatal(router.Run(":8081"))
}
