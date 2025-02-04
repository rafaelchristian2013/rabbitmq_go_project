package main

import (
	"database/sql"
	"log"
	"os/exec"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestPublishPayloads(t *testing.T) {
	// Start Docker containers
	err := startDockerContainers()
	if err != nil {
		t.Fatalf("Failed to start Docker containers: %v", err)
	}
	defer stopDockerContainers()

	// Connect to RabbitMQ
	conn, err := amqp.Dial("amqp://admin:g79LK1aeHn8@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"payment_events", // name
		false,            // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	failOnError(err, "Failed to declare a queue")

	// Connect to MySQL
	db, err := sql.Open("mysql", "app:apppassword@tcp(localhost:3306)/app")
	failOnError(err, "Failed to connect to MySQL")
	defer db.Close()

	// Publish payloads
	publishPayloads(ch, q, db)

	// Check if the payloads were written to the database
	rows, err := db.Query("SELECT user_id, deposit_amount FROM payment_events")
	failOnError(err, "Failed to query payment_events table")
	defer rows.Close()

	var count int
	for rows.Next() {
		count++
	}

	assert.Equal(t, 3, count, "Expected 3 rows in the payment_events table")
}

func startDockerContainers() error {
	cmd := exec.Command("docker-compose", "up", "-d")
	return cmd.Run()
}

func stopDockerContainers() {
	cmd := exec.Command("docker-compose", "down")
	cmd.Run()
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
