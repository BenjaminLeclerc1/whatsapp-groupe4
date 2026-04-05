package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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

// Claims JWT
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// rowScanner abstrait pgx.Row pour permettre les mocks en tests.
type rowScanner interface {
	Scan(dest ...any) error
}

// txDB abstrait les opérations d'une transaction pgx.
type txDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) rowScanner
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// dbPool abstrait *pgxpool.Pool pour les tests unitaires.
type dbPool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) rowScanner
	Begin(ctx context.Context) (txDB, error)
	Ping(ctx context.Context) error
}

// pgxTxWrapper adapte pgx.Tx vers l'interface txDB.
type pgxTxWrapper struct{ tx pgx.Tx }

func (t *pgxTxWrapper) QueryRow(ctx context.Context, sql string, args ...any) rowScanner {
	return t.tx.QueryRow(ctx, sql, args...)
}
func (t *pgxTxWrapper) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, args...)
}
func (t *pgxTxWrapper) Commit(ctx context.Context) error   { return t.tx.Commit(ctx) }
func (t *pgxTxWrapper) Rollback(ctx context.Context) error { return t.tx.Rollback(ctx) }

// poolAdapter adapte *pgxpool.Pool vers l'interface dbPool.
type poolAdapter struct{ p *pgxpool.Pool }

func (a *poolAdapter) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return a.p.Exec(ctx, sql, args...)
}
func (a *poolAdapter) QueryRow(ctx context.Context, sql string, args ...any) rowScanner {
	return a.p.QueryRow(ctx, sql, args...)
}
func (a *poolAdapter) Begin(ctx context.Context) (txDB, error) {
	tx, err := a.p.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxTxWrapper{tx: tx}, nil
}
func (a *poolAdapter) Ping(ctx context.Context) error { return a.p.Ping(ctx) }

func main() {
	logger.Init("auth-service")
	defer logger.Close()

	databaseURL := requireEnv("DATABASE_URL")
	jwtSecret := requireEnv("JWT_SECRET")
	port := getEnv("PORT", "8084")
	accessTokenTTL := getEnvDuration("ACCESS_TOKEN_TTL", 24*time.Hour)
	refreshTokenTTL := getEnvDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour)

	runMigrations(databaseURL)

	pool, err := initDB(databaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	db := &poolAdapter{p: pool}

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		dbStatus := "connected"
		if err := pool.Ping(ctx); err != nil {
			dbStatus = "disconnected"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"service":  "auth-service",
			"database": dbStatus,
		})
	})

	api := router.Group("/api/v1/auth")
	{
		api.POST("/register", register(db, jwtSecret, accessTokenTTL, refreshTokenTTL))
		api.POST("/login", login(db, jwtSecret, accessTokenTTL, refreshTokenTTL))
		api.POST("/refresh", refresh(db, jwtSecret, accessTokenTTL, refreshTokenTTL))
		api.POST("/logout", logout(db))
		api.GET("/me", authMiddleware(jwtSecret), me(db))
	}

	logger.Info("Auth Service démarré sur le port %s", port)
	if err := router.Run(":" + port); err != nil {
		logger.Fatal("Erreur démarrage serveur: %v", err)
	}
}

func initDB(databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 50
	cfg.MinConns = 10
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func runMigrations(databaseURL string) {
	m, err := migrate.New("file://migrations/auth-service", databaseURL)
	if err != nil {
		log.Fatalf("Could not create migrate instance: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Could not run up migrations: %v", err)
	}
	logger.Info("Auth migrations applied successfully")
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

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func isUniqueViolation(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23505"
}

func deleteExpiredRefreshTokens(ctx context.Context, pool dbPool) {
	_, _ = pool.Exec(ctx, `DELETE FROM auth_refresh_tokens WHERE expires_at < NOW()`)
}

func register(pool dbPool, jwtSecret string, accessTokenTTL, refreshTokenTTL time.Duration) gin.HandlerFunc {
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

		emailNorm := normalizeEmail(input.Email)
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du chiffrement du mot de passe"})
			return
		}

		userID := uuid.New().String()
		now := time.Now().UTC()

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		deleteExpiredRefreshTokens(ctx, pool)

		_, err = pool.Exec(ctx, `
			INSERT INTO auth_users (id, username, email, password_hash, status, created_at)
			VALUES ($1, $2, $3, $4, 'active', $5)
		`, userID, strings.TrimSpace(input.Username), emailNorm, string(hash), now)
		if err != nil {
			if isUniqueViolation(err) {
				c.JSON(http.StatusConflict, gin.H{"error": "Un compte existe déjà avec cet email"})
				return
			}
			logger.Error("register insert user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du compte"})
			return
		}

		user := User{
			ID:        userID,
			Username:  input.Username,
			Email:     emailNorm,
			Status:    "active",
			CreatedAt: now.Format(time.RFC3339),
		}

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

		_, err = pool.Exec(ctx, `
			INSERT INTO auth_refresh_tokens (token_hash, user_id, created_at, expires_at)
			VALUES ($1, $2, $3, $4)
		`, refreshHash, userID, now, now.Add(refreshTokenTTL))
		if err != nil {
			logger.Error("register insert refresh: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la session"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"user":          user,
			"token":         token,
			"refresh_token": refreshToken,
		})
	}
}

