package send

import (
	"context"
	"encoding/json"
	"log"
	"rabbitmq_go_project/pkg/models"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func publishPayloads(ch *amqp.Channel, q amqp.Queue) {
	payloads := []models.PaymentEvent{
		{UserID: 1, DepositAmount: 10},
		{UserID: 1, DepositAmount: 20},
		{UserID: 2, DepositAmount: 20},
	}

	for _, payload := range payloads {
		go func(payload models.PaymentEvent) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			body, err := json.Marshal(payload)
			failOnError(err, "Failed to marshal payload")

			err = ch.PublishWithContext(ctx,
				"",     // exchange
				q.Name, // routing key
				false,  // mandatory
				false,  // immediate
				amqp.Publishing{
					ContentType: "application/json",
					Body:        body,
				})
			failOnError(err, "Failed to publish a message")
			log.Printf(" [x] Sent %s\n", body)
		}(payload)
	}
}

func Run(rabbitMQURL, queueName string) {
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

	publishPayloads(ch, q)

	// Wait for a while to ensure all go routines finish
	time.Sleep(10 * time.Second)
}
