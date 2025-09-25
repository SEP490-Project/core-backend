package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"core-backend/config"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type RabbitMQ struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	queue    amqp.Queue
	exchange string
	config   *config.RabbitMQConfig
}

type MessageHandler func([]byte) error

func NewRabbitMQ() (*RabbitMQ, error) {
	zap.L().Info("Initializing RabbitMQ connection")
	
	cfg := config.GetAppConfig().RabbitMQ
	zap.L().Debug("RabbitMQ configuration loaded",
		zap.String("url", cfg.URL),
		zap.String("exchange", cfg.Exchange),
		zap.String("queue", cfg.Queue),
		zap.String("routing_key", cfg.RoutingKey))

	// Connect to RabbitMQ
	zap.L().Debug("Attempting to connect to RabbitMQ server")
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		zap.L().Error("Failed to connect to RabbitMQ",
			zap.String("url", cfg.URL),
			zap.Error(err))
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Create a channel
	zap.L().Debug("Creating RabbitMQ channel")
	channel, err := conn.Channel()
	if err != nil {
		zap.L().Error("Failed to open RabbitMQ channel", zap.Error(err))
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	rabbitmq := &RabbitMQ{
		conn:     conn,
		channel:  channel,
		exchange: cfg.Exchange,
		config:   &cfg,
	}

	// Declare exchange if specified
	if cfg.Exchange != "" {
		zap.L().Debug("Declaring RabbitMQ exchange", zap.String("exchange", cfg.Exchange))
		err = rabbitmq.declareExchange()
		if err != nil {
			zap.L().Error("Failed to declare RabbitMQ exchange",
				zap.String("exchange", cfg.Exchange),
				zap.Error(err))
			rabbitmq.Close()
			return nil, fmt.Errorf("failed to declare exchange: %w", err)
		}
		zap.L().Debug("RabbitMQ exchange declared successfully", zap.String("exchange", cfg.Exchange))
	}

	// Declare queue
	zap.L().Debug("Declaring RabbitMQ queue", zap.String("queue", cfg.Queue))
	err = rabbitmq.declareQueue()
	if err != nil {
		zap.L().Error("Failed to declare RabbitMQ queue",
			zap.String("queue", cfg.Queue),
			zap.Error(err))
		rabbitmq.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}
	zap.L().Debug("RabbitMQ queue declared successfully", zap.String("queue", cfg.Queue))

	// Bind queue to exchange if both are specified
	if cfg.Exchange != "" && cfg.RoutingKey != "" {
		zap.L().Debug("Binding queue to exchange",
			zap.String("queue", cfg.Queue),
			zap.String("exchange", cfg.Exchange),
			zap.String("routing_key", cfg.RoutingKey))
		err = rabbitmq.bindQueue()
		if err != nil {
			zap.L().Error("Failed to bind queue to exchange",
				zap.String("queue", cfg.Queue),
				zap.String("exchange", cfg.Exchange),
				zap.String("routing_key", cfg.RoutingKey),
				zap.Error(err))
			rabbitmq.Close()
			return nil, fmt.Errorf("failed to bind queue: %w", err)
		}
		zap.L().Debug("Queue bound to exchange successfully",
			zap.String("queue", cfg.Queue),
			zap.String("exchange", cfg.Exchange))
	}

	zap.L().Info("RabbitMQ connected successfully",
		zap.String("url", cfg.URL),
		zap.String("exchange", cfg.Exchange),
		zap.String("queue", cfg.Queue),
		zap.String("routing_key", cfg.RoutingKey))

	return rabbitmq, nil
}

// declareExchange declares an exchange
func (r *RabbitMQ) declareExchange() error {
	return r.channel.ExchangeDeclare(
		r.config.Exchange,   // exchange name
		"direct",            // type
		r.config.Durable,    // durable
		r.config.AutoDelete, // auto-deleted
		false,               // internal
		r.config.NoWait,     // no-wait
		nil,                 // arguments
	)
}

// declareQueue declares a queue
func (r *RabbitMQ) declareQueue() error {
	queue, err := r.channel.QueueDeclare(
		r.config.Queue,      // queue name
		r.config.Durable,    // durable
		r.config.AutoDelete, // delete when unused
		r.config.Exclusive,  // exclusive
		r.config.NoWait,     // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return err
	}
	r.queue = queue
	return nil
}

// bindQueue binds the queue to the exchange
func (r *RabbitMQ) bindQueue() error {
	return r.channel.QueueBind(
		r.queue.Name,        // queue name
		r.config.RoutingKey, // routing key
		r.config.Exchange,   // exchange
		r.config.NoWait,     // no-wait
		nil,                 // arguments
	)
}

// Publish publishes a message to the exchange or directly to a queue
func (r *RabbitMQ) Publish(ctx context.Context, body []byte) error {
	exchange := r.config.Exchange
	routingKey := r.config.RoutingKey

	// If no exchange is specified, publish directly to the queue
	if exchange == "" {
		exchange = ""
		routingKey = r.queue.Name
	}

	zap.L().Debug("Publishing message to RabbitMQ",
		zap.String("exchange", exchange),
		zap.String("routing_key", routingKey),
		zap.Int("message_size", len(body)))

	err := r.channel.PublishWithContext(
		ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	)
	
	if err != nil {
		zap.L().Error("Failed to publish message to RabbitMQ",
			zap.String("exchange", exchange),
			zap.String("routing_key", routingKey),
			zap.Error(err))
	} else {
		zap.L().Debug("Message published successfully to RabbitMQ",
			zap.String("exchange", exchange),
			zap.String("routing_key", routingKey))
	}
	
	return err
}

// PublishJSON publishes a JSON message
func (r *RabbitMQ) PublishJSON(ctx context.Context, message interface{}) error {
	zap.L().Debug("Publishing JSON message to RabbitMQ", zap.Any("message_type", fmt.Sprintf("%T", message)))
	
	body, err := json.Marshal(message)
	if err != nil {
		zap.L().Error("Failed to marshal JSON message for RabbitMQ",
			zap.Any("message_type", fmt.Sprintf("%T", message)),
			zap.Error(err))
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	return r.Publish(ctx, body)
}

// Consume starts consuming messages from the queue
func (r *RabbitMQ) Consume(ctx context.Context, handler MessageHandler) error {
	msgs, err := r.channel.Consume(
		r.queue.Name, // queue
		"",           // consumer
		true,         // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for {
			select {
			case msg := <-msgs:
				if msg.Body != nil {
					if err := handler(msg.Body); err != nil {
						zap.L().Error("Failed to handle message", zap.Error(err))
					}
				}
			case <-ctx.Done():
				zap.L().Info("Consumer stopped due to context cancellation")
				return
			}
		}
	}()

	zap.L().Info("Started consuming messages from queue", zap.String("queue", r.queue.Name))
	return nil
}

// ConsumeWithManualAck starts consuming messages with manual acknowledgment
func (r *RabbitMQ) ConsumeWithManualAck(ctx context.Context, handler func(amqp.Delivery) error) error {
	msgs, err := r.channel.Consume(
		r.queue.Name, // queue
		"",           // consumer
		false,        // auto-ack (disabled for manual ack)
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for {
			select {
			case msg := <-msgs:
				if msg.Body != nil {
					if err := handler(msg); err != nil {
						zap.L().Error("Failed to handle message", zap.Error(err))
						// Reject message and requeue
						msg.Nack(false, true)
					} else {
						// Acknowledge message
						msg.Ack(false)
					}
				}
			case <-ctx.Done():
				zap.L().Info("Consumer stopped due to context cancellation")
				return
			}
		}
	}()

	zap.L().Info("Started consuming messages from queue with manual ack", zap.String("queue", r.queue.Name))
	return nil
}

// GetQueueInfo returns information about the queue
func (r *RabbitMQ) GetQueueInfo() (int, int, error) {
	queue, err := r.channel.QueueInspect(r.queue.Name)
	if err != nil {
		return 0, 0, err
	}
	return queue.Messages, queue.Consumers, nil
}

// PurgeQueue removes all messages from the queue
func (r *RabbitMQ) PurgeQueue() (int, error) {
	return r.channel.QueuePurge(r.queue.Name, false)
}

// Close closes the RabbitMQ connection
func (r *RabbitMQ) Close() error {
	var err error
	if r.channel != nil {
		if closeErr := r.channel.Close(); closeErr != nil {
			err = closeErr
		}
	}
	if r.conn != nil {
		if closeErr := r.conn.Close(); closeErr != nil {
			err = closeErr
		}
	}
	return err
}

// IsConnected checks if the connection is still alive
func (r *RabbitMQ) IsConnected() bool {
	return r.conn != nil && !r.conn.IsClosed()
}

// GetConnection returns the underlying AMQP connection
func (r *RabbitMQ) GetConnection() *amqp.Connection {
	return r.conn
}

// GetChannel returns the underlying AMQP channel
func (r *RabbitMQ) GetChannel() *amqp.Channel {
	return r.channel
}
