// Package rabbitmq provides RabbitMQ producer management.
package rabbitmq

import (
	"context"
	"core-backend/config"
	"encoding/json"
	"fmt"
	"math"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// Producer handles publishing messages to RabbitMQ with retry logic
type Producer struct {
	name       string
	channel    *amqp.Channel
	config     config.RabbitMQProducerConfig
	confirms   chan amqp.Confirmation // For publisher confirms
	confirming bool                   // Whether confirmation mode is enabled
}

// ProducerManager manages multiple producers
type ProducerManager struct {
	conn      *amqp.Connection
	producers map[string]*Producer
	configs   []config.RabbitMQProducerConfig
}

// NewProducerManager creates a new producer manager
func NewProducerManager(conn *amqp.Connection, producerConfigs []config.RabbitMQProducerConfig) *ProducerManager {
	return &ProducerManager{
		conn:      conn,
		producers: make(map[string]*Producer),
		configs:   producerConfigs,
	}
}

// InitializeProducers creates and initializes all configured producers
func (pm *ProducerManager) InitializeProducers() error {
	zap.L().Info("Initializing RabbitMQ producers", zap.Int("count", len(pm.configs)))

	for _, producerConfig := range pm.configs {
		producer, err := pm.createProducer(producerConfig)
		if err != nil {
			return fmt.Errorf("failed to create producer %s: %w", producerConfig.Name, err)
		}

		pm.producers[producerConfig.Name] = producer
		zap.L().Info("Producer initialized",
			zap.String("name", producerConfig.Name),
			zap.String("exchange", producerConfig.Exchange),
			zap.String("routing_key", producerConfig.RoutingKey),
			zap.Bool("confirms", producerConfig.Confirm))
	}

	return nil
}

// createProducer creates a single producer with dedicated channel
func (pm *ProducerManager) createProducer(producerConfig config.RabbitMQProducerConfig) (*Producer, error) {
	// Create dedicated channel for this producer
	channel, err := pm.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	producer := &Producer{
		name:    producerConfig.Name,
		channel: channel,
		config:  producerConfig,
	}

	// Enable publisher confirms if configured
	if producerConfig.Confirm {
		if err := channel.Confirm(false); err != nil {
			channel.Close()
			return nil, fmt.Errorf("failed to enable confirm mode: %w", err)
		}
		producer.confirms = make(chan amqp.Confirmation, 1)
		channel.NotifyPublish(producer.confirms)
		producer.confirming = true
		zap.L().Debug("Publisher confirms enabled", zap.String("producer", producerConfig.Name))
	}

	return producer, nil
}

// GetProducer returns a producer by name
func (pm *ProducerManager) GetProducer(name string) (*Producer, error) {
	producer, exists := pm.producers[name]
	if !exists {
		return nil, fmt.Errorf("producer %s not found", name)
	}
	return producer, nil
}

// Close closes all producers
func (pm *ProducerManager) Close() error {
	zap.L().Info("Closing all producers", zap.Int("count", len(pm.producers)))

	var lastErr error
	for name, producer := range pm.producers {
		if err := producer.Close(); err != nil {
			zap.L().Error("Failed to close producer", zap.String("producer", name), zap.Error(err))
			lastErr = err
		}
	}

	return lastErr
}

// Publish publishes a message with retry logic
func (p *Producer) Publish(ctx context.Context, body []byte) error {
	return p.publishWithRetry(ctx, body, 0)
}

// PublishJSON publishes a JSON message with retry logic
func (p *Producer) PublishJSON(ctx context.Context, message any) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	return p.Publish(ctx, body)
}

// PublishJSONWithDelay publishes a JSON message with a delay (for delayed message exchange)
// The delayMs parameter specifies the delay in milliseconds before the message is delivered
// Requires the rabbitmq_delayed_message_exchange plugin to be enabled on the RabbitMQ server
func (p *Producer) PublishJSONWithDelay(ctx context.Context, message any, delayMs int64) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	return p.publishWithDelay(ctx, body, delayMs)
}

// publishWithDelay publishes a message with a delay header
func (p *Producer) publishWithDelay(ctx context.Context, body []byte, delayMs int64) error {
	deliveryMode := amqp.Transient
	if p.config.Persistent {
		deliveryMode = amqp.Persistent
	}

	zap.L().Debug("Publishing delayed message",
		zap.String("producer", p.name),
		zap.String("exchange", p.config.Exchange),
		zap.String("routing_key", p.config.RoutingKey),
		zap.Int64("delay_ms", delayMs),
		zap.Int("message_size", len(body)))

	// Publish message with x-delay header
	err := p.channel.PublishWithContext(
		ctx,
		p.config.Exchange,
		p.config.RoutingKey,
		p.config.Mandatory,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: deliveryMode,
			Timestamp:    time.Now(),
			Body:         body,
			Headers: amqp.Table{
				"x-delay": delayMs, // Delay in milliseconds for delayed message exchange
			},
		},
	)

	if err != nil {
		zap.L().Error("Failed to publish delayed message",
			zap.String("producer", p.name),
			zap.Error(err))
		return fmt.Errorf("failed to publish delayed message: %w", err)
	}

	// Wait for confirmation if enabled
	if p.confirming {
		select {
		case confirm := <-p.confirms:
			if !confirm.Ack {
				return fmt.Errorf("delayed message not acknowledged by broker")
			}
			zap.L().Debug("Delayed message confirmed by broker",
				zap.String("producer", p.name),
				zap.Uint64("delivery_tag", confirm.DeliveryTag))
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for confirmation: %w", ctx.Err())
		case <-time.After(5 * time.Second):
			return fmt.Errorf("confirmation timeout for delayed message")
		}
	}

	zap.L().Debug("Delayed message published successfully",
		zap.String("producer", p.name),
		zap.Int64("delay_ms", delayMs))

	return nil
}

