package cloud

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// MustDB membuka koneksi PostgreSQL atau panic saat init.
// Panic tidak fatal saat init karena service tidak bisa berfungsi tanpa database.
func MustDB(databaseURL string) *sql.DB {
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	fmt.Println("PostgreSQL connected")
	return db
}
