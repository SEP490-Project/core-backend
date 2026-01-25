// Package rabbitmq provides RabbitMQ consumer management.
package rabbitmq

import (
	"context"
	"core-backend/config"
	"fmt"
	"math"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// MessageHandler is the interface for handling consumed messages
type MessageHandler interface {
	Handle(ctx context.Context, body []byte) error
}

// MessageHandlerFunc is a function adapter for MessageHandler
type MessageHandlerFunc func(ctx context.Context, body []byte) error

// Handle implements MessageHandler interface
func (f MessageHandlerFunc) Handle(ctx context.Context, body []byte) error {
	return f(ctx, body)
}

// Consumer handles consuming messages from a RabbitMQ queue
type Consumer struct {
	name     string
	channel  *amqp.Channel
	config   config.RabbitMQConsumerConfig
	handler  MessageHandler
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// ConsumerManager manages multiple consumers
type ConsumerManager struct {
	conn      *amqp.Connection
	consumers map[string]*Consumer
	configs   []config.RabbitMQConsumerConfig
	handlers  map[string]MessageHandler // Handler registry
}

// NewConsumerManager creates a new consumer manager
func NewConsumerManager(conn *amqp.Connection, consumerConfigs []config.RabbitMQConsumerConfig) *ConsumerManager {
	return &ConsumerManager{
		conn:      conn,
		consumers: make(map[string]*Consumer),
		configs:   consumerConfigs,
		handlers:  make(map[string]MessageHandler),
	}
}

// RegisterHandler registers a message handler for a consumer
func (cm *ConsumerManager) RegisterHandler(consumerName string, handler MessageHandler) {
	cm.handlers[consumerName] = handler
	zap.L().Debug("Handler registered for consumer", zap.String("consumer", consumerName))
}

// RegisterHandlerFunc registers a function as a message handler
func (cm *ConsumerManager) RegisterHandlerFunc(consumerName string, handlerFunc func(context.Context, []byte) error) {
	cm.RegisterHandler(consumerName, MessageHandlerFunc(handlerFunc))
}

// StartConsumers initializes and starts all configured consumers
func (cm *ConsumerManager) StartConsumers(ctx context.Context) error {
	zap.L().Info("Starting RabbitMQ consumers", zap.Int("count", len(cm.configs)))

	for _, consumerConfig := range cm.configs {
		// Get handler for this consumer
		handler, exists := cm.handlers[consumerConfig.Name]
		if !exists {
			zap.L().Warn("No handler registered for consumer, skipping",
				zap.String("consumer", consumerConfig.Name))
			continue
		}

		consumer, err := cm.createConsumer(consumerConfig, handler)
		if err != nil {
			return fmt.Errorf("failed to create consumer %s: %w", consumerConfig.Name, err)
		}

		cm.consumers[consumerConfig.Name] = consumer

		// Start consumer in background
		if err := consumer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start consumer %s: %w", consumerConfig.Name, err)
		}
	}

	return nil
}

// createConsumer creates a single consumer with dedicated channel
func (cm *ConsumerManager) createConsumer(consumerConfig config.RabbitMQConsumerConfig, handler MessageHandler) (*Consumer, error) {
	// Create dedicated channel for this consumer
	channel, err := cm.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Set QoS (prefetch count)
	if consumerConfig.Prefetch > 0 {
		err = channel.Qos(
			consumerConfig.Prefetch, // prefetch count
			0,                       // prefetch size (0 = no limit)
			false,                   // global (false = per consumer)
		)
		if err != nil {
			channel.Close()
			return nil, fmt.Errorf("failed to set QoS: %w", err)
		}
		zap.L().Debug("QoS set for consumer",
			zap.String("consumer", consumerConfig.Name),
			zap.Int("prefetch", consumerConfig.Prefetch))
	}

	consumer := &Consumer{
		name:     consumerConfig.Name,
		channel:  channel,
		config:   consumerConfig,
		handler:  handler,
		stopChan: make(chan struct{}),
	}

	return consumer, nil
}

// Start starts consuming messages with configured concurrency
func (c *Consumer) Start(ctx context.Context) error {
	zap.L().Info("Starting consumer",
		zap.String("consumer", c.name),
		zap.String("queue", c.config.Queue),
		zap.Int("concurrency", c.config.Concurrency))

	// Register consumer with RabbitMQ
	deliveries, err := c.channel.Consume(
		c.config.Queue,   // queue
		c.name,           // consumer tag
		c.config.AutoAck, // auto-ack
		false,            // exclusive
		false,            // no-local
		false,            // no-wait
		nil,              // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	// Start worker goroutines based on concurrency setting
	concurrency := c.config.Concurrency
	if concurrency <= 0 {
		concurrency = 1 // Default to 1 worker
	}

	for i := 0; i < concurrency; i++ {
		c.wg.Add(1)
		go c.worker(ctx, i, deliveries)
	}

	zap.L().Info("Consumer started successfully",
		zap.String("consumer", c.name),
		zap.Int("workers", concurrency))

	return nil
}

// worker processes messages from the deliveries channel
func (c *Consumer) worker(ctx context.Context, workerID int, deliveries <-chan amqp.Delivery) {
	defer c.wg.Done()

	zap.L().Debug("Consumer worker started",
		zap.String("consumer", c.name),
		zap.Int("worker_id", workerID))

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Consumer worker stopped due to context cancellation",
				zap.String("consumer", c.name),
				zap.Int("worker_id", workerID))
			return
		case <-c.stopChan:
			zap.L().Info("Consumer worker stopped",
				zap.String("consumer", c.name),
				zap.Int("worker_id", workerID))
			return
		case delivery, ok := <-deliveries:
			if !ok {
				zap.L().Warn("Deliveries channel closed",
					zap.String("consumer", c.name),
					zap.Int("worker_id", workerID))
				return
			}
			c.handleDelivery(ctx, delivery, workerID)
		}
	}
}

