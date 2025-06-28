package main

import (
	"log"
	"video-api/internal/infrastructure/database"
	"video-api/internal/infrastructure/http/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := database.Load()

	// Conectar banco
	db := database.Connect(cfg)
	defer db.Close()

	// Repositórios e handlers
	videoRepo := database.NewVideoRepository(db)
	videoHandler := handlers.NewVideoHandler(videoRepo)

	// Router
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Rotas de vídeo (CRUD)
	api := router.Group("/api/v1")
	{
		api.POST("/videos", videoHandler.Create)
		api.GET("/videos", videoHandler.GetAll)
		api.GET("/videos/:id", videoHandler.GetByID)
		api.PUT("/videos/:id", videoHandler.Update)
		api.DELETE("/videos/:id", videoHandler.Delete)
	}

	log.Println("🚀 API CRUD iniciada na porta 8000")
	log.Fatal(router.Run(cfg.Server.Port))
}
