package main

import (
	"log"
	"os"

	"github.com/jihadprakoso/menu-tree/backend/internal/config"
	"github.com/jihadprakoso/menu-tree/backend/internal/handler"
	"github.com/jihadprakoso/menu-tree/backend/internal/repository"
	"github.com/jihadprakoso/menu-tree/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/jihadprakoso/menu-tree/backend/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)


// @title        Menu Tree API
// @version      1.0
// @description  API untuk mengelola menu bertingkat (tree) dengan kedalaman tak terbatas
// @host         localhost:8080
// @BasePath     /api
func main() {
	_ = godotenv.Load()

	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalf("gagal konek database: %v", err)
	}

	// Rangkai layer: repository -> service -> handler
	repo := repository.NewMenuRepository(db)
	svc := service.NewMenuService(repo)
	menuHandler := handler.NewMenuHandler(svc)

	r := gin.Default()

	// CORS sederhana agar frontend (localhost:3000) boleh memanggil API
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	
	menuHandler.Register(r.Group("/api"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(r.Run(":" + port))
}