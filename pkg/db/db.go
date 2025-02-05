package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// Config holds the database configuration details
type Config struct {
	User     string
	Password string
	Host     string
	Port     int
	DBName   string
}

// NewDB initializes and returns a new database connection
func NewDB(cfg Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Verify the connection
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

// MustNewDB initializes a new database connection and panics if there's an error
func MustNewDB(cfg Config) *sql.DB {
	db, err := NewDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	return db
}

func InsertPaymentEvent(db *sql.DB, userID int, depositAmount float64) error {
	_, err := db.Exec("INSERT INTO payment_events (user_id, deposit_amount) VALUES (?, ?)", userID, depositAmount)
	if err != nil {
		log.Printf("Error inserting into database: %s", err)
		return err
	}
	log.Printf("Inserted payment event: userID=%d, depositAmount=%f", userID, depositAmount)
	return nil
}
