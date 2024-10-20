package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var PostgresEngine *sql.DB

func InitDB() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	connStr := os.Getenv("POSTGRES_URL")
	if connStr == "" {
		log.Fatal("POSTGRES_URL is not set in the .env file")
	}

	PostgresEngine, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = PostgresEngine.Ping()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Database connection established")
}
