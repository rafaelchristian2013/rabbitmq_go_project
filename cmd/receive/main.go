package main

import (
	"log"
	"os"
	"rabbitmq_go_project/internal/receive"
	"strconv"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	queueName := os.Getenv("QUEUE_NAME")

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPortStr := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		log.Fatal("Invalid port number")
	}

	receive.Run(rabbitMQURL, queueName, dbUser, dbPassword, dbHost, dbPort, dbName)
}
