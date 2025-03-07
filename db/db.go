package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type PostgresUrlVariableMissing struct{}

func (m *PostgresUrlVariableMissing) Error() string {
	return "POSTGRES_URL is not set in the .env file"
}

var PostgresEngine *sql.DB

func InitDB() error {
	connStr := os.Getenv("POSTGRES_URL")
	// log.Printf("Loaded POSTGRES_URL: %q", connStr)
	if connStr == "" {
		log.Fatal("POSTGRES_URL is not set in the .env file")
		return &PostgresUrlVariableMissing{}
	}

	var err error
	PostgresEngine, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
		return err
	}

	err = PostgresEngine.Ping()
	if err != nil {
		log.Fatal(err)
		return err
	}

	fmt.Println("Database connection established")
	return nil
}
