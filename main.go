package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create router (production-style: explicit middleware)
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Health endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Configurable port (Docker-friendly)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	_ = r.Run(":" + port)
}
