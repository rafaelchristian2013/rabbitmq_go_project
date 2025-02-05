package main

import (
	"rabbitmq_go_project/internal/receive"
)

func main() {
	rabbitMQURL := "amqp://admin:g79LK1aeHn8@localhost:5672/"
	queueName := "payment_events"

	dbUser := "app"
	dbPassword := "SAjfdbas54"
	dbHost := "localhost"
	dbPort := 3306
	dbName := "app"

	receive.Run(rabbitMQURL, queueName, dbUser, dbPassword, dbHost, dbPort, dbName)
}
