// Package rabbitmq provides a wrapper around the RabbitMQ client.
package rabbitmq

import (
	"context"
	"core-backend/config"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// RabbitMQ is the main RabbitMQ manager that orchestrates topology, producers, and consumers
type RabbitMQ struct {
	conn            *amqp.Connection
	config          *config.RabbitMQConfig
	topologyManager *TopologyManager
	producerManager *ProducerManager
	consumerManager *ConsumerManager
}

// NewRabbitMQ creates a new RabbitMQ manager with topology, producers, and consumers
func NewRabbitMQ() (*RabbitMQ, error) {
	zap.L().Info("Initializing RabbitMQ connection")

	appConfig := config.GetAppConfig()
	cfg := &appConfig.RabbitMQ

	zap.L().Debug("RabbitMQ configuration loaded",
		zap.String("url", cfg.URL),
		zap.String("vhost", cfg.VHost),
		zap.Int("exchanges", len(cfg.Topology.Exchanges)),
		zap.Int("producers", len(cfg.Producers)),
		zap.Int("consumers", len(cfg.Consumers)))

	// Connect to RabbitMQ
	zap.L().Debug("Attempting to connect to RabbitMQ server")
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		zap.L().Error("Failed to connect to RabbitMQ",
			zap.String("url", cfg.URL),
			zap.Error(err))
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	rabbitmq := &RabbitMQ{
		conn:   conn,
		config: cfg,
	}

	// Setup topology (exchanges, queues, bindings with retry/DLQ)
	if len(cfg.Topology.Exchanges) > 0 {
		zap.L().Info("Setting up RabbitMQ topology")
		rabbitmq.topologyManager = NewTopologyManager(conn, &cfg.Topology)
		if err := rabbitmq.topologyManager.SetupTopology(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to setup topology: %w", err)
		}
		zap.L().Info("RabbitMQ topology setup completed successfully")
	}

	// Initialize producers
	if len(cfg.Producers) > 0 {
		zap.L().Info("Initializing RabbitMQ producers", zap.Int("count", len(cfg.Producers)))
		rabbitmq.producerManager = NewProducerManager(conn, cfg.Producers)
		if err := rabbitmq.producerManager.InitializeProducers(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to initialize producers: %w", err)
		}
		zap.L().Info("RabbitMQ producers initialized successfully")
	}

	// Initialize consumer manager (consumers will be started later via StartConsumers)
	if len(cfg.Consumers) > 0 {
		zap.L().Info("Initializing RabbitMQ consumer manager", zap.Int("count", len(cfg.Consumers)))
		rabbitmq.consumerManager = NewConsumerManager(conn, cfg.Consumers)
		zap.L().Info("RabbitMQ consumer manager initialized successfully")
	}

	zap.L().Info("RabbitMQ connection established successfully")
	return rabbitmq, nil
}

// ===== Producer Methods =====

// GetProducer returns a producer by name
func (r *RabbitMQ) GetProducer(name string) (*Producer, error) {
	if r.producerManager == nil {
		return nil, fmt.Errorf("producer manager not initialized")
	}
	return r.producerManager.GetProducer(name)
}

// ===== Consumer Methods =====

// RegisterConsumerHandler registers a message handler for a consumer
func (r *RabbitMQ) RegisterConsumerHandler(consumerName string, handler MessageHandler) error {
	if r.consumerManager == nil {
		return fmt.Errorf("consumer manager not initialized")
	}
	r.consumerManager.RegisterHandler(consumerName, handler)
	return nil
}

// RegisterConsumerHandlerFunc registers a function as a message handler
func (r *RabbitMQ) RegisterConsumerHandlerFunc(consumerName string, handlerFunc func(context.Context, []byte) error) error {
	if r.consumerManager == nil {
		return fmt.Errorf("consumer manager not initialized")
	}
	r.consumerManager.RegisterHandlerFunc(consumerName, handlerFunc)
	return nil
}

// StartConsumers starts all configured consumers
func (r *RabbitMQ) StartConsumers(ctx context.Context) error {
	if r.consumerManager == nil {
		return fmt.Errorf("consumer manager not initialized")
	}
	return r.consumerManager.StartConsumers(ctx)
}

// StopConsumers stops all consumers gracefully
func (r *RabbitMQ) StopConsumers() error {
	if r.consumerManager == nil {
		return nil // No consumers to stop
	}
	return r.consumerManager.StopAllConsumers()
}

// ===== Connection Management =====

// Close closes the RabbitMQ connection and all managers
func (r *RabbitMQ) Close() error {
	zap.L().Info("Closing RabbitMQ connection")

	var lastErr error

	// Stop consumers
	if err := r.StopConsumers(); err != nil {
		zap.L().Error("Failed to stop consumers", zap.Error(err))
		lastErr = err
	}

	// Close producers
	if r.producerManager != nil {
		if err := r.producerManager.Close(); err != nil {
			zap.L().Error("Failed to close producers", zap.Error(err))
			lastErr = err
		}
	}

	// Close connection
	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			zap.L().Error("Failed to close connection", zap.Error(err))
			lastErr = err
		}
	}

	zap.L().Info("RabbitMQ connection closed")
	return lastErr
}

// IsConnected checks if the connection is still alive
func (r *RabbitMQ) IsConnected() bool {
	return r.conn != nil && !r.conn.IsClosed()
}

// GetConnection returns the underlying AMQP connection
func (r *RabbitMQ) GetConnection() *amqp.Connection {
	return r.conn
}

// GetConfig returns the RabbitMQ configuration
func (r *RabbitMQ) GetConfig() *config.RabbitMQConfig {
	return r.config
}
