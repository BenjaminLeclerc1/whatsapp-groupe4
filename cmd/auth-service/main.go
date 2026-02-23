package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// User représente un utilisateur (inscription/connexion)
type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// account stocke email + mot de passe (plain pour l'instant ; hash prévu sur une autre branche)
type account struct {
	User     User
	Password string
}

// Claims JWT
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

var (
	accountsByID    = make(map[string]account)
	accountsByEmail = make(map[string]string) // email -> userID
	mu              sync.RWMutex
)

func main() {
	router := gin.Default()

	port := getEnv("PORT", "8084")
	jwtSecret := getEnv("JWT_SECRET", "whatsapp-groupe4-secret-change-in-prod")

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "auth-service",
		})
	})

	api := router.Group("/api/v1/auth")
	{
		// Inscription
		api.POST("/register", register(jwtSecret))
		// Connexion
		api.POST("/login", login(jwtSecret))
		// Utilisateur courant (token requis)
		api.GET("/me", authMiddleware(jwtSecret), me())
	}

	log.Printf("Auth Service démarré sur le port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Erreur démarrage serveur: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// register : inscription (création compte + JWT)
func register(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Username string `json:"username" binding:"required"`
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required,min=6"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		mu.Lock()
		if _, exists := accountsByEmail[input.Email]; exists {
			mu.Unlock()
			c.JSON(http.StatusConflict, gin.H{"error": "Un compte existe déjà avec cet email"})
			return
		}

		user := User{
			ID:        uuid.New().String(),
			Username:  input.Username,
			Email:     input.Email,
			Status:    "active",
			CreatedAt: time.Now().Format(time.RFC3339),
		}
		acc := account{User: user, Password: input.Password}
		accountsByID[user.ID] = acc
		accountsByEmail[input.Email] = user.ID
		mu.Unlock()

		token, err := generateJWT(user.ID, user.Email, jwtSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération token"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"user":  user,
			"token": token,
		})
	}
}

// login : connexion (email + mot de passe -> JWT)
func login(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		mu.RLock()
		userID, exists := accountsByEmail[input.Email]
		if !exists {
			mu.RUnlock()
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
			return
		}
		acc := accountsByID[userID]
		mu.RUnlock()

		if acc.Password != input.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
			return
		}

		token, err := generateJWT(acc.User.ID, acc.User.Email, jwtSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user":  acc.User,
			"token": token,
		})
	}
}

// me : retourne l'utilisateur courant (à partir du JWT)
func me() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		mu.RLock()
		acc, exists := accountsByID[userID.(string)]
		mu.RUnlock()
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
			return
		}
		c.JSON(http.StatusOK, acc.User)
	}
}

func generateJWT(userID, email, secret string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func authMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || len(auth) < 8 || auth[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token manquant ou invalide"})
			c.Abort()
			return
		}
		tokenStr := auth[7:]
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
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
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}
