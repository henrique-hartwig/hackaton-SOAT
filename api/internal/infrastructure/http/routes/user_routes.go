package routes

import (
	"video-api/internal/infrastructure/http/handlers"

	"github.com/gin-gonic/gin"
)

func SetupUserRoutes(router *gin.Engine, userHandler *handlers.UserHandler) {
	api := router.Group("/api/v1")
	{
		api.POST("/auth/signup", userHandler.Signup)
		api.POST("/auth/login", userHandler.Login)
	}
}
