package receive

import (
	"database/sql"
	"encoding/json"
	"log"

	_ "github.com/go-sql-driver/mysql"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PaymentEvent struct {
	UserID        int     `json:"user_id"`
	DepositAmount float64 `json:"deposit_amount"`
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
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

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var event PaymentEvent
			err := json.Unmarshal(d.Body, &event)
			failOnError(err, "Failed to unmarshal JSON")

			_, err = db.Exec("INSERT INTO payment_events (user_id, deposit_amount) VALUES (?, ?)", event.UserID, event.DepositAmount)
			failOnError(err, "Failed to insert into payment_events table")
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
