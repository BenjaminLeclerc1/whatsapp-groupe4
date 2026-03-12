package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Message représente un message indexé pour la recherche
type Message struct {
	ID        string    `json:"id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	ChatID    string    `json:"chat_id"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

// SearchResult représente un résultat de recherche avec score
type SearchResult struct {
	Message   Message `json:"message"`
	Score     int     `json:"score"`
	Highlight string  `json:"highlight"`
}

var (
	// Index des messages pour la recherche
	messages = make(map[string]Message)
	// Index inversé : mot -> liste de message IDs
	invertedIndex = make(map[string][]string)
	mu            sync.RWMutex
)

func main() {
	router := gin.Default()

	port := getEnv("PORT", "8084")

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "search-service",
		})
	})

	// Routes de recherche
	api := router.Group("/api/v1/search")
	{
		api.GET("/messages", searchMessages)
		api.GET("/messages/chat/:chatId", searchInChat)
		api.GET("/messages/user/:userId", searchByUser)
		api.POST("/index", indexMessage)
		api.DELETE("/index/:messageId", removeFromIndex)
		api.GET("/stats", getSearchStats)
	}

	log.Printf("Search Service démarré sur le port %s", port)

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

// indexMessage indexe un nouveau message pour la recherche
func indexMessage(c *gin.Context) {
	var input struct {
		ID        string    `json:"id" binding:"required"`
		SenderID  string    `json:"sender_id" binding:"required"`
		Content   string    `json:"content" binding:"required"`
		ChatID    string    `json:"chat_id" binding:"required"`
		CreatedAt time.Time `json:"created_at"`
		Status    string    `json:"status"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now()
	}
	if input.Status == "" {
		input.Status = "sent"
	}

	message := Message{
		ID:        input.ID,
		SenderID:  input.SenderID,
		Content:   input.Content,
		ChatID:    input.ChatID,
		CreatedAt: input.CreatedAt,
		Status:    input.Status,
	}

	mu.Lock()
	defer mu.Unlock()

	// Stocker le message
	messages[message.ID] = message

	// Indexer les mots du contenu
	words := tokenize(message.Content)
	for _, word := range words {
		if !contains(invertedIndex[word], message.ID) {
			invertedIndex[word] = append(invertedIndex[word], message.ID)
		}
	}

	log.Printf("Message indexé: %s (mots: %v)", message.ID, words)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Message indexé avec succès",
		"message_id":   message.ID,
		"words_indexed": len(words),
	})
}

// removeFromIndex supprime un message de l'index
func removeFromIndex(c *gin.Context) {
	messageID := c.Param("messageId")

	mu.Lock()
	defer mu.Unlock()

	message, exists := messages[messageID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message non trouvé dans l'index"})
		return
	}

	// Supprimer des index inversés
	words := tokenize(message.Content)
	for _, word := range words {
		invertedIndex[word] = removeString(invertedIndex[word], messageID)
		if len(invertedIndex[word]) == 0 {
			delete(invertedIndex, word)
		}
	}

	// Supprimer le message
	delete(messages, messageID)

	log.Printf("Message supprimé de l'index: %s", messageID)

	c.JSON(http.StatusOK, gin.H{"message": "Message supprimé de l'index"})
}

// searchMessages recherche des messages par mot-clé
func searchMessages(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Paramètre 'q' (query) requis"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	var limit int
	if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
		limit = 50
	}

	mu.RLock()
	defer mu.RUnlock()

	results := performSearch(query, "", "")

	// Limiter les résultats
	if len(results) > limit {
		results = results[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}

// searchInChat recherche des messages dans un chat spécifique
func searchInChat(c *gin.Context) {
	chatID := c.Param("chatId")
	query := c.Query("q")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Paramètre 'q' (query) requis"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	var limit int
	if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
		limit = 50
	}

	mu.RLock()
	defer mu.RUnlock()

	results := performSearch(query, chatID, "")

	// Limiter les résultats
	if len(results) > limit {
		results = results[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"chat_id": chatID,
		"results": results,
		"count":   len(results),
	})
}

// searchByUser recherche des messages d'un utilisateur spécifique
func searchByUser(c *gin.Context) {
	userID := c.Param("userId")
	query := c.Query("q")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Paramètre 'q' (query) requis"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	var limit int
	if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
		limit = 50
	}

	mu.RLock()
	defer mu.RUnlock()

	results := performSearch(query, "", userID)

	// Limiter les résultats
	if len(results) > limit {
		results = results[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"user_id": userID,
		"results": results,
		"count":   len(results),
	})
}

// getSearchStats retourne des statistiques sur l'index
func getSearchStats(c *gin.Context) {
	mu.RLock()
	defer mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"total_messages":     len(messages),
		"total_words_indexed": len(invertedIndex),
		"service":            "search-service",
	})
}

// performSearch effectue la recherche réelle
func performSearch(query, chatID, userID string) []SearchResult {
	queryWords := tokenize(query)
	if len(queryWords) == 0 {
		return []SearchResult{}
	}

	// Map pour compter les occurrences par message
	messageScores := make(map[string]int)

	// Chercher les messages contenant les mots de la requête
	for _, word := range queryWords {
		if messageIDs, exists := invertedIndex[word]; exists {
			for _, msgID := range messageIDs {
				messageScores[msgID]++
			}
		}
	}

	// Construire les résultats
	var results []SearchResult
	for msgID, score := range messageScores {
		message, exists := messages[msgID]
		if !exists {
			continue
		}

		// Filtrer par chat si spécifié
		if chatID != "" && message.ChatID != chatID {
			continue
		}

		// Filtrer par utilisateur si spécifié
		if userID != "" && message.SenderID != userID {
			continue
		}

		// Créer un highlight du texte
		highlight := createHighlight(message.Content, queryWords)

		results = append(results, SearchResult{
			Message:   message,
			Score:     score,
			Highlight: highlight,
		})
	}

	// Trier par score (décroissant)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// tokenize découpe un texte en mots normalisés
func tokenize(text string) []string {
	// Convertir en minuscules
	text = strings.ToLower(text)

	// Séparer par les espaces et la ponctuation
	separators := []string{" ", ",", ".", "!", "?", ";", ":", "'", "\"", "(", ")", "[", "]", "\n", "\t"}
	words := []string{text}

	for _, sep := range separators {
		var newWords []string
		for _, word := range words {
			parts := strings.Split(word, sep)
			for _, part := range parts {
				if part != "" {
					newWords = append(newWords, part)
				}
			}
		}
		words = newWords
	}

	// Filtrer les mots trop courts et les mots vides communs
	stopWords := map[string]bool{
		"le": true, "la": true, "les": true, "un": true, "une": true, "des": true,
		"de": true, "du": true, "a": true, "et": true, "ou": true, "est": true,
		"the": true, "an": true, "and": true, "or": true, "is": true,
	}

	var filtered []string
	for _, word := range words {
		if len(word) >= 2 && !stopWords[word] {
			filtered = append(filtered, word)
		}
	}

	return filtered
}

// createHighlight crée un extrait avec les mots recherchés mis en évidence
func createHighlight(content string, queryWords []string) string {
	lowerContent := strings.ToLower(content)

	// Trouver la première occurrence d'un mot de recherche
	firstPos := -1
	for _, word := range queryWords {
		pos := strings.Index(lowerContent, word)
		if pos != -1 && (firstPos == -1 || pos < firstPos) {
			firstPos = pos
		}
	}

	if firstPos == -1 {
		// Aucune occurrence trouvée, retourner le début
		if len(content) > 100 {
			return content[:100] + "..."
		}
		return content
	}

	// Extraire un contexte autour du mot trouvé
	start := firstPos - 30
	if start < 0 {
		start = 0
	}

	end := firstPos + 100
	if end > len(content) {
		end = len(content)
	}

	highlight := content[start:end]

	if start > 0 {
		highlight = "..." + highlight
	}
	if end < len(content) {
		highlight = highlight + "..."
	}

	return highlight
}

// Fonctions utilitaires
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeString(slice []string, item string) []string {
	result := []string{}
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
