package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/whatsapp-groupe4/internal/logger"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email,omitempty"`
	Role   string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

func main() {
	logger.Init("api-gateway")
	defer logger.Close()

	router := gin.Default()

	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	config := cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(config))

	userServiceURL := getEnv("USER_SERVICE_URL", "http://user-service:8081")
	messageServiceURL := getEnv("MESSAGE_SERVICE_URL", "http://message-service:8082")
	notificationServiceURL := getEnv("NOTIFICATION_SERVICE_URL", "http://notification-service:8083")
	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://auth-service:8084")
	searchServiceURL := getEnv("SEARCH_SERVICE_URL", "http://search-service:8087")
	presenceServiceURL := getEnv("PRESENCE_SERVICE_URL", "http://presence-service:8086")
	channelServiceURL := getEnv("CHANNEL_SERVICE_URL", "http://channel-service:8085")
	chatServiceURL := getEnv("CHAT_SERVICE_URL", "http://chat-service:8088")
	wsGatewayURL := getEnv("WS_GATEWAY_URL", "http://ws-gateway:8089")
	jwtSecret := getEnv("JWT_SECRET", "whatsapp-groupe4-secret-default")

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "api-gateway",
		})
	})

	router.GET("/ws", wsProxyHandler(wsGatewayURL))

	api := router.Group("/api/v1")
	{
		api.Any("/auth/*path", proxyHandler(authServiceURL))
		api.Any("/search/*path", proxyHandler(searchServiceURL))
		api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

		protected := api.Group("/", authMiddleware(jwtSecret))
		{
			protected.Any("/chats", proxyHandler(chatServiceURL))
			protected.Any("/chats/*path", proxyHandler(chatServiceURL))

			protected.Any("/messages", proxyHandler(messageServiceURL))
			protected.Any("/messages/*path", proxyHandler(messageServiceURL))

			protected.Any("/users/*path", proxyHandler(userServiceURL))
			protected.Any("/presence/*path", proxyHandler(presenceServiceURL))
			protected.Any("/notification/*path", proxyHandler(notificationServiceURL))
			protected.Any("/channels/*path", proxyHandler(channelServiceURL))
		}
	}

	port := getEnv("API_GATEWAY_PORT", "8080")
	log.Printf("API Gateway started on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func proxyHandler(targetURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		remote, err := url.Parse(targetURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid target URL"})
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			req.URL.Path = c.Request.URL.Path
			if userID, exists := c.Get("user_id"); exists {
				req.Header.Set("X-User-ID", userID.(string))
			}
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func wsProxyHandler(target string) gin.HandlerFunc {
	u, _ := url.Parse(target)
	return func(c *gin.Context) {
		backend, err := net.DialTimeout("tcp", u.Host, 5*time.Second)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "service unavailable"})
			return
		}

		hijacker, ok := c.Writer.(http.Hijacker)
		if !ok {
			backend.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		_ = c.Request.Write(backend)

		clientConn, _, err := hijacker.Hijack()
		if err != nil {
			backend.Close()
			return
		}

		go func() {
			_, _ = io.Copy(backend, clientConn)
			backend.Close()
		}()
		_, _ = io.Copy(clientConn, backend)
		clientConn.Close()
	}
}

func authMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token missing"})
			return
		}

		tokenStr := authHeader[7:]
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		if claims, ok := token.Claims.(*Claims); ok {
			c.Set("user_id", claims.UserID)
			c.Next()
		} else {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
