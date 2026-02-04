package main

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
)

type ShortenRequest struct {
	OriginalURL string `json:"original_url" binding:"required"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
	Code     string `json:"code"`
}

// thread-safe in-memory store: code -> originalURL
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]string)}
}

func (s *MemoryStore) Get(code string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[code]
	return v, ok
}

func (s *MemoryStore) Set(code, url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[code] = url
}

// generate a URL-safe random code of length n
func generateShortCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	// base64 URL-safe; remove padding; take first n chars
	code := base64.RawURLEncoding.EncodeToString(b)
	if len(code) < n {
		return "", nil
	}
	return code[:n], nil
}

func main() {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	store := NewMemoryStore()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/:code", func(c *gin.Context) {
		code := c.Param("code")
		originalURL, ok := store.Get(code)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "url not found"})
			return
		}
		c.Redirect(http.StatusFound, originalURL)
	})

	r.POST("/shorten", func(c *gin.Context) {
		var req ShortenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "original_url is required"})
			return
		}

		baseURL := os.Getenv("BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8080"
		}

		// Generate with collision retries (rare, but interview-worthy)
		const codeLen = 7
		const maxAttempts = 5

		var code string
		for i := 0; i < maxAttempts; i++ {
			newCode, err := generateShortCode(codeLen)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate code"})
				return
			}
			if _, exists := store.Get(newCode); !exists {
				code = newCode
				break
			}
		}

		if code == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "too many collisions generating code"})
			return
		}

		store.Set(code, req.OriginalURL)

		c.JSON(http.StatusOK, ShortenResponse{
			ShortURL: baseURL + "/" + code,
			Code:     code,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	_ = r.Run(":" + port)
}
