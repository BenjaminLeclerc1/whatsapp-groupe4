

package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func ConnectDB() *sql.DB {
	// Use the service name 'db' as defined in docker-compose.yml
	connStr := "host=db port=5432 user=whatsapp password=password dbname=whatsapp sslmode=disable"
	
	var db *sql.DB
	var err error

	// Retry loop: Essential for Docker containers starting at different speeds
	for i := 0; i < 20; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				fmt.Println("Successfully connected to database")
				return db
			}
		}
		
		log.Printf("Database not ready, retrying in 3 seconds... (%d/20)", i+1)
		time.Sleep(3 * time.Second)
	}

	log.Fatal("Could not connect to database after 20 attempts:", err)
	return nil
}