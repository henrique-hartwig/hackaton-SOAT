package main

import (
	"log"
	"video-api/internal/infrastructure/database"
	"video-api/internal/infrastructure/http/handlers"
	"video-api/internal/infrastructure/http/routes"
)

func main() {
	cfg := database.Load()

	// Conectar banco
	db := database.Connect(cfg)
	defer db.Close()

	// Repositórios e handlers
	videoRepo := database.NewVideoRepository(db)
	userRepo := database.NewUserRepository(db)
	videoHandler := handlers.NewVideoHandler(videoRepo)
	userHandler := handlers.NewUserHandler(userRepo)

	// Configurar router com todas as rotas
	router := routes.SetupRouter(videoHandler, userHandler)

	log.Println("🚀 API iniciada na porta", cfg.Server.Port)
	log.Fatal(router.Run(cfg.Server.Port))
}
