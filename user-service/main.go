package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	
	// Only import the package you use directly in main.go
	"whatsapp-groupe4/user-service/handlers"
)

var db *sql.DB

func main() {
	gin.SetMode(gin.ReleaseMode)

	connectDB()
	defer db.Close()

	// 2. Call the autoMigrate function from the handlers package
	// Ensure AutoMigrate is capitalized in user_handler.go
	handlers.AutoMigrate(db)

	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// 3. Call the exported functions from the handlers package
	r.POST("/register", handlers.Register(db))
	r.POST("/login", handlers.Login(db))

	fmt.Println("User Service started on :8080")
	r.Run(":8080")
}

func connectDB() {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil && db.Ping() == nil {
			log.Println("Database connection established successfully!")
			return
		}
		log.Printf("Database not ready, retrying... (attempt %d/10)", i+1)
		time.Sleep(3 * time.Second)
	}
	log.Fatalf("Could not connect to database after 10 attempts: %v", err)
}