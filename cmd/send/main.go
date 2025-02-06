package main

import (
	"log"
	"os"
	"rabbitmq_go_project/internal/send"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	queueName := os.Getenv("QUEUE_NAME")

	send.Run(rabbitMQURL, queueName)
}
