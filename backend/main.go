package main

import (
	"fmt"
	"log"

	"iupdater/config"
	"iupdater/models"
	"iupdater/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Fatal("Failed to load config:", err)
	}

	if err := models.InitDB(); err != nil {
		log.Fatal("Failed to init database:", err)
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	routes.SetupRoutes(r)

	addr := fmt.Sprintf(":%d", config.AppConfig.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
