package main

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/whatsapp-groupe4/internal/logger"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

const authBearerPrefix = "Bearer "

func main() {
	logger.Init("api-gateway")
	defer logger.Close()

	jwtSecret := getEnv("JWT_SECRET", "whatsapp-groupe4-secret-change-in-prod")
	router := newGatewayRouter(jwtSecret)
	port := getEnv("API_GATEWAY_PORT", "8080")

	log.Printf("Gateway running on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("gateway run failed: %v", err)
	}
}

func newGatewayRouter(jwtSecret string) *gin.Engine {
	router := gin.Default()
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false
	router.Use(cors.New(cors.Config{
		AllowOrigins:     corsAllowOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	userServiceURL := getEnv("USER_SERVICE_URL", "http://user-service:8081")
	messageServiceURL := getEnv("MESSAGE_SERVICE_URL", "http://message-service:8082")
	notificationServiceURL := getEnv("NOTIFICATION_SERVICE_URL", "http://notification-service:8083")
	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://auth-service:8084")
	searchServiceURL := getEnv("SEARCH_SERVICE_URL", "http://search-service:8087")
	presenceServiceURL := getEnv("PRESENCE_SERVICE_URL", "http://presence-service:8086")
	channelServiceURL := getEnv("CHANNEL_SERVICE_URL", "http://channel-service:8085")
	chatServiceURL := getEnv("CHAT_SERVICE_URL", "http://chat-service:8088")
	wsGatewayURL := getEnv("WS_GATEWAY_URL", "http://ws-gateway:8089")

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "api-gateway"})
	})

	router.GET("/ws", wsProxyHandler(wsGatewayURL))

	api := router.Group("/api/v1")
	{
		api.Any("/auth/*path", proxyHandler(authServiceURL))
		api.Any("/search/*path", proxyHandler(searchServiceURL))
		api.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

		api.Any("/users/*path", authMiddleware(jwtSecret), proxyHandler(userServiceURL))

		protected := api.Group("/", authMiddleware(jwtSecret))
		{
			protected.Any("/chats", proxyHandler(chatServiceURL))
			protected.Any("/chats/*path", proxyHandler(chatServiceURL))
			protected.Any("/messages", proxyHandler(messageServiceURL))
			protected.Any("/messages/*path", proxyHandler(messageServiceURL))
			protected.Any("/presence/*path", proxyHandler(presenceServiceURL))
			protected.Any("/notification/*path", proxyHandler(notificationServiceURL))
			protected.Any("/channels/*path", proxyHandler(channelServiceURL))
		}
	}

	return router
}

func proxyHandler(targetURL string) gin.HandlerFunc {
	remote, err := url.Parse(targetURL)
	if err != nil {
		return func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid target URL"})
		}
	}

	return func(c *gin.Context) {
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.Director = func(req *http.Request) {
			// Clone pour ne pas muter la requête entrante et garantir l’envoi de X-User-ID vers les backends.
			if c.Request.Header != nil {
				req.Header = c.Request.Header.Clone()
			}
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			req.URL.Path = c.Request.URL.Path
			req.URL.RawQuery = c.Request.URL.RawQuery

			if userID, exists := c.Get("user_id"); exists {
				if uid, ok := userID.(string); ok && uid != "" {
					req.Header.Set("X-User-ID", uid)
				}
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

		claims, err := parseBearerClaims(c.GetHeader("Authorization"), jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}

func isPublicUserPath(path string) bool {
	return path == "/api/v1/users/register" || path == "/api/v1/users/login"
}

func parseBearerClaims(authHeader, jwtSecret string) (*Claims, error) {
	if authHeader == "" || !strings.HasPrefix(authHeader, authBearerPrefix) || len(authHeader) <= len(authBearerPrefix) {
		return nil, errors.New("missing bearer token")
	}

	tokenStr := strings.TrimPrefix(authHeader, authBearerPrefix)
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s environment variable is required", key)
	}
	return v
}

func corsAllowOrigins() []string {
	if v := os.Getenv("CORS_ALLOW_ORIGINS"); v != "" {
		var out []string
		for _, p := range strings.Split(v, ",") {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return []string{"http://localhost:3000"}
}
