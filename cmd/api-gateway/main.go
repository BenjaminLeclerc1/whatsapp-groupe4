package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func main() {
	router := gin.Default()

	userServiceURL := getEnv("USER_SERVICE_URL", "http://localhost:8081")
	messageServiceURL := getEnv("MESSAGE_SERVICE_URL", "http://localhost:8082")
	notificationServiceURL := getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8083")
	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8084")
	jwtSecret := getEnv("JWT_SECRET", "whatsapp-groupe4-secret-change-in-prod")

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
		// Routes publiques d'authentification (pas de JWT requis ici)
		api.Any("/auth/*path", proxyHandler(authServiceURL))

		// Toutes les autres routes /api/v1/** nécessitent un JWT valide
		protected := api.Group("/", authMiddleware(jwtSecret))
		{
			protected.Any("/users/*path", proxyHandler(userServiceURL))
			protected.Any("/messages/*path", proxyHandler(messageServiceURL))
			protected.Any("/notification/*path", proxyHandler(notificationServiceURL))
		}
	}

	log.Println("API Gateway démarré sur le port 8080")
	log.Printf("User Service URL: %s", userServiceURL)
	log.Printf("Message Service URL: %s", messageServiceURL)
	log.Printf("Notification Service URL: %s", notificationServiceURL)
	log.Printf("Auth Service URL: %s", authServiceURL)

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

func authMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token manquant ou invalide"})
			c.Abort()
			return
		}

		tokenStr := authHeader[7:]
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide ou expiré"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
			c.Abort()
			return
		}

		// On propage l'identité dans le contexte et (plus tard) dans les headers si on met un vrai proxy HTTP.
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		c.Next()
	}
}
