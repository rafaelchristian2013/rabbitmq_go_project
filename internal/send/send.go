package send

import (
	"database/sql"
	"encoding/json"
	"log"

	_ "github.com/go-sql-driver/mysql"
	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func publishPayloads(ch *amqp.Channel, q amqp.Queue, db *sql.DB) {
	payloads := []struct {
		UserID        int `json:"user_id"`
		DepositAmount int `json:"deposit_amount"`
	}{
		{1, 10},
		{1, 20},
		{2, 20},
	}

	for _, payload := range payloads {
		go func(payload struct {
			UserID        int `json:"user_id"`
			DepositAmount int `json:"deposit_amount"`
		}) {
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

			_, err = db.Exec("INSERT INTO payment_events (user_id, deposit_amount) VALUES (?, ?)", payload.UserID, payload.DepositAmount)
			failOnError(err, "Failed to insert into payment_events table")
		}(payload)
	}
}

func Run() {
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

	db, err := sql.Open("mysql", "app:apppassword@tcp(localhost:3306)/app")
	failOnError(err, "Failed to connect to MySQL")
	defer db.Close()

	publishPayloads(ch, q, db)
}
