// package main

// import (
//     "log"
//     "os"
//     "github.com/gin-gonic/gin"
//     "github.com/nats-io/nats.go" // Important: Import the NATS driver
//     "whatsapp-group4/message-service/handlers"
//     "whatsapp-group4/message-service/repository"
// )


// func main() {
//     // 1. Initialize NATS using env var
//     natsURL := os.Getenv("NATS_URL")
//     if natsURL == "" { natsURL = nats.DefaultURL }
//     nc, err := nats.Connect(natsURL)
//     if err != nil { log.Fatal(err) }
//     defer nc.Close()

//     // 2. Initialize Repo and Handler
//     repo := repository.NewMessageRepo()
//     handler := &handlers.MessageHandler{
//         Repo:       repo,
//         NATSClient: nc,
//     }
    
//     // 3. Setup Router
//     router := gin.Default()
    
//     // Define routes
//     msgGroup := router.Group("/messages")
//     {
//         msgGroup.POST("/", handler.CreateMessage)
//         msgGroup.GET("/chat/:chat_id", handler.GetChatHistory)
//         msgGroup.GET("/:id", handler.GetMessage)
//         msgGroup.DELETE("/:id", handler.DeleteMessage)
//     }

//     // 4. Run on port 8082 as planned
//     router.Run(":8082")
// }

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // Required for PostgreSQL driver
	"github.com/nats-io/nats.go"
	"whatsapp-groupe4/message-service/handlers"
	"whatsapp-groupe4/message-service/repository"
)

func main() {
	// 1. Setup Database Connection
	dbInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 2. Automatically create the table if it doesn't exist
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS messages (
		id VARCHAR(255) PRIMARY KEY,
		sender_id VARCHAR(255) NOT NULL,
		chat_id VARCHAR(255) NOT NULL,
		content TEXT NOT NULL,
		status VARCHAR(50),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// 3. Initialize NATS
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// 4. Initialize Repo (Passing db connection) and Handler
	repo := repository.NewMessageRepo(db)
	handler := &handlers.MessageHandler{
		Repo:       repo,
		NATSClient: nc,
	}

	// 5. Setup Router
	router := gin.Default()
	msgGroup := router.Group("/messages")
	{
		msgGroup.POST("/send", handler.CreateMessage)
		msgGroup.GET("/chat/:chat_id", handler.GetChatHistory)
		msgGroup.GET("/:id", handler.GetMessage)
		msgGroup.DELETE("/:id", handler.DeleteMessage)
	}

	// Print all registered routes to the console
for _, route := range router.Routes() {
    log.Printf("Method: %s, Path: %s\n", route.Method, route.Path)
}
	router.Run(":8082")
}