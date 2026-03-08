package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	userServiceURL := getEnv("USER_SERVICE_URL", "http://localhost:8081")
	messageServiceURL := getEnv("MESSAGE_SERVICE_URL", "http://localhost:8082")
	presenceServiceURL := getEnv("PRESENCE_SERVICE_URL", "http://localhost:8083")
	searchServiceURL := getEnv("SEARCH_SERVICE_URL", "http://localhost:8084")

	// Routes API Gateway
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "api-gateway",
		})
	})

	// Proxy vers les microservices
	api := router.Group("/api/v1")
	{
		api.Any("/users/*path", proxyHandler(userServiceURL))
		api.Any("/messages/*path", proxyHandler(messageServiceURL))
		api.Any("/presence/*path", proxyHandler(presenceServiceURL))
		api.Any("/search/*path", proxyHandler(searchServiceURL))
	}

	log.Println("API Gateway démarré sur le port 8080")
	log.Printf("User Service URL: %s", userServiceURL)
	log.Printf("Message Service URL: %s", messageServiceURL)
	log.Printf("Presence Service URL: %s", presenceServiceURL)
	log.Printf("Search Service URL: %s", searchServiceURL)

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Erreur démarrage serveur: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func proxyHandler(targetURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Param("path")
		c.JSON(http.StatusOK, gin.H{
			"message":    "Proxy vers " + targetURL + path,
			"target_url": targetURL,
			"path":       path,
		})
	}
}
