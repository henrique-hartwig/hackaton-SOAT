package routes

import (
	"video-api/internal/infrastructure/http/handlers"
	"video-api/internal/infrastructure/http/middleware"

	"github.com/gin-gonic/gin"
)

func SetupVideoRoutes(router *gin.Engine, videoHandler *handlers.VideoHandler) {
	api := router.Group("/api/v1")
	{
		protected := api.Group("videos")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/me", videoHandler.GetMyVideos)
			protected.GET("/:id", videoHandler.GetByID)
			protected.POST("", videoHandler.Create)
			protected.PUT("/:id", videoHandler.Update)
			protected.DELETE("/:id", videoHandler.Delete)
		}
	}
}
