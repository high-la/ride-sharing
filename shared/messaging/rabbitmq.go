package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

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
	conn, err := amqp.Dial(uri)
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
		// Clean up if setup fails
		rmq.Close()
		return nil, fmt.Errorf("failed to setup exchanges and queues: %v", err)
	}

	return rmq, nil
}

type MessageHandler func(context.Context, amqp.Delivery) error

func (r *RabbitMQ) ConsumeMessages(queueName string, handler MessageHandler) error {
	// Set prefetch count to 1 for fair dispatch
	// This tells RabbitMQ not to give more than one message to a service at a time.
	// The worker will only get the next message after it has acknowledged the previous one.
	err := r.Channel.Qos(
		1,     // prefetchCount: Limit to 1 unacknowledged message per consumer
		0,     // prefetchSize: No specific limit on message size
		false, // global: Apply prefetchCount to each consumer individually
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}

	msgs, err := r.Channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
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

					// Add failure context before sending to the DLQ
					headers := amqp.Table{}
					if d.Headers != nil {
						headers = d.Headers
					}

					headers["x-death-reason"] = err.Error()
					headers["x-origin-exchange"] = d.Exchange
					headers["x-original-routing-key"] = d.RoutingKey
					headers["x-retry-count"] = cfg.MaxRetries
					d.Headers = headers

					// Reject without requeue - message will go to the DLQ
					_ = d.Reject(false)
					return err
				}

				// Only Ack if the handler succeeds
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
	return r.Channel.PublishWithContext(ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		msg,
	)
}

func (r *RabbitMQ) setupDeadLetterExchange() error {
	// Declare the dead letter exchange
	err := r.Channel.ExchangeDeclare(
		DeadLetterExchange,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter exchange: %v", err)
	}

	// Declare the dead letter queue
	q, err := r.Channel.QueueDeclare(
		DeadLetterQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter queue: %v", err)
	}

	// Bind the queue to the exchange with a wildcard routing key
	err = r.Channel.QueueBind(
		q.Name,
		"#", // wildcard routing key to catch all messages
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
		if err := r.declareAndBindQueueSafely(cfg.Name, cfg.RoutingKeys, cfg.Exchange); err != nil {
			return err
		}
	}

	return nil
}

func (r *RabbitMQ) setupExchangesAndQueues() error {
	// First setup the DLQ exchange and queue
	if err := r.setupDeadLetterExchange(); err != nil {
		return err
	}

	err := r.Channel.ExchangeDeclare(
		TripExchange, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange %s: %w", TripExchange, err)
	}

	return r.setupQueues()
}

// declareAndBindQueueSafely handles queue declaration with proper error handling for existing queues
// declareAndBindQueueSafely handles queue declaration with proper error handling for existing queues
func (r *RabbitMQ) declareAndBindQueueSafely(queueName string, messageTypes []string, exchange string) error {
	// First, try to inspect if the queue exists
	existingQueue, inspectErr := r.Channel.QueueInspect(queueName)

	if inspectErr == nil {
		// Queue exists, we need to check if it has the dead letter configuration
		// Since Args isn't directly accessible, we'll try to redeclare with passive mode
		// to check if our desired args match

		// Try to declare the queue passively first - this will succeed if queue exists
		// and args match, fail if args don't match
		args := amqp.Table{
			"x-dead-letter-exchange": DeadLetterExchange,
		}

		_, declareErr := r.Channel.QueueDeclare(
			queueName,
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			args,  // arguments with DLX config
		)

		if declareErr != nil {
			// Check if it's a precondition failed error (406)
			if amqpErr, ok := declareErr.(*amqp.Error); ok && amqpErr.Code == 406 {
				log.Printf("WARNING: Queue %s exists with different arguments. Attempting to delete and recreate...", queueName)

				// Delete the queue (this is safe in development)
				_, deleteErr := r.Channel.QueueDelete(queueName, false, false, false)
				if deleteErr != nil {
					return fmt.Errorf("failed to delete queue %s: %v", queueName, deleteErr)
				}

				// Now create it with the correct configuration
				newQueue, createErr := r.Channel.QueueDeclare(
					queueName,
					true,  // durable
					false, // delete when unused
					false, // exclusive
					false, // no-wait
					args,  // arguments with DLX config
				)
				if createErr != nil {
					return fmt.Errorf("failed to recreate queue %s: %v", queueName, createErr)
				}

				log.Printf("Successfully recreated queue %s with dead letter exchange configuration", queueName)
				existingQueue = newQueue
			} else {
				return fmt.Errorf("failed to declare queue %s: %v", queueName, declareErr)
			}
		} else {
			log.Printf("Queue %s already exists with correct configuration", queueName)
		}
	} else {
		// Queue doesn't exist, create it with DLX
		if amqpErr, ok := inspectErr.(*amqp.Error); ok && amqpErr.Code == 404 {
			log.Printf("Queue %s doesn't exist, creating with dead letter exchange configuration", queueName)

			args := amqp.Table{
				"x-dead-letter-exchange": DeadLetterExchange,
			}

			newQueue, createErr := r.Channel.QueueDeclare(
				queueName,
				true,  // durable
				false, // delete when unused
				false, // exclusive
				false, // no-wait
				args,  // arguments with DLX config
			)
			if createErr != nil {
				return fmt.Errorf("failed to create queue %s: %v", queueName, createErr)
			}

			existingQueue = newQueue
		} else {
			return fmt.Errorf("failed to inspect queue %s: %v", queueName, inspectErr)
		}
	}

	// Bind the queue to all routing keys
	for _, routingKey := range messageTypes {
		if err := r.Channel.QueueBind(
			existingQueue.Name,
			routingKey,
			exchange,
			false,
			nil,
		); err != nil {
			// Check if it's a binding error
			if amqpErr, ok := err.(*amqp.Error); ok && amqpErr.Code == 406 {
				log.Printf("Queue %s already bound to routing key %s", queueName, routingKey)
				continue
			}
			return fmt.Errorf("failed to bind queue %s to routing key %s: %v", queueName, routingKey, err)
		}
	}

	return nil
}

func (r *RabbitMQ) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
	if r.Channel != nil {
		r.Channel.Close()
	}
}
