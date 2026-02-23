package messaging

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn *amqp.Connection
}

func NewRabbitMQ(uri string) (*RabbitMQ, error) {

	// RabbitMQ connection
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect rabbitmq %v", err)
	}

	rmq := &RabbitMQ{
		Conn: conn,
	}

	return rmq, nil
}

func (r *RabbitMQ) Close() {

	if r.Conn != nil {
		r.Conn.Close()
	}
}
