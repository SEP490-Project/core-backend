// Package rabbitmq provides RabbitMQ topology management.
package rabbitmq

import (
	"core-backend/config"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// TopologyManager handles RabbitMQ topology setup (exchanges, queues, bindings)
type TopologyManager struct {
	conn   *amqp.Connection
	config *config.RabbitMQTopologyConfig
}

// NewTopologyManager creates a new topology manager
func NewTopologyManager(conn *amqp.Connection, topologyConfig *config.RabbitMQTopologyConfig) *TopologyManager {
	return &TopologyManager{
		conn:   conn,
		config: topologyConfig,
	}
}

// SetupTopology declares all exchanges, queues, retry queues, DLQs, and bindings
func (tm *TopologyManager) SetupTopology() error {
	if tm.config == nil {
		return fmt.Errorf("topology config is nil")
	}

	zap.L().Info("Setting up RabbitMQ topology",
		zap.Int("exchanges", len(tm.config.Exchanges)))

	// Create a channel for topology setup
	channel, err := tm.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel for topology setup: %w", err)
	}
	defer func() {
		if !channel.IsClosed() {
			if err := channel.Close(); err != nil {
				zap.L().Error("Failed to close channel", zap.Error(err))
			}
		}
	}()

	// Declare all exchanges and their queues
	for _, exchangeConfig := range tm.config.Exchanges {
		if err := tm.setupExchange(channel, exchangeConfig); err != nil {
			return fmt.Errorf("failed to setup exchange %s: %w", exchangeConfig.Name, err)
		}
	}

	zap.L().Info("RabbitMQ topology setup completed successfully")
	return nil
}

// setupExchange declares an exchange and all its associated queues
func (tm *TopologyManager) setupExchange(channel *amqp.Channel, exchangeConfig config.RabbitMQExchangeConfig) error {
	zap.L().Debug("Declaring exchange",
		zap.String("exchange", exchangeConfig.Name),
		zap.String("type", exchangeConfig.Type))

	// Declare the exchange
	err := channel.ExchangeDeclare(
		exchangeConfig.Name,       // name
		exchangeConfig.Type,       // type (direct, topic, fanout, headers)
		exchangeConfig.Durable,    // durable
		exchangeConfig.AutoDelete, // auto-deleted
		false,                     // internal
		false,                     // no-wait
		exchangeConfig.Arguments,  // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange %s: %w", exchangeConfig.Name, err)
	}

	zap.L().Debug("Exchange declared successfully", zap.String("exchange", exchangeConfig.Name))

	// Declare all queues for this exchange
	for _, queueConfig := range exchangeConfig.Queues {
		if err := tm.setupQueue(channel, exchangeConfig.Name, queueConfig); err != nil {
			return fmt.Errorf("failed to setup queue %s: %w", queueConfig.Name, err)
		}
	}

	return nil
}

// setupQueue declares a main queue, retry queue (if enabled), DLQ (if enabled), and bindings
func (tm *TopologyManager) setupQueue(channel *amqp.Channel, exchangeName string, queueConfig config.RabbitMQQueueConfig) error {
	zap.L().Debug("Setting up queue",
		zap.String("queue", queueConfig.Name),
		zap.String("routing_key", queueConfig.RoutingKey))

	// 1. Setup DLQ first (if enabled) - DLQ has no further routing
	var dlqName string
	if queueConfig.DLQ.Enabled {
		dlqName = queueConfig.Name + "." + queueConfig.DLQ.Suffix
		if err := tm.declareDLQ(channel, dlqName, queueConfig); err != nil {
			return fmt.Errorf("failed to declare DLQ %s: %w", dlqName, err)
		}
	}

	// 2. Setup Retry Queue (if enabled) - routes back to main queue after TTL
	var retryQueueName string
	if queueConfig.Retry.Enabled {
		retryQueueName = queueConfig.Name + "." + queueConfig.Retry.Suffix
		retryExchange := queueConfig.Retry.Exchange
		if retryExchange == "" {
			retryExchange = exchangeName // Default to parent exchange
		}

		if err := tm.declareRetryQueue(channel, retryQueueName, retryExchange, queueConfig); err != nil {
			return fmt.Errorf("failed to declare retry queue %s: %w", retryQueueName, err)
		}
	}

	// 3. Setup Main Queue - can route to retry queue or DLQ on rejection
	if err := tm.declareMainQueue(channel, queueConfig, dlqName); err != nil {
		return fmt.Errorf("failed to declare main queue %s: %w", queueConfig.Name, err)
	}

	// 4. Bind Main Queue to Exchange
	zap.L().Debug("Binding queue to exchange",
		zap.String("queue", queueConfig.Name),
		zap.String("exchange", exchangeName),
		zap.String("routing_key", queueConfig.RoutingKey))

	err := channel.QueueBind(
		queueConfig.Name,       // queue name
		queueConfig.RoutingKey, // routing key
		exchangeName,           // exchange
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue %s to exchange %s: %w", queueConfig.Name, exchangeName, err)
	}

	// 5. Bind Additional Bindings
	for _, binding := range queueConfig.AdditionalBindings {
		zap.L().Debug("Binding queue to exchange (additional)",
			zap.String("queue", queueConfig.Name),
			zap.String("exchange", exchangeName),
			zap.String("routing_key", binding))

		err := channel.QueueBind(
			queueConfig.Name, // queue name
			binding,          // routing key
			exchangeName,     // exchange
			false,            // no-wait
			nil,              // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue %s to exchange %s with key %s: %w", queueConfig.Name, exchangeName, binding, err)
		}
	}

	zap.L().Info("Queue setup completed",
		zap.String("queue", queueConfig.Name),
		zap.String("exchange", exchangeName),
		zap.Bool("has_retry", queueConfig.Retry.Enabled),
		zap.Bool("has_dlq", queueConfig.DLQ.Enabled))

	return nil
}