// handleDelivery processes a single message delivery
func (c *Consumer) handleDelivery(ctx context.Context, delivery amqp.Delivery, workerID int) {
	startTime := time.Now()

	zap.L().Info("Processing message",
		zap.String("consumer", c.name),
		zap.Int("worker_id", workerID),
		zap.String("message_id", delivery.MessageId),
		zap.Uint64("delivery_tag", delivery.DeliveryTag),
		zap.Int("message_size", len(delivery.Body)))

	zap.L().Debug("Message details",
		zap.String("consumer", c.name),
		zap.Int("worker_id", workerID),
		zap.Uint64("delivery_tag", delivery.DeliveryTag),
		zap.Any("headers", delivery.Headers),
		zap.Any("body", delivery.Body),
	)

	defer func() {
		if r := recover(); r != nil {
			zap.L().Error("Panic recovered while processing message",
				zap.String("consumer", c.name),
				zap.Int("worker_id", workerID),
				zap.Uint64("delivery_tag", delivery.DeliveryTag),
				zap.Any("panic", r))
			c.handleError(delivery, fmt.Errorf("panic: %v", r), workerID, time.Since(startTime))
		}
	}()

	// Call the message handler
	err := c.handler.Handle(ctx, delivery.Body)

	duration := time.Since(startTime)

	// Handle result
	if err != nil {
		c.handleError(delivery, err, workerID, duration)
	} else {
		c.handleSuccess(delivery, workerID, duration)
	}
}

// handleSuccess handles successful message processing
func (c *Consumer) handleSuccess(delivery amqp.Delivery, workerID int, duration time.Duration) {
	// If auto-ack is disabled, manually acknowledge
	if !c.config.AutoAck {
		if err := delivery.Ack(false); err != nil {
			zap.L().Error("Failed to acknowledge message",
				zap.String("consumer", c.name),
				zap.Int("worker_id", workerID),
				zap.Uint64("delivery_tag", delivery.DeliveryTag),
				zap.Error(err))
			return
		}
	}

	zap.L().Debug("Message processed successfully",
		zap.String("consumer", c.name),
		zap.Int("worker_id", workerID),
		zap.Uint64("delivery_tag", delivery.DeliveryTag),
		zap.Duration("duration", duration))
}

