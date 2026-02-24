package handlers

import (
	"database/sql"
	"net/http"
	"log"
	"github.com/gin-gonic/gin"
	"whatsapp-groupe4/user-service/auth"
	"whatsapp-groupe4/user-service/models"
)

// AutoMigrate is now exported and callable from main.go
func AutoMigrate(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name TEXT,
		telephone TEXT,
		email TEXT UNIQUE,
		password TEXT,
		role TEXT
	)`
	db.Exec(query)
}

func Register(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var u models.User
		if err := c.BindJSON(&u); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		hashedPassword, _ := auth.HashPassword(u.Password)
		_, err := db.Exec("INSERT INTO users (name, telephone, email, password, role) VALUES ($1, $2, $3, $4, $5)",
			u.Name, u.Telephone, u.Email, hashedPassword, u.Role)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
	}
}

// Login is now exported and callable from main.go
func Login(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input models.User
		if err := c.BindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}

		var storedHash, role string
		err := db.QueryRow("SELECT password, role FROM users WHERE email=$1", input.Email).Scan(&storedHash, &role)
		if err != nil {
			log.Printf("DB error: %v", err) // Check logs for this
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}

		if err := auth.ComparePasswords(storedHash, input.Password); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		token, err := auth.GenerateToken(input.Email, role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
			return
		}

		// Ensure the token is actually included in the JSON response
		c.JSON(http.StatusOK, gin.H{
			"message": "Login successful",
			"token":   token, 
		})
	}
}