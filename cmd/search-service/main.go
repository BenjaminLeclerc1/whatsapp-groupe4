package main

import (
	"context"
	// "fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Message represents an indexed message
type Message struct {
	ID        string    `json:"id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	ChatID    string    `json:"chat_id"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

// SearchResult represents a result with scoring
type SearchResult struct {
	Message   Message `json:"message"`
	Score     int     `json:"score"`
	Highlight string  `json:"highlight"`
}

var (
	messages      = make(map[string]Message)
	invertedIndex = make(map[string][]string)
	mu            sync.RWMutex
)

func main() {
	port := getEnv("PORT", "8084")
	// Search maps to Shard 1 (where messages live)
	databaseURL := getEnv("DATABASE_URL", "postgres://whatsapp:whatsapp_secret@postgres_shard_1:5432/whatsapp_shard_1?sslmode=disable")

	// 1. Initialize DB Pool
	pool, err := initDB(databaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer pool.Close()

	// 2. Run Migrations (using unique tracking table)
	runMigrations(databaseURL)

	router := gin.Default()

	// Health check with DB status
	router.GET("/health", func(c *gin.Context) {
		err := pool.Ping(context.Background())
		dbStatus := "connected"
		if err != nil {
			dbStatus = "disconnected"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"service":  "search-service",
			"database": dbStatus,
		})
	})

	// Search Routes
	api := router.Group("/api/v1/search")
	{
		api.GET("/messages", searchMessages)
		api.GET("/messages/chat/:chatId", searchInChat)
		api.GET("/messages/user/:userId", searchByUser)
		api.POST("/index", indexMessage)
		api.DELETE("/index/:messageId", removeFromIndex)
		api.GET("/stats", getSearchStats)
	}

	log.Printf("Search Service started on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// --- DATABASE LOGIC ---

func initDB(databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(context.Background(), config)
}

func runMigrations(databaseURL string) {
	// Add unique migration table to avoid collisions on Shard 1
	targetURL := databaseURL + "&x-migrations-table=migrations_search"

	m, err := migrate.New("file://migrations/search-service", targetURL)
	if err != nil {
		log.Fatalf("Search migration init failed: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Search migration up failed: %v", err)
	}
	log.Println("Search migrations applied successfully!")
}

// --- SEARCH LOGIC (In-Memory Indexing) ---

func indexMessage(c *gin.Context) {
	var input Message
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now()
	}

	mu.Lock()
	defer mu.Unlock()

	messages[input.ID] = input
	words := tokenize(input.Content)
	for _, word := range words {
		if !contains(invertedIndex[word], input.ID) {
			invertedIndex[word] = append(invertedIndex[word], input.ID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "indexed", "message_id": input.ID})
}

func searchMessages(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query param 'q' is required"})
		return
	}

	mu.RLock()
	results := performSearch(query, "", "")
	mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"results": results, "count": len(results)})
}

func searchInChat(c *gin.Context) {
	chatID := c.Param("chatId")
	query := c.Query("q")

	mu.RLock()
	results := performSearch(query, chatID, "")
	mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"results": results, "count": len(results)})
}

func searchByUser(c *gin.Context) {
	userID := c.Param("userId")
	query := c.Query("q")

	mu.RLock()
	results := performSearch(query, "", userID)
	mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"results": results, "count": len(results)})
}

func removeFromIndex(c *gin.Context) {
	msgID := c.Param("messageId")
	mu.Lock()
	defer mu.Unlock()

	if msg, exists := messages[msgID]; exists {
		words := tokenize(msg.Content)
		for _, word := range words {
			invertedIndex[word] = removeString(invertedIndex[word], msgID)
		}
		delete(messages, msgID)
		c.JSON(http.StatusOK, gin.H{"message": "removed"})
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}

func getSearchStats(c *gin.Context) {
	mu.RLock()
	defer mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"indexed_messages": len(messages),
		"unique_words":    len(invertedIndex),
	})
}

// --- UTILS ---

func performSearch(query, chatID, userID string) []SearchResult {
	queryWords := tokenize(query)
	if len(queryWords) == 0 {
		return []SearchResult{}
	}

	scores := make(map[string]int)
	for _, word := range queryWords {
		if ids, exists := invertedIndex[word]; exists {
			for _, id := range ids {
				scores[id]++
			}
		}
	}

	var results []SearchResult
	for id, score := range scores {
		msg := messages[id]
		if (chatID != "" && msg.ChatID != chatID) || (userID != "" && msg.SenderID != userID) {
			continue
		}
		results = append(results, SearchResult{
			Message:   msg,
			Score:     score,
			Highlight: createHighlight(msg.Content, queryWords),
		})
	}
	return results
}

func tokenize(text string) []string {
	text = strings.ToLower(text)
	f := func(c rune) bool {
		return !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'))
	}
	return strings.FieldsFunc(text, f)
}

func createHighlight(content string, queryWords []string) string {
	// Simple slice for preview
	if len(content) > 100 {
		return content[:100] + "..."
	}
	return content
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item { return true }
	}
	return false
}

func removeString(slice []string, item string) []string {
	var result []string
	for _, s := range slice {
		if s != item { result = append(result, s) }
	}
	return result
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" { return value }
	return defaultValue
}