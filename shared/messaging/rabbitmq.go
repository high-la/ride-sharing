package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/high-la/ride-sharing/shared/contracts"
	"github.com/high-la/ride-sharing/shared/retry"
	"github.com/high-la/ride-sharing/shared/tracing"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	TripExchange       = "trip"
	DeadLetterExchange = "dlx"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewRabbitMQ(uri string) (*RabbitMQ, error) {

	var conn *amqp.Connection
	var err error

	// Retry connection (important for Kubernetes startup)
	for i := 0; i < 10; i++ {

		log.Println("Connecting to RabbitMQ...")

		conn, err = amqp.Dial(uri)
		if err == nil {
			break
		}

		log.Printf("RabbitMQ not ready, retrying in 5 seconds... (%d/10)", i+1)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create channel: %v", err)
	}

	rmq := &RabbitMQ{
		conn:    conn,
		Channel: ch,
	}

	if err := rmq.setupExchangesAndQueues(); err != nil {
		rmq.Close()
		return nil, fmt.Errorf("failed to setup exchanges and queues: %v", err)
	}

	return rmq, nil
}

type MessageHandler func(context.Context, amqp.Delivery) error

func (r *RabbitMQ) ConsumeMessages(queueName string, handler MessageHandler) error {

	err := r.Channel.Qos(
		1,
		0,
		false,
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}

	msgs, err := r.Channel.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {

			if err := tracing.TracedConsumer(msg, func(ctx context.Context, d amqp.Delivery) error {

				log.Printf("Received a message: %s", msg.Body)

				cfg := retry.DefaultConfig()

				err := retry.WithBackoff(ctx, cfg, func() error {
					return handler(ctx, d)
				})

				if err != nil {

					log.Printf("Message processing failed after %d retries for message ID: %s, err: %v", cfg.MaxRetries, d.MessageId, err)

					headers := amqp.Table{}
					if d.Headers != nil {
						headers = d.Headers
					}

					headers["x-death-reason"] = err.Error()
					headers["x-origin-exchange"] = d.Exchange
					headers["x-original-routing-key"] = d.RoutingKey
					headers["x-retry-count"] = cfg.MaxRetries
					d.Headers = headers

					_ = d.Reject(false)

					return err
				}

				if ackErr := msg.Ack(false); ackErr != nil {
					log.Printf("ERROR: Failed to Ack message: %v. Message body: %s", ackErr, msg.Body)
				}

				return nil

			}); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		}
	}()

	return nil
}

func (r *RabbitMQ) PublishMessage(ctx context.Context, routingKey string, message contracts.AmqpMessage) error {

	log.Printf("Publishing message with routing key: %s", routingKey)

	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         jsonMsg,
	}

	return tracing.TracedPublisher(ctx, TripExchange, routingKey, msg, r.publish)
}

func (r *RabbitMQ) publish(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {

	return r.Channel.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false,
		false,
		msg,
	)
}

func (r *RabbitMQ) setupDeadLetterExchange() error {

	err := r.Channel.ExchangeDeclare(
		DeadLetterExchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter exchange: %v", err)
	}

	q, err := r.Channel.QueueDeclare(
		DeadLetterQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter queue: %v", err)
	}

	err = r.Channel.QueueBind(
		q.Name,
		"#",
		DeadLetterExchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind dead letter queue: %v", err)
	}

	return nil
}

type QueueConfig struct {
	Name        string
	RoutingKeys []string
	Exchange    string
}

func (r *RabbitMQ) setupQueues() error {

	queueConfigs := []QueueConfig{
		{
			Name:        FindAvailableDriversQueue,
			RoutingKeys: []string{contracts.TripEventCreated, contracts.TripEventDriverNotInterested},
			Exchange:    TripExchange,
		},
		{
			Name:        DriverCmdTripRequestQueue,
			RoutingKeys: []string{contracts.DriverCmdTripRequest},
			Exchange:    TripExchange,
		},
		{
			Name:        DriverTripResponseQueue,
			RoutingKeys: []string{contracts.DriverCmdTripAccept, contracts.DriverCmdTripDecline},
			Exchange:    TripExchange,
		},
		{
			Name:        NotifyDriverNoDriversFoundQueue,
			RoutingKeys: []string{contracts.TripEventNoDriversFound},
			Exchange:    TripExchange,
		},
		{
			Name:        NotifyDriverAssignQueue,
			RoutingKeys: []string{contracts.TripEventDriverAssigned},
			Exchange:    TripExchange,
		},
		{
			Name:        PaymentTripResponseQueue,
			RoutingKeys: []string{contracts.PaymentCmdCreateSession},
			Exchange:    TripExchange,
		},
		{
			Name:        NotifyPaymentSessionCreatedQueue,
			RoutingKeys: []string{contracts.PaymentEventSessionCreated},
			Exchange:    TripExchange,
		},
		{
			Name:        NotifyPaymentSuccessQueue,
			RoutingKeys: []string{contracts.PaymentEventSuccess},
			Exchange:    TripExchange,
		},
	}

	for _, cfg := range queueConfigs {

		if err := r.declareAndBindQueue(cfg.Name, cfg.RoutingKeys, cfg.Exchange); err != nil {
			return err
		}

	}

	return nil
}

func (r *RabbitMQ) setupExchangesAndQueues() error {

	if err := r.setupDeadLetterExchange(); err != nil {
		return err
	}

	err := r.Channel.ExchangeDeclare(
		TripExchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange %s: %w", TripExchange, err)
	}

	return r.setupQueues()
}

func (r *RabbitMQ) declareAndBindQueue(queueName string, routingKeys []string, exchange string) error {

	args := amqp.Table{
		"x-dead-letter-exchange": DeadLetterExchange,
	}

	q, err := r.Channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		args,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %v", queueName, err)
	}

	for _, routingKey := range routingKeys {

		err := r.Channel.QueueBind(
			q.Name,
			routingKey,
			exchange,
			false,
			nil,
		)

		if err != nil {
			return fmt.Errorf("failed to bind queue %s to routing key %s: %v", queueName, routingKey, err)
		}

	}

	log.Printf("Queue %s declared and bound successfully", queueName)

	return nil
}

func (r *RabbitMQ) Close() {

	if r.Channel != nil {
		r.Channel.Close()
	}

	if r.conn != nil {
		r.conn.Close()
	}

}
