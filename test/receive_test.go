package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	"rabbitmq_go_project/internal/receive"

	// Ensure this import is present
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PaymentEvent struct {
	UserID        int `json:"user_id"`
	DepositAmount int `json:"deposit_amount"`
}

func TestReceiveAndWriteToDB(t *testing.T) {
	ctx := context.Background()

	// Start RabbitMQ container
	log.Println("Starting RabbitMQ container...")
	rabbitmqContainer, err := startRabbitMQContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start RabbitMQ container: %v", err)
	}
	defer rabbitmqContainer.Terminate(ctx)
	log.Println("RabbitMQ container started.")

	// Start MySQL container
	log.Println("Starting MySQL container...")
	mysqlContainer, err := startMySQLContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start MySQL container: %v", err)
	}
	defer mysqlContainer.Terminate(ctx)
	log.Println("MySQL container started.")

	// Get RabbitMQ container host and port
	rabbitmqHost, err := rabbitmqContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get RabbitMQ container host: %v", err)
	}
	rabbitmqPort, err := rabbitmqContainer.MappedPort(ctx, "5672")
	if err != nil {
		t.Fatalf("Failed to get RabbitMQ container port: %v", err)
	}
	log.Printf("RabbitMQ is running at %s:%s\n", rabbitmqHost, rabbitmqPort.Port())

	// Connect to RabbitMQ
	conn, err := amqp.Dial(fmt.Sprintf("amqp://admin:g79LK1aeHn8@%s:%s/", rabbitmqHost, rabbitmqPort.Port()))
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

	// Get MySQL container host and port
	mysqlHost, err := mysqlContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get MySQL container host: %v", err)
	}
	mysqlPort, err := mysqlContainer.MappedPort(ctx, "3306")
	if err != nil {
		t.Fatalf("Failed to get MySQL container port: %v", err)
	}
	log.Printf("MySQL is running at %s:%s\n", mysqlHost, mysqlPort.Port())

	// Connect to MySQL test database
	db, err := sql.Open("mysql", fmt.Sprintf("app:apppassword@tcp(%s:%s)/app_test", mysqlHost, mysqlPort.Port()))
	failOnError(err, "Failed to connect to MySQL")
	defer db.Close()

	// Run migrations
	err = runMigrations(mysqlHost, mysqlPort.Port())
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Start the receive process
	go receive.Run(fmt.Sprintf("amqp://admin:g79LK1aeHn8@%s:%s/", rabbitmqHost, rabbitmqPort.Port()), "payment_events", "app", "apppassword", mysqlHost, mysqlPort.Int(), "app_test")

	// Publish test messages
	payloads := []PaymentEvent{
		{UserID: 1, DepositAmount: 10},
		{UserID: 1, DepositAmount: 20},
		{UserID: 2, DepositAmount: 20},
	}

	for _, payload := range payloads {
		body, err := json.Marshal(payload)
		failOnError(err, "Failed to marshal payload")

		err = ch.Publish(
			"",     // exchange
			q.Name, // routing key
			false,  // mandatory
			false,  // immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			})
		failOnError(err, "Failed to publish a message")
	}

	// Give the receiver some time to process the messages
	time.Sleep(10 * time.Second) // Increased from 5 to 10 seconds

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

func startRabbitMQContainer(ctx context.Context) (testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        "rabbitmq:3-management",
		ExposedPorts: []string{"5672/tcp", "15672/tcp"},
		Env: map[string]string{
			"RABBITMQ_DEFAULT_USER": "admin",
			"RABBITMQ_DEFAULT_PASS": "g79LK1aeHn8",
		},
		WaitingFor: wait.ForLog("Server startup complete"),
	}
	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func startMySQLContainer(ctx context.Context) (testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "rootpassword",
			"MYSQL_DATABASE":      "app_test",
			"MYSQL_USER":          "app",
			"MYSQL_PASSWORD":      "apppassword",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("port: 3306  MySQL Community Server - GPL"),
			wait.ForListeningPort("3306/tcp"),
		).WithStartupTimeout(2 * time.Minute),
	}
	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func runMigrations(mysqlHost, mysqlPort string) error {
	log.Println("Running migrations...")
	migrationPath := "file://D:/Trems/rabbitmq_go_project/db/migrations" // Use absolute path
	m, err := migrate.New(
		migrationPath,
		fmt.Sprintf("mysql://app:apppassword@tcp(%s:%s)/app_test", mysqlHost, mysqlPort),
	)
	if err != nil {
		log.Printf("Error creating migrate instance: %v", err)
		return err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Printf("Error applying migrations: %v", err)
		return err
	}

	log.Println("Migrations applied successfully.")
	return nil
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
