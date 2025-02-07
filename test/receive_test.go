package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"rabbitmq_go_project/pkg/db"
	"strconv"
	"testing"
	"time"

	"rabbitmq_go_project/internal/receive"
	"rabbitmq_go_project/pkg/models"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestReceiveAndWriteToDB(t *testing.T) {
	ctx := context.Background()

	log.Println("Starting RabbitMQ container...")
	rabbitmqContainer, err := startRabbitMQContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start RabbitMQ container: %v", err)
	}
	defer rabbitmqContainer.Terminate(ctx)
	log.Println("RabbitMQ container started.")

	log.Println("Starting MySQL container...")
	mysqlContainer, err := startMySQLContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start MySQL container: %v", err)
	}
	defer mysqlContainer.Terminate(ctx)
	log.Println("MySQL container started.")

	rabbitmqHost, err := rabbitmqContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get RabbitMQ container host: %v", err)
	}
	rabbitmqPort, err := rabbitmqContainer.MappedPort(ctx, "5672")
	if err != nil {
		t.Fatalf("Failed to get RabbitMQ container port: %v", err)
	}
	log.Printf("RabbitMQ is running at %s:%s\n", rabbitmqHost, rabbitmqPort.Port())

	conn, err := amqp.Dial(fmt.Sprintf("amqp://admin:g79LK1aeHn8@%s:%s/", rabbitmqHost, rabbitmqPort.Port()))
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"payment_events",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	mysqlHost, err := mysqlContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get MySQL container host: %v", err)
	}
	mysqlPort, err := mysqlContainer.MappedPort(ctx, "3306")
	if err != nil {
		t.Fatalf("Failed to get MySQL container port: %v", err)
	}
	log.Printf("MySQL is running at %s:%s\n", mysqlHost, mysqlPort.Port())

	mysqlPortInt, err := strconv.Atoi(mysqlPort.Port())
	if err != nil {
		t.Fatalf("Failed to convert MySQL port to integer: %v", err)
	}

	dbConfig := db.Config{
		User:     "app",
		Password: "apppassword",
		Host:     mysqlHost,
		Port:     mysqlPortInt,
		DBName:   "app_test",
	}
	db, err := db.OpenDBConnection(dbConfig)
	failOnError(err, "Failed to connect to MySQL")
	defer db.Close()

	err = runMigrations(mysqlHost, mysqlPort.Port())
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	go receive.Run(fmt.Sprintf("amqp://admin:g79LK1aeHn8@%s:%s/", rabbitmqHost, rabbitmqPort.Port()), "payment_events", "app", "apppassword", mysqlHost, mysqlPort.Int(), "app_test")

	payloads := []models.PaymentEvent{
		{UserID: 1, DepositAmount: 10},
		{UserID: 1, DepositAmount: 20},
		{UserID: 2, DepositAmount: 20},
	}
	for _, payload := range payloads {
		body, err := json.Marshal(payload)
		failOnError(err, "Failed to marshal payload")

		err = ch.Publish(
			"",
			q.Name,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			})
		failOnError(err, "Failed to publish a message")
	}

	// Give the receiver some time to process the messages
	time.Sleep(10 * time.Second)

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
		Image:        "rabbitmq:4.0.5-management-alpine",
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
		Image:        "mysql:9.2.0",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "root",
			"MYSQL_DATABASE":      "app_test",
			"MYSQL_USER":          "app",
			"MYSQL_PASSWORD":      "apppassword",
		},
		// Waits for MySQL to be fully ready before proceeding with tests.
		// This ensures that MySQL has started and is listening on port 3306, preventing connection failures.
		WaitingFor: wait.ForAll(
			wait.ForLog("port: 3306  MySQL Community Server - GPL"), // Wait for MySQL startup log
			wait.ForListeningPort("3306/tcp"),                       // Ensure MySQL is accepting connections
		).WithDeadline(2 * time.Minute), // Set a timeout to avoid infinite waiting
	}
	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func runMigrations(mysqlHost, mysqlPort string) error {
	log.Println("Running migrations...")
	migrationPath := "file://../db/migrations"
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