// handleError handles message processing errors
func (c *Consumer) handleError(delivery amqp.Delivery, err error, workerID int, duration time.Duration) {
	zap.L().Error("Failed to process message",
		zap.String("consumer", c.name),
		zap.Int("worker_id", workerID),
		zap.Uint64("delivery_tag", delivery.DeliveryTag),
		zap.Duration("duration", duration),
		zap.Error(err))

	// If auto-ack is disabled, decide whether to requeue or reject
	if !c.config.AutoAck {
		// Get retry count from message headers
		retryCount := getRetryCount(delivery.Headers)

		if c.config.RequeueOnError && retryCount == 0 {
			// Requeue for first failure
			if nackErr := delivery.Nack(false, true); nackErr != nil {
				zap.L().Error("Failed to nack and requeue message",
					zap.String("consumer", c.name),
					zap.Error(nackErr))
			} else {
				zap.L().Info("Message requeued",
					zap.String("consumer", c.name),
					zap.Uint64("delivery_tag", delivery.DeliveryTag))
			}
		} else {
			// Reject message (goes to DLQ if configured)
			if nackErr := delivery.Nack(false, false); nackErr != nil {
				zap.L().Error("Failed to nack message",
					zap.String("consumer", c.name),
					zap.Error(nackErr))
			} else {
				zap.L().Info("Message rejected (sent to DLQ)",
					zap.String("consumer", c.name),
					zap.Uint64("delivery_tag", delivery.DeliveryTag),
					zap.Int("retry_count", retryCount))
			}
		}
	}
}

// getRetryCount extracts retry count from message headers
func getRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}

	if count, ok := headers["x-retry-count"].(int32); ok {
		return int(count)
	}
	if count, ok := headers["x-retry-count"].(int); ok {
		return count
	}

	return 0
}

// Stop stops the consumer gracefully
func (c *Consumer) Stop() error {
	zap.L().Info("Stopping consumer", zap.String("consumer", c.name))

	// Signal workers to stop
	close(c.stopChan)

	// Cancel consumer on RabbitMQ side
	if err := c.channel.Cancel(c.name, false); err != nil {
		zap.L().Error("Failed to cancel consumer", zap.String("consumer", c.name), zap.Error(err))
	}

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		zap.L().Info("All consumer workers stopped", zap.String("consumer", c.name))
	case <-time.After(30 * time.Second):
		zap.L().Warn("Consumer workers did not stop gracefully within timeout",
			zap.String("consumer", c.name))
	}

	// Close channel
	if !c.channel.IsClosed() {
		if err := c.channel.Close(); err != nil {
			return fmt.Errorf("failed to close channel: %w", err)
		}
	}

	return nil
}

// StopAllConsumers stops all consumers gracefully
func (cm *ConsumerManager) StopAllConsumers() error {
	zap.L().Info("Stopping all consumers", zap.Int("count", len(cm.consumers)))

	var lastErr error
	for name, consumer := range cm.consumers {
		if err := consumer.Stop(); err != nil {
			zap.L().Error("Failed to stop consumer", zap.String("consumer", name), zap.Error(err))
			lastErr = err
		}
	}

	return lastErr
}

// GetConsumer returns a consumer by name
func (cm *ConsumerManager) GetConsumer(name string) (*Consumer, error) {
	consumer, exists := cm.consumers[name]
	if !exists {
		return nil, fmt.Errorf("consumer %s not found", name)
	}
	return consumer, nil
}

// IsHealthy checks if the consumer channel is still open
func (c *Consumer) IsHealthy() bool {
	return c.channel != nil && !c.channel.IsClosed()
}

// GetConfig returns the consumer configuration
func (c *Consumer) GetConfig() config.RabbitMQConsumerConfig {
	return c.config
}

// CalculateBackoff calculates exponential backoff delay
func CalculateBackoff(baseMs int, attempt int, multiplier float64) time.Duration {
	if multiplier <= 0 {
		multiplier = 1.5 // Default multiplier
	}
	backoffMs := float64(baseMs) * math.Pow(multiplier, float64(attempt))
	return time.Duration(backoffMs) * time.Millisecond
}
