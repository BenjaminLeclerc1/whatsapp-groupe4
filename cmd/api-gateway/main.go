package main

import (
	"errors"
	"log"
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

	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8086")
	userServiceURL := getEnv("USER_SERVICE_URL", "http://localhost:8081")
	chatServiceURL := getEnv("CHAT_SERVICE_URL", "http://localhost:8088")
	messageServiceURL := getEnv("MESSAGE_SERVICE_URL", "http://localhost:8082")
	presenceServiceURL := getEnv("PRESENCE_SERVICE_URL", "http://localhost:8083")
	searchServiceURL := getEnv("SEARCH_SERVICE_URL", "http://localhost:8084")
	notificationServiceURL := getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8085")
	channelServiceURL := getEnv("CHANNEL_SERVICE_URL", "http://localhost:8087")

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "api-gateway"})
	})

	api := router.Group("/api/v1")
	{
		// Public routes
		api.Any("/auth/*path", proxyHandler(authServiceURL))
		api.Any("/search/*path", proxyHandler(searchServiceURL))
		api.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

		// Protected routes
		protected := api.Group("/", authMiddleware(jwtSecret))
		{
			protected.Any("/users/*path", proxyHandler(userServiceURL))
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
			req.Header = c.Request.Header
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			req.URL.Path = c.Request.URL.Path

			if userID, exists := c.Get("user_id"); exists {
				if uid, ok := userID.(string); ok && uid != "" {
					req.Header.Set("X-User-ID", uid)
				}
			}
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func authMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Public user endpoints (no JWT required)
		if c.Request.URL.Path == "/api/v1/users/register" || c.Request.URL.Path == "/api/v1/users/login" {
			c.Next()
			return
		}

		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") || len(authHeader) <= len("Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
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
/*
package main

import (
	"errors"
	"log"
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

	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8086")
	userServiceURL := getEnv("USER_SERVICE_URL", "http://localhost:8081")
	chatServiceURL := getEnv("CHAT_SERVICE_URL", "http://localhost:8088")
	messageServiceURL := getEnv("MESSAGE_SERVICE_URL", "http://localhost:8082")
	presenceServiceURL := getEnv("PRESENCE_SERVICE_URL", "http://localhost:8083")
	searchServiceURL := getEnv("SEARCH_SERVICE_URL", "http://localhost:8084")
	notificationServiceURL := getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8085")
	channelServiceURL := getEnv("CHANNEL_SERVICE_URL", "http://localhost:8087")

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "api-gateway"})
	})

	api := router.Group("/api/v1")
	{
		// Public routes
		api.Any("/auth/*path", proxyHandler(authServiceURL))
		api.Any("/search/*path", proxyHandler(searchServiceURL))
		api.POST("/users/register", proxyHandler(userServiceURL))
		api.POST("/users/login", proxyHandler(userServiceURL))
		api.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

		// Protected routes
		protected := api.Group("/", authMiddleware(jwtSecret))
		{
			protected.Any("/users/*path", proxyHandler(userServiceURL))
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
			req.Header = c.Request.Header
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			req.URL.Path = c.Request.URL.Path

			if userID, exists := c.Get("user_id"); exists {
				if uid, ok := userID.(string); ok && uid != "" {
					req.Header.Set("X-User-ID", uid)
				}
			}
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func authMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") || len(authHeader) <= len("Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
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
package main

import (
	"errors"
	"log"
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

	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8086")
	userServiceURL := getEnv("USER_SERVICE_URL", "http://localhost:8081")
	chatServiceURL := getEnv("CHAT_SERVICE_URL", "http://localhost:8088")
	messageServiceURL := getEnv("MESSAGE_SERVICE_URL", "http://localhost:8082")
	presenceServiceURL := getEnv("PRESENCE_SERVICE_URL", "http://localhost:8083")
	searchServiceURL := getEnv("SEARCH_SERVICE_URL", "http://localhost:8084")
	notificationServiceURL := getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8085")
	channelServiceURL := getEnv("CHANNEL_SERVICE_URL", "http://localhost:8087")

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "api-gateway"})
	})

	api := router.Group("/api/v1")
	{
		// Public routes
		api.Any("/auth/*path", proxyHandler(authServiceURL))
		api.Any("/search/*path", proxyHandler(searchServiceURL))
		api.POST("/users/register", proxyHandler(userServiceURL))
		api.POST("/users/login", proxyHandler(userServiceURL))
		api.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

		// Protected routes
		protected := api.Group("/", authMiddleware(jwtSecret))
		{
			protected.Any("/users/*path", proxyHandler(userServiceURL))
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
			req.Header = c.Request.Header
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			req.URL.Path = c.Request.URL.Path

			if userID, exists := c.Get("user_id"); exists {
				if uid, ok := userID.(string); ok && uid != "" {
					req.Header.Set("X-User-ID", uid)
				}
			}
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func authMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") || len(authHeader) <= len("Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
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
// package main

// import (
// 	"log"
// 	"errors"
// 	"net/http"
// 	"os"
// 	"net/http/httputil"
//     "net/url"
// 	"time"

// 	"github.com/gin-contrib/cors"
// 	"github.com/gin-gonic/gin"
// 	"github.com/golang-jwt/jwt/v5"
// 	"github.com/whatsapp-groupe4/internal/logger"
// )

// type Claims struct {
// 	UserID string `json:"user_id"`
// 	Email  string `json:"email"`
// 	jwt.RegisteredClaims
// }

// func main() {
// 	logger.Init("api-gateway")
// 	defer logger.Close()

// 	router := gin.Default()

// 	// 1. Setup CORS configuration (KEEP THIS)
//     config := cors.Config{
//         AllowOrigins:     []string{"http://localhost:3000"},
//         AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
//         AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-ID"},
//         ExposeHeaders:    []string{"Content-Length"},
//         AllowCredentials: true,
//         MaxAge:           12 * time.Hour,
//     }
//     router.Use(cors.New(config))

//     // 2. 🔥 ADD THIS RIGHT HERE: Global OPTIONS handler
//     // This ensures OPTIONS never hits the authMiddleware
//     router.Use(func(c *gin.Context) {
//         if c.Request.Method == "OPTIONS" {
//             c.AbortWithStatus(204)
//             return
//         }
//         c.Next()
//     })

// 	userServiceURL := getEnv("USER_SERVICE_URL", "http://localhost:8081")
// 	messageServiceURL := getEnv("MESSAGE_SERVICE_URL", "http://localhost:8082")
// 	presenceServiceURL := getEnv("PRESENCE_SERVICE_URL", "http://localhost:8083")
// 	searchServiceURL := getEnv("SEARCH_SERVICE_URL", "http://localhost:8084")
// 	notificationServiceURL := getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8085")
// 	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8086")
// 	channelServiceURL := getEnv("CHANNEL_SERVICE_URL", "http://localhost:8087")
// 	jwtSecret := getEnv("JWT_SECRET", "whatsapp-groupe4-secret-change-in-prod")

// 	// Routes API Gateway
// 	router.GET("/health", func(c *gin.Context) {
// 		c.JSON(http.StatusOK, gin.H{
// 			"status":  "healthy",
// 			"service": "api-gateway",
// 		})
// 	})

// 	// Proxy vers les microservices
// 	// Proxy vers les microservices
//     api := router.Group("/api/v1")
//     {
//         // --- PUBLIC ROUTES ---
//         api.Any("/auth/*path", proxyHandler(authServiceURL))
//         api.Any("/search/*path", proxyHandler(searchServiceURL))
//         api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

//         // --- PROTECTED ROUTES (JWT Required) ---
//         protected := api.Group("/", authMiddleware(jwtSecret))
//         {
//             protected.Any("/users/*path", proxyHandler(userServiceURL))
//             protected.Any("/messages/*path", proxyHandler(messageServiceURL))
//             protected.Any("/presence/*path", proxyHandler(presenceServiceURL))
//             protected.Any("/notification/*path", proxyHandler(notificationServiceURL))
//             protected.Any("/channels/*path", proxyHandler(channelServiceURL))
            
//             // 🔥 ADD THE CHAT SERVICE HERE
//             chatServiceURL := getEnv("CHAT_SERVICE_URL", "http://localhost:8088")
//             protected.Any("/chats/*path", proxyHandler(chatServiceURL))
//         }
//     }

// 	port := getEnv("API_GATEWAY_PORT", "8080")

// 	log.Printf("API Gateway démarré sur le port %s", port)
// 	log.Printf("User Service URL: %s", userServiceURL)
// 	log.Printf("Message Service URL: %s", messageServiceURL)
// 	log.Printf("Presence Service URL: %s", presenceServiceURL)
// 	log.Printf("Search Service URL: %s", searchServiceURL)
// 	log.Printf("Notification Service URL: %s", notificationServiceURL)
// 	log.Printf("Auth Service URL: %s", authServiceURL)
// 	log.Printf("Channel Service URL: %s", channelServiceURL)

// 	if err := router.Run(":" + port); err != nil {
// 		log.Fatalf("Erreur démarrage serveur: %v", err)
// 	}
// }

// func getEnv(key, defaultValue string) string {
// 	if value := os.Getenv(key); value != "" {
// 		return value
// 	}
// 	return defaultValue
// }

// func requireEnv(key string) string {
// 	value := os.Getenv(key)
// 	if value == "" {
// 		log.Fatalf("Variable d'environnement requise non définie : %s", key)
// 	}
// 	return value
// }

// func proxyHandler(targetURL string) gin.HandlerFunc {
//     return func(c *gin.Context) {
//         // 1. Parse the destination service URL (e.g., http://chat-service:8088)
//         remote, err := url.Parse(targetURL)
//         if err != nil {
//             c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid target URL"})
//             return
//         }

//         // 2. Create the Reverse Proxy
//         proxy := httputil.NewSingleHostReverseProxy(remote)
        
//         // 3. The Director modifies the request before it leaves the Gateway
//         proxy.Director = func(req *http.Request) {
//             req.Header = c.Request.Header
//             req.Host = remote.Host
//             req.URL.Scheme = remote.Scheme
//             req.URL.Host = remote.Host
            
//             // 🔥 CRITICAL: Get UserID from JWT (set in authMiddleware) 
//             // and pass it to the microservice as a header
//             if userID, exists := c.Get("user_id"); exists {
//                 req.Header.Set("X-User-ID", userID.(string))
//             }
//         }

//         // 4. Send the request!
//         proxy.ServeHTTP(c.Writer, c.Request)
//     }
// }

// func authMiddleware(jwtSecret string) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		authHeader := c.GetHeader("Authorization")
// 		if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
// 			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token manquant ou invalide"})
// 			c.Abort()
// 			return
// 		}

// 		tokenStr := authHeader[7:]
// 		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
// 			if t.Method != jwt.SigningMethodHS256 {
// 				return nil, errors.New("unexpected signing method")
// 			}
// 			return []byte(jwtSecret), nil
// 		})
// 		if err != nil || !token.Valid {
// 			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide ou expiré"})
// 			c.Abort()
// 			return
// 		}

// 		claims, ok := token.Claims.(*Claims)
// 		if !ok {
// 			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
// 			c.Abort()
// 			return
// 		}

// 		// On propage l'identité dans le contexte et (plus tard) dans les headers si on met un vrai proxy HTTP.
// 		c.Set("user_id", claims.UserID)
// 		c.Set("email", claims.Email)

// 		c.Next()
// 	}
// }



// package main

// import (
// 	// "errors"
// 	"log"
// 	"net/http"
// 	"net/http/httputil"
// 	"net/url"
// 	"os"
// 	"time"

// 	"github.com/gin-contrib/cors"
// 	"github.com/gin-gonic/gin"
// 	"github.com/golang-jwt/jwt/v5"
// 	"github.com/whatsapp-groupe4/internal/logger"
// )

// type Claims struct {
// 	UserID string `json:"user_id"`
// 	Email  string `json:"email"`
// 	jwt.RegisteredClaims
// }

// func main() {
// 	logger.Init("api-gateway")
// 	defer logger.Close()

// 	router := gin.Default()

// 	// 🔥 ADD THESE TWO LINES:
//     router.RedirectTrailingSlash = false
//     router.RedirectFixedPath = false
// 	// 1. GLOBAL CORS CONFIGURATION
// 	// Must include "Authorization" and "X-User-ID" in AllowHeaders
// 	router.Use(cors.New(cors.Config{
// 		AllowOrigins:     []string{"http://localhost:3000"},
// 		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
// 		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-ID"},
// 		ExposeHeaders:    []string{"Content-Length"},
// 		AllowCredentials: true,
// 		MaxAge:           12 * time.Hour,
// 	}))

// 	// 2. GLOBAL OPTIONS HANDLER
// 	// Forces a 204 success for all preflight requests before they hit any middleware
// 	router.NoRoute(func(c *gin.Context) {
// 		if c.Request.Method == "OPTIONS" {
// 			c.AbortWithStatus(204)
// 			return
// 		}
// 		c.JSON(404, gin.H{"message": "Route not found"})
// 	})

// 	// Service URLs
// 	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8086")
// 	userServiceURL := getEnv("USER_SERVICE_URL", "http://localhost:8081")
// 	chatServiceURL := getEnv("CHAT_SERVICE_URL", "http://localhost:8088") // Ensure this port is correct
// 	jwtSecret := getEnv("JWT_SECRET", "whatsapp-groupe4-secret-change-in-prod")

// 	api := router.Group("/api/v1")
// 	{
// 		// PUBLIC ROUTES
// 		api.Any("/auth/*path", proxyHandler(authServiceURL))

// 		// PROTECTED ROUTES
// 		protected := api.Group("/", authMiddleware(jwtSecret))
// 		{
// 			protected.Any("/users/*path", proxyHandler(userServiceURL))
			
// 			// 🔥 FIX: Handle both the base /chats and subpaths /chats/:id
// 			protected.Any("/chats", proxyHandler(chatServiceURL))
// 			protected.Any("/chats/*path", proxyHandler(chatServiceURL))
// 		}
// 	}

// 	port := getEnv("API_GATEWAY_PORT", "8080")
// 	log.Printf("Gateway running on port %s", port)
// 	router.Run(":" + port)
// }
// func proxyHandler(targetURL string) gin.HandlerFunc {
//     remote, _ := url.Parse(targetURL)
//     return func(c *gin.Context) {
//         proxy := httputil.NewSingleHostReverseProxy(remote)
        
//         // Custom director to ensure headers and paths are clean
//         proxy.Director = func(req *http.Request) {
//             req.Header = c.Request.Header
//             req.Host = remote.Host
//             req.URL.Scheme = remote.Scheme
//             req.URL.Host = remote.Host
            
//             // Pass the UserID from middleware to the microservice
//             if userID, exists := c.Get("user_id"); exists {
//                 req.Header.Set("X-User-ID", userID.(string))
//             }
//         }
//         proxy.ServeHTTP(c.Writer, c.Request)
//     }
// }

// func authMiddleware(jwtSecret string) gin.HandlerFunc {
//     return func(c *gin.Context) {
//         // 🔥 FIX: Preflight requests don't have tokens. Let them pass to the CORS handler.
//         if c.Request.Method == "OPTIONS" {
//             c.Next()
//             return
//         }

//         authHeader := c.GetHeader("Authorization")
//         if authHeader == "" || len(authHeader) < 8 {
//             c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
//             return
//         }
//         // ... rest of your JWT logic ...
//         tokenStr := authHeader[7:]
//         token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
//             return []byte(jwtSecret), nil
//         })
//         if err != nil || !token.Valid {
//             c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
//             return
//         }
//         claims := token.Claims.(*Claims)
//         c.Set("user_id", claims.UserID)
//         c.Next()
//     }
// }
// func getEnv(key, defaultValue string) string {
// 	if value := os.Getenv(key); value != "" { return value }
// 	return defaultValue
// }


package main

import (
	"fmt"
	"log"
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

func main() {
	logger.Init("api-gateway")
	defer logger.Close()

	jwtSecret := getEnv("JWT_SECRET", "whatsapp-groupe4-secret-change-in-prod")
	router := newGatewayRouter(jwtSecret)
	port := getEnv("API_GATEWAY_PORT", "8080")
	log.Printf("Gateway running on port %s", port)
	router.Run(":" + port)
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

	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8086")
	userServiceURL := getEnv("USER_SERVICE_URL", "http://localhost:8081")
	chatServiceURL := getEnv("CHAT_SERVICE_URL", "http://localhost:8088")

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "api-gateway"})
	})

	api := router.Group("/api/v1")
	{
		api.Any("/auth/*path", proxyHandler(authServiceURL))
		protected := api.Group("/", authMiddleware(jwtSecret))
		{
			protected.Any("/users/*path", proxyHandler(userServiceURL))
			protected.Any("/chats", proxyHandler(chatServiceURL))
			protected.Any("/chats/*path", proxyHandler(chatServiceURL))
		}
	}
	return router
}

func proxyHandler(targetURL string) gin.HandlerFunc {
	remote, _ := url.Parse(targetURL)
	return func(c *gin.Context) {
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			
			// ✅ FORCE the path to match EXACTLY what the gateway got.
			// This prevents the proxy from adding/removing slashes.
			req.URL.Path = c.Request.URL.Path

			// Propagate authenticated user id to downstream services.
			if userID, exists := c.Get("user_id"); exists {
				if uid, ok := userID.(string); ok && uid != "" {
					req.Header.Set("X-User-ID", uid)
				}
			}
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
func authMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ✅ FIX 3: Let OPTIONS requests pass without checking for a token
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || len(authHeader) < 8 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
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

		claims := token.Claims.(*Claims)
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" { return value }
	return defaultValue
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatal(fmt.Sprintf("%s environment variable is required", key))
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

// package main

// import (
// 	"errors"
// 	"log"
// 	"net/http"
// 	"net/http/httputil"
// 	"net/url"
// 	"os"
// 	"time"

// 	"github.com/gin-contrib/cors"
// 	"github.com/gin-gonic/gin"
// 	"github.com/golang-jwt/jwt/v5"
// 	"github.com/whatsapp-groupe4/internal/logger"
// )

// type Claims struct {
// 	UserID string `json:"user_id"`
// 	Email  string `json:"email,omitempty"`
// 	Role   string `json:"role,omitempty"`
// 	jwt.RegisteredClaims
// }

// func main() {
// 	logger.Init("api-gateway")
// 	defer logger.Close()

// 	router := gin.Default()
// 	// ADD THESE TWO LINES:
//     // router.RedirectTrailingSlash = falsemessages
//     // router.RedirectFixedPath = false

// 	// 1. Setup CORS
// 	config := cors.Config{
// 		AllowOrigins:     []string{"http://localhost:3000"},
// 		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
// 		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-ID"},
// 		AllowCredentials: true,
// 		MaxAge:           12 * time.Hour,
// 	}


// 	// router.Use(cors.New(config))

// 	// // 2. Global OPTIONS handler (Handles pre-flight)
// 	// router.Use(func(c *gin.Context) {
// 	// 	if c.Request.Method == "OPTIONS" {
// 	// 		c.AbortWithStatus(204)
// 	// 		return
// 	// 	}
// 	// 	c.Next()
// 	// })


// 	// 1. Setup CORS (Keep this)
// router.Use(cors.New(config))

// // 2. Global OPTIONS handler - MUST be before any routes
// router.Use(func(c *gin.Context) {
//     if c.Request.Method == "OPTIONS" {
//         c.Header("Access-Control-Allow-Origin", "http://localhost:3000")
//         c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
//         c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
//         c.AbortWithStatus(204)
//         return
//     }
//     c.Next()
// })



// 	// 3. Define Service URLs
// 	userServiceURL := getEnv("USER_SERVICE_URL", "http://user-service:8081")
// 	messageServiceURL := getEnv("MESSAGE_SERVICE_URL", "http://message-service:8082")
// 	presenceServiceURL := getEnv("PRESENCE_SERVICE_URL", "http://presence-service:8083")
// 	searchServiceURL := getEnv("SEARCH_SERVICE_URL", "http://search-service:8084")
// 	notificationServiceURL := getEnv("NOTIFICATION_SERVICE_URL", "http://notification-service:8085")
// 	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://auth-service:8084")
// 	channelServiceURL := getEnv("CHANNEL_SERVICE_URL", "http://channel-service:8087")
// 	chatServiceURL := getEnv("CHAT_SERVICE_URL", "http://chat-service:8088")

// 	jwtSecret := os.Getenv("JWT_SECRET")
// 	if jwtSecret == "" {
// 		jwtSecret = "whatsapp-groupe4-secret-default"
// 	}
// 	log.Printf("DEBUG: Gateway using Secret: %s", jwtSecret)

// 	// 4. Routes
// 	api := router.Group("/api/v1")
// 	{
// 		// --- PUBLIC ---
// 		api.Any("/auth/*path", proxyHandler(authServiceURL))
// 		api.Any("/search/*path", proxyHandler(searchServiceURL))
// 		api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

// 		// --- PROTECTED ---
// 		protected := api.Group("/", authMiddleware(jwtSecret))
// 		{
// 			// Notice: No " : " here because we already declared chatServiceURL above
// 			protected.Any("/chats", proxyHandler(chatServiceURL))
// 			protected.Any("/chats/*path", proxyHandler(chatServiceURL))

// 			// protected.Any("/messages", proxyHandler(messageServiceURL))     // For POST (sending)
// // protected.Any("/messages/*path", proxyHandler(messageServiceURL)) // For GET/DELETE (IDs)
			
// 			protected.Any("/users/*path", proxyHandler(userServiceURL))
// 			protected.Any("/messages/*path", proxyHandler(messageServiceURL))
// 			protected.Any("/presence/*path", proxyHandler(presenceServiceURL))
// 			protected.Any("/notification/*path", proxyHandler(notificationServiceURL))
// 			protected.Any("/channels/*path", proxyHandler(channelServiceURL))
// 		}
// 	}

// 	port := getEnv("API_GATEWAY_PORT", "8080")
// 	log.Printf("API Gateway démarré sur le port %s", port)

// 	if err := router.Run(":" + port); err != nil {
// 		log.Fatalf("Erreur démarrage serveur: %v", err)
// 	}
// }

// func getEnv(key, defaultValue string) string {
// 	if value := os.Getenv(key); value != "" {
// 		return value
// 	}
// 	return defaultValue
// }

// func proxyHandler(targetURL string) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		remote, err := url.Parse(targetURL)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid target URL"})
// 			return
// 		}

// 		proxy := httputil.NewSingleHostReverseProxy(remote)


// 		// proxy.Director = func(req *http.Request) {
// 		// 	req.Header = c.Request.Header
// 		// 	req.Host = remote.Host
// 		// 	req.URL.Scheme = remote.Scheme
// 		// 	req.URL.Host = remote.Host
			
// 		// 	if userID, exists := c.Get("user_id"); exists {
// 		// 		req.Header.Set("X-User-ID", userID.(string))
// 		// 	}
// 		// }

// 		proxy.Director = func(req *http.Request) {
//     req.Header = c.Request.Header
//     req.Host = remote.Host
//     req.URL.Scheme = remote.Scheme
//     req.URL.Host = remote.Host
//     // Ensure the path sent to the microservice is exactly what the Gateway received
//     req.URL.Path = c.Request.URL.Path 
    
//     if userID, exists := c.Get("user_id"); exists {
//         req.Header.Set("X-User-ID", userID.(string))
//     }
// }

// 		proxy.ServeHTTP(c.Writer, c.Request)
// 	}
// }
*/