// declareMainQueue declares the main queue with optional DLX routing
func (tm *TopologyManager) declareMainQueue(channel *amqp.Channel, queueConfig config.RabbitMQQueueConfig, dlqName string) error {
	args := amqp.Table{}

	// If DLQ is enabled, set dead-letter exchange for rejected messages
	if queueConfig.DLQ.Enabled && dlqName != "" {
		// Messages rejected/expired go to DLQ exchange
		dlqExchange := queueConfig.DLQ.Exchange
		if dlqExchange == "" {
			dlqExchange = "" // Direct routing to DLQ queue
		}
		args["x-dead-letter-exchange"] = dlqExchange
		args["x-dead-letter-routing-key"] = dlqName
		zap.L().Debug("Main queue configured with DLQ",
			zap.String("queue", queueConfig.Name),
			zap.String("dlq_name", dlqName))
	}

	_, err := channel.QueueDeclare(
		queueConfig.Name,       // name
		queueConfig.Durable,    // durable
		queueConfig.AutoDelete, // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		args,                   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare main queue: %w", err)
	}

	zap.L().Debug("Main queue declared", zap.String("queue", queueConfig.Name))
	return nil
}

// declareRetryQueue declares a retry queue with TTL that routes back to main queue
func (tm *TopologyManager) declareRetryQueue(channel *amqp.Channel, retryQueueName, retryExchange string, queueConfig config.RabbitMQQueueConfig) error {
	// Retry queue arguments
	args := amqp.Table{
		"x-message-ttl":             queueConfig.Retry.TTLMs, // TTL in milliseconds
		"x-dead-letter-exchange":    retryExchange,           // Route back to original exchange after TTL
		"x-dead-letter-routing-key": queueConfig.RoutingKey,  // Use original routing key
	}

	_, err := channel.QueueDeclare(
		retryQueueName,         // name
		queueConfig.Durable,    // durable
		queueConfig.AutoDelete, // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		args,                   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare retry queue: %w", err)
	}

	zap.L().Debug("Retry queue declared",
		zap.String("retry_queue", retryQueueName),
		zap.Int("ttl_ms", queueConfig.Retry.TTLMs))

	return nil
}

// declareDLQ declares a dead letter queue (no further routing)
func (tm *TopologyManager) declareDLQ(channel *amqp.Channel, dlqName string, queueConfig config.RabbitMQQueueConfig) error {
	_, err := channel.QueueDeclare(
		dlqName,                // name
		queueConfig.Durable,    // durable
		queueConfig.AutoDelete, // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // no arguments - final destination
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLQ: %w", err)
	}

	zap.L().Debug("DLQ declared", zap.String("dlq_name", dlqName))
	return nil
}

// GetQueueNames returns all queue names defined in the topology (including retry and DLQ)
func (tm *TopologyManager) GetQueueNames() []string {
	var queueNames []string

	for _, exchange := range tm.config.Exchanges {
		for _, queue := range exchange.Queues {
			queueNames = append(queueNames, queue.Name)

			if queue.Retry.Enabled {
				queueNames = append(queueNames, queue.Name+"."+queue.Retry.Suffix)
			}

			if queue.DLQ.Enabled {
				queueNames = append(queueNames, queue.Name+"."+queue.DLQ.Suffix)
			}
		}
	}

	return queueNames
}
