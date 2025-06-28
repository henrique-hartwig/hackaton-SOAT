package main

import (
	"log"
	"video-api/internal/infrastructure/database"
	"video-api/internal/infrastructure/http/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := database.Load()

	db := database.Connect(cfg)
	defer db.Close()

	videoRepo := database.NewVideoRepository(db)

	videoHandler := handlers.NewVideoHandler(videoRepo)

	router := gin.Default()

	api := router.Group("/api/v1")
	{
		api.POST("/videos", videoHandler.Create)
		api.GET("/videos", videoHandler.GetAll)
		api.GET("/videos/:id", videoHandler.GetByID)
		api.PUT("/videos/:id", videoHandler.Update)
		api.DELETE("/videos/:id", videoHandler.Delete)
	}

	log.Fatal(router.Run(cfg.Server.Port))
}
