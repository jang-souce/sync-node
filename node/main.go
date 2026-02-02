package main

import (
	"log"
	"sync-node/node/service"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize HTTP service
	httpService := service.NewHttpService()

	// Define a simple ping route to verify Gin is working
	httpService.GetEngine().GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong from node",
		})
	})

	log.Println("Starting Node HTTP server on :8081")
	// Using a different port than central to avoid conflict if running locally
	if err := httpService.Start(":8081"); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
