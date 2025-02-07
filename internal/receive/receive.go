package receive

import (
	"encoding/json"
	"log"

	"rabbitmq_go_project/pkg/db"
	"rabbitmq_go_project/pkg/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func Run(rabbitMQURL string, queueName string, dbUser string, dbPassword string, dbHost string, dbPort int, dbName string) {
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
	dbConn, err := db.OpenDBConnection(dbConfig)
	failOnError(err, "Failed to connect to database")
	defer dbConn.Close()

	var forever chan struct{}

	go func() {
		for d := range msgs {
			var event models.PaymentEvent
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
