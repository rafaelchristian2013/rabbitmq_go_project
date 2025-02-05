# RabbitMQ Go Project

This project demonstrates how to use RabbitMQ with a Go application. It includes functionality for sending and receiving messages, as well as writing received messages to a MySQL database.

## Project Structure

- `cmd/receive/main.go`: Entry point for the message receiver.
- `cmd/send/main.go`: Entry point for the message sender.
- `internal/receive/receive.go`: Contains the logic for receiving messages from RabbitMQ and writing them to the database.
- `internal/send/send.go`: Contains the logic for sending messages to RabbitMQ.
- `pkg/db/db.go`: Contains database connection and query functions.
- `test/receive_test.go`: Contains integration tests for the message receiver.
- `db/migrations`: Contains database migration files.

## Prerequisites

- Go 1.18 or later
- Docker
- Docker Compose

## Setup

1. Start RabbitMQ and MySQL using Docker Compose:

    ```sh
    docker-compose up -d
    ```

2. Run database migrations:

    ```sh
    migrate -database "mysql://<DB_USER>:<DB_PASSWORD>@tcp(<DB_HOST>:<DB_PORT>)/<DB_NAME>" -path db/migrations up
    ```

## Running the Application

### Running the Receiver

The receiver listens for messages from RabbitMQ and writes them to the MySQL database.

1. Navigate to the `cmd/receive` directory:

    ```sh
    cd cmd/receive
    ```

2. Run the receiver:

    ```sh
    go run main.go
    ```

### Running the Sender

The sender publishes messages to RabbitMQ.

1. Navigate to the `cmd/send` directory:

    ```sh
    cd cmd/send
    ```

2. Run the sender:

    ```sh
    go run main.go
    ```

## Running Tests

To run the integration tests, use the following command:

```sh
go test -v ./test/...
```

## Configuration

The RabbitMQ and MySQL connection details are hardcoded in the source files. You can modify them as needed.
