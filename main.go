package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type ShortenRequest struct {
	OriginalURL string `json:"original_url" binding:"required"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
	Code     string `json:"code"`
}

func generateShortCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := base64.RawURLEncoding.EncodeToString(b)
	if len(code) < n {
		return "", fmt.Errorf("generated code too short")
	}
	return code[:n], nil
}

func main() {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// --- Config (env-based) ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)

	// --- Redis client ---
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx := context.Background()

	// Optional but very useful: fail fast if Redis isn't reachable
	if err := rdb.Ping(ctx).Err(); err != nil {
		// Keep running, but make it obvious Redis isn't connected
		// In production you might exit; for dev, we log and continue.
		fmt.Printf("WARNING: could not connect to Redis at %s: %v\n", redisAddr, err)
	} else {
		fmt.Printf("Connected to Redis at %s\n", redisAddr)
	}

	r.GET("/health", func(c *gin.Context) {
		// include redis health signal
		redisOK := true
		if err := rdb.Ping(ctx).Err(); err != nil {
			redisOK = false
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "redis": redisOK})
	})

	r.POST("/shorten", func(c *gin.Context) {
		var req ShortenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "original_url is required"})
			return
		}

		const codeLen = 7
		const maxAttempts = 5

		var code string
		for i := 0; i < maxAttempts; i++ {
			newCode, err := generateShortCode(codeLen)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate code"})
				return
			}

			// SETNX: set only if key does not already exist (atomic collision avoidance)
			ok, err := rdb.SetNX(ctx, newCode, req.OriginalURL, 0).Result()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "redis error"})
				return
			}
			if ok {
				code = newCode
				break
			}
		}

		if code == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "too many collisions generating code"})
			return
		}

		c.JSON(http.StatusOK, ShortenResponse{
			ShortURL: baseURL + "/" + code,
			Code:     code,
		})
	})

	r.GET("/:code", func(c *gin.Context) {
		code := c.Param("code")

		originalURL, err := rdb.Get(ctx, code).Result()
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "url not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "redis error"})
			return
		}

		c.Redirect(http.StatusFound, originalURL)
	})

	// Small graceful shutdown hint: give redis timeouts if you expand later
	_ = rdb.WithTimeout(5 * time.Second)

	_ = r.Run(":" + port)
}
