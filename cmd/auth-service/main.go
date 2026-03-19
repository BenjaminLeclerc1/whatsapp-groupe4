package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"sync"
	"time"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/whatsapp-groupe4/internal/logger"
	"golang.org/x/crypto/bcrypt"
)

// User représente un utilisateur (inscription/connexion)
type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// account stocke email + hash bcrypt du mot de passe (données sensibles chiffrées)
type account struct {
	User         User
	PasswordHash string
}

// Claims JWT
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type refreshTokenRecord struct {
	UserID         string
	TokenHash      string
	CreatedAt      time.Time
	ExpiresAt      time.Time
	RevokedAt      *time.Time
	ReplacedByHash string
}

var (
	accountsByID    = make(map[string]account)
	accountsByEmail = make(map[string]string) // email -> userID
	refreshTokens   = make(map[string]*refreshTokenRecord) // tokenHash -> record
	mu              sync.RWMutex
)

func main() {
	logger.Init("auth-service")
	defer logger.Close()

	router := gin.Default()

	port := getEnv("PORT", "8084")
	jwtSecret := requireEnv("JWT_SECRET")
	accessTokenTTL := getEnvDuration("ACCESS_TOKEN_TTL", 24*time.Hour)
	refreshTokenTTL := getEnvDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour)

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
		api.POST("/register", register(jwtSecret, accessTokenTTL, refreshTokenTTL))
		// Connexion
		api.POST("/login", login(jwtSecret, accessTokenTTL, refreshTokenTTL))
		// Renouveler le JWT via refresh token
		api.POST("/refresh", refresh(jwtSecret, accessTokenTTL, refreshTokenTTL))
		// Déconnexion (révoque un refresh token)
		api.POST("/logout", logout())
		// Utilisateur courant (token requis)
		api.GET("/me", authMiddleware(jwtSecret), me())
	}

	logger.Info("Auth Service démarré sur le port %s", port)
	if err := router.Run(":" + port); err != nil {
		logger.Fatal("Erreur démarrage serveur: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func requireEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Variable d'environnement requise non définie : %s", key)
	}
	return value
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(value)
	if err != nil || d <= 0 {
		return defaultValue
	}
	return d
}

// register : inscription (création compte + JWT)
func register(jwtSecret string, accessTokenTTL, refreshTokenTTL time.Duration) gin.HandlerFunc {
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

		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			mu.Unlock()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du chiffrement du mot de passe"})
			return
		}

		user := User{
			ID:        uuid.New().String(),
			Username:  input.Username,
			Email:     input.Email,
			Status:    "active",
			CreatedAt: time.Now().Format(time.RFC3339),
		}
		acc := account{User: user, PasswordHash: string(hash)}
		accountsByID[user.ID] = acc
		accountsByEmail[input.Email] = user.ID
		mu.Unlock()

		token, err := generateJWT(user.ID, user.Email, jwtSecret, accessTokenTTL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération token"})
			return
		}

		refreshToken, refreshHash, err := generateRefreshToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération refresh token"})
			return
		}
		now := time.Now()
		mu.Lock()
		cleanupExpiredRefreshTokensLocked(now)
		refreshTokens[refreshHash] = &refreshTokenRecord{
			UserID:    user.ID,
			TokenHash: refreshHash,
			CreatedAt: now,
			ExpiresAt: now.Add(refreshTokenTTL),
		}
		mu.Unlock()

		c.JSON(http.StatusCreated, gin.H{
			"user":          user,
			"token":         token,
			"refresh_token": refreshToken,
		})
	}
}

// login : connexion (email + mot de passe -> JWT)
func login(jwtSecret string, accessTokenTTL, refreshTokenTTL time.Duration) gin.HandlerFunc {
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

		if err := bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(input.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
			return
		}

		token, err := generateJWT(acc.User.ID, acc.User.Email, jwtSecret, accessTokenTTL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération token"})
			return
		}

		refreshToken, refreshHash, err := generateRefreshToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération refresh token"})
			return
		}
		now := time.Now()
		mu.Lock()
		cleanupExpiredRefreshTokensLocked(now)
		refreshTokens[refreshHash] = &refreshTokenRecord{
			UserID:    acc.User.ID,
			TokenHash: refreshHash,
			CreatedAt: now,
			ExpiresAt: now.Add(refreshTokenTTL),
		}
		mu.Unlock()

		c.JSON(http.StatusOK, gin.H{
			"user":          acc.User,
			"token":         token,
			"refresh_token": refreshToken,
		})
	}
}

func refresh(jwtSecret string, accessTokenTTL, refreshTokenTTL time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		hash := hashRefreshToken(input.RefreshToken)
		now := time.Now()

		mu.Lock()
		defer mu.Unlock()

		cleanupExpiredRefreshTokensLocked(now)

		rec, ok := refreshTokens[hash]
		if !ok || rec == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token invalide"})
			return
		}
		if rec.RevokedAt != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token révoqué"})
			return
		}
		if !now.Before(rec.ExpiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token expiré"})
			return
		}

		acc, ok := accountsByID[rec.UserID]
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur introuvable"})
			return
		}

		accessToken, err := generateJWT(acc.User.ID, acc.User.Email, jwtSecret, accessTokenTTL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération token"})
			return
		}

		newRefreshToken, newRefreshHash, err := generateRefreshToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération refresh token"})
			return
		}

		revokedAt := now
		rec.RevokedAt = &revokedAt
		rec.ReplacedByHash = newRefreshHash

		refreshTokens[newRefreshHash] = &refreshTokenRecord{
			UserID:    acc.User.ID,
			TokenHash: newRefreshHash,
			CreatedAt: now,
			ExpiresAt: now.Add(refreshTokenTTL),
		}

		c.JSON(http.StatusOK, gin.H{
			"token":         accessToken,
			"refresh_token": newRefreshToken,
		})
	}
}

func logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token requis"})
			return
		}

		hash := hashRefreshToken(input.RefreshToken)
		now := time.Now()

		mu.Lock()
		defer mu.Unlock()

		rec, ok := refreshTokens[hash]
		if ok && rec != nil && rec.RevokedAt == nil {
			rec.RevokedAt = &now
		}

		// Toujours OK (idempotent) pour éviter l'énumération de tokens
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
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

func generateJWT(userID, email, secret string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func generateRefreshToken() (plain string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(b)
	hash = hashRefreshToken(plain)
	return plain, hash, nil
}

func hashRefreshToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func cleanupExpiredRefreshTokensLocked(now time.Time) {
	for h, rec := range refreshTokens {
		if rec == nil {
			delete(refreshTokens, h)
			continue
		}
		if !now.Before(rec.ExpiresAt) {
			delete(refreshTokens, h)
			continue
		}
	}
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
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}