// publishWithRetry publishes a message with exponential backoff retry
func (p *Producer) publishWithRetry(ctx context.Context, body []byte, attempt int) error {
	// Prepare publishing
	deliveryMode := amqp.Transient
	if p.config.Persistent {
		deliveryMode = amqp.Persistent
	}

	zap.L().Debug("Publishing message",
		zap.String("producer", p.name),
		zap.String("exchange", p.config.Exchange),
		zap.String("routing_key", p.config.RoutingKey),
		zap.Int("attempt", attempt+1),
		zap.Int("message_size", len(body)))

	// Publish message
	err := p.channel.PublishWithContext(
		ctx,
		p.config.Exchange,   // exchange
		p.config.RoutingKey, // routing key
		p.config.Mandatory,  // mandatory
		false,               // immediate (deprecated, always false)
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: deliveryMode,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)

	if err != nil {
		return p.handlePublishError(ctx, body, attempt, err)
	}

	// Wait for confirmation if enabled
	if p.confirming {
		select {
		case confirm := <-p.confirms:
			if !confirm.Ack {
				return p.handlePublishError(ctx, body, attempt, fmt.Errorf("message not acknowledged by broker"))
			}
			zap.L().Debug("Message confirmed by broker",
				zap.String("producer", p.name),
				zap.Uint64("delivery_tag", confirm.DeliveryTag))
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for confirmation: %w", ctx.Err())
		case <-time.After(5 * time.Second):
			return p.handlePublishError(ctx, body, attempt, fmt.Errorf("confirmation timeout"))
		}
	}

	zap.L().Debug("Message published successfully",
		zap.String("producer", p.name),
		zap.String("exchange", p.config.Exchange),
		zap.String("routing_key", p.config.RoutingKey))

	return nil
}

// handlePublishError handles publish failures with retry logic
func (p *Producer) handlePublishError(ctx context.Context, body []byte, attempt int, err error) error {
	// Check if retry is enabled
	if !p.config.Retry.Enabled {
		zap.L().Error("Message publish failed (no retry)",
			zap.String("producer", p.name),
			zap.Int("attempt", attempt+1),
			zap.Error(err))
		return fmt.Errorf("failed to publish message: %w", err)
	}

	// Check if max attempts reached
	if attempt >= p.config.Retry.MaxAttempts {
		zap.L().Error("Message publish failed after max retries",
			zap.String("producer", p.name),
			zap.Int("max_attempts", p.config.Retry.MaxAttempts),
			zap.Error(err))
		return fmt.Errorf("failed to publish message after %d attempts: %w", p.config.Retry.MaxAttempts, err)
	}

	// Calculate backoff delay with exponential multiplier
	backoffMs := p.config.Retry.BackoffMs
	if p.config.Retry.ExponentialMultiply > 0 && attempt > 0 {
		backoffMs = int(float64(backoffMs) * math.Pow(p.config.Retry.ExponentialMultiply, float64(attempt)))
	}

	zap.L().Warn("Message publish failed, retrying",
		zap.String("producer", p.name),
		zap.Int("attempt", attempt+1),
		zap.Int("max_attempts", p.config.Retry.MaxAttempts),
		zap.Int("backoff_ms", backoffMs),
		zap.Error(err))

	// Wait for backoff period
	select {
	case <-time.After(time.Duration(backoffMs) * time.Millisecond):
		return p.publishWithRetry(ctx, body, attempt+1)
	case <-ctx.Done():
		return fmt.Errorf("context cancelled during retry backoff: %w", ctx.Err())
	}
}

// Close closes the producer and its channel
func (p *Producer) Close() error {
	if p.channel != nil && !p.channel.IsClosed() {
		return p.channel.Close()
	}
	return nil
}

// GetConfig returns the producer configuration
func (p *Producer) GetConfig() config.RabbitMQProducerConfig {
	return p.config
}

// IsHealthy checks if the producer channel is still open
func (p *Producer) IsHealthy() bool {
	return p.channel != nil && !p.channel.IsClosed()
}