func login(pool dbPool, jwtSecret string, accessTokenTTL, refreshTokenTTL time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		emailNorm := normalizeEmail(input.Email)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		deleteExpiredRefreshTokens(ctx, pool)

		var id, username, email, status string
		var createdAt time.Time
		var passwordHash string
		err := pool.QueryRow(ctx, `
			SELECT id, username, email, password_hash, status, created_at
			FROM auth_users WHERE email = $1
		`, emailNorm).Scan(&id, &username, &email, &passwordHash, &status, &createdAt)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
			return
		}

		user := User{
			ID:        id,
			Username:  username,
			Email:     email,
			Status:    status,
			CreatedAt: createdAt.UTC().Format(time.RFC3339),
		}

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

		now := time.Now().UTC()
		_, err = pool.Exec(ctx, `
			INSERT INTO auth_refresh_tokens (token_hash, user_id, created_at, expires_at)
			VALUES ($1, $2, $3, $4)
		`, refreshHash, id, now, now.Add(refreshTokenTTL))
		if err != nil {
			logger.Error("login insert refresh: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user":          user,
			"token":         token,
			"refresh_token": refreshToken,
		})
	}
}

func refresh(pool dbPool, jwtSecret string, accessTokenTTL, refreshTokenTTL time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		hash := hashRefreshToken(input.RefreshToken)
		now := time.Now().UTC()

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		deleteExpiredRefreshTokens(ctx, pool)

		tx, txErr := pool.Begin(ctx)
		if txErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		var userID string
		var expiresAt time.Time
		var revokedAt *time.Time
		err := tx.QueryRow(ctx, `
			SELECT user_id, expires_at, revoked_at
			FROM auth_refresh_tokens WHERE token_hash = $1
			FOR UPDATE
		`, hash).Scan(&userID, &expiresAt, &revokedAt)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token invalide"})
			return
		}
		if revokedAt != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token révoqué"})
			return
		}
		if !now.Before(expiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token expiré"})
			return
		}

		var username, email, status string
		var createdAt time.Time
		if err2 := tx.QueryRow(ctx, `
			SELECT username, email, status, created_at FROM auth_users WHERE id = $1
		`, userID).Scan(&username, &email, &status, &createdAt); err2 != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur introuvable"})
			return
		}

		newRefreshToken, newRefreshHash, err3 := generateRefreshToken()
		if err3 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération refresh token"})
			return
		}

		if _, err4 := tx.Exec(ctx, `
			UPDATE auth_refresh_tokens
			SET revoked_at = $1, replaced_by_hash = $2
			WHERE token_hash = $3
		`, now, newRefreshHash, hash); err4 != nil {
			logger.Error("refresh update old token: %v", err4)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
			return
		}

		if _, err5 := tx.Exec(ctx, `
			INSERT INTO auth_refresh_tokens (token_hash, user_id, created_at, expires_at)
			VALUES ($1, $2, $3, $4)
		`, newRefreshHash, userID, now, now.Add(refreshTokenTTL)); err5 != nil {
			logger.Error("refresh insert new token: %v", err5)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
			return
		}

		if err6 := tx.Commit(ctx); err6 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
			return
		}

		accessToken, err7 := generateJWT(userID, email, jwtSecret, accessTokenTTL)
		if err7 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":         accessToken,
			"refresh_token": newRefreshToken,
		})
	}
}

func logout(pool dbPool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token requis"})
			return
		}

		hash := hashRefreshToken(input.RefreshToken)
		now := time.Now().UTC()

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		_, err := pool.Exec(ctx, `
			UPDATE auth_refresh_tokens SET revoked_at = $1
			WHERE token_hash = $2 AND revoked_at IS NULL
		`, now, hash)
		if err != nil {
			logger.Error("logout: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func me(pool dbPool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var username, email, status string
		var createdAt time.Time
		err := pool.QueryRow(ctx, `
			SELECT username, email, status, created_at FROM auth_users WHERE id = $1
		`, userID).Scan(&username, &email, &status, &createdAt)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
			return
		}

		c.JSON(http.StatusOK, User{
			ID:        userID.(string),
			Username:  username,
			Email:     email,
			Status:    status,
			CreatedAt: createdAt.UTC().Format(time.RFC3339),
		})
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
