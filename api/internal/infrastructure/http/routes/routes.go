package routes

import (
	"video-api/internal/infrastructure/http/handlers"

	"github.com/gin-gonic/gin"
)

func SetupVideoRoutes(router *gin.Engine, videoHandler *handlers.VideoHandler) {
	api := router.Group("/api/v1")
	{
		videos := api.Group("/videos")
		{
			videos.GET("", videoHandler.GetAll)
			videos.POST("", videoHandler.Create)
			videos.GET("/:id", videoHandler.GetByID)
			videos.PUT("/:id", videoHandler.Update)
			videos.DELETE("/:id", videoHandler.Delete)
		}
	}
}
