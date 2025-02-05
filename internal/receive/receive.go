package receive

import (
	"encoding/json"
	"log"

	"rabbitmq_go_project/pkg/db"

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

func Run(rabbitMQURL, queueName, dbUser, dbPassword, dbHost string, dbPort int, dbName string) {
	conn, err := amqp.Dial(rabbitMQURL)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	failOnError(err, "Failed to declare a queue")

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

	dbConfig := db.Config{
		User:     dbUser,
		Password: dbPassword,
		Host:     dbHost,
		Port:     dbPort,
		DBName:   dbName,
	}
	dbConn := db.MustNewDB(dbConfig)
	defer dbConn.Close()

	var forever chan struct{}

	go func() {
		for d := range msgs {
			var event PaymentEvent
			err := json.Unmarshal(d.Body, &event)
			if err != nil {
				log.Printf("Error decoding JSON: %s", err)
				continue
			}
			err = db.InsertPaymentEvent(dbConn, event.UserID, event.DepositAmount)
			if err != nil {
				log.Printf("Error inserting payment event: %s", err)
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
