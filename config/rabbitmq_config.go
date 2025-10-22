package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// RabbitMQTopologyConfig defines the infrastructure (exchanges, queues, bindings)
type RabbitMQTopologyConfig struct {
	Exchanges []RabbitMQExchangeConfig `mapstructure:"exchanges" json:"exchanges" yaml:"exchanges"`
}

// RabbitMQExchangeConfig defines an exchange and its queues
type RabbitMQExchangeConfig struct {
	Name       string                `mapstructure:"name" json:"name" yaml:"name"`
	Type       string                `mapstructure:"type" json:"type" yaml:"type"` // direct, topic, fanout, headers
	Durable    bool                  `mapstructure:"durable" json:"durable" yaml:"durable"`
	AutoDelete bool                  `mapstructure:"autoDelete" json:"autoDelete" yaml:"autoDelete"`
	Queues     []RabbitMQQueueConfig `mapstructure:"queues" json:"queues" yaml:"queues"`
}

// RabbitMQQueueConfig defines a queue with retry and DLQ settings
type RabbitMQQueueConfig struct {
	Name       string              `mapstructure:"name" json:"name" yaml:"name"`
	RoutingKey string              `mapstructure:"routingKey" json:"routingKey" yaml:"routingKey"`
	Durable    bool                `mapstructure:"durable" json:"durable" yaml:"durable"`
	AutoDelete bool                `mapstructure:"autoDelete" json:"autoDelete" yaml:"autoDelete"`
	Prefetch   int                 `mapstructure:"prefetch" json:"prefetch" yaml:"prefetch"`
	Retry      RabbitMQRetryConfig `mapstructure:"retry" json:"retry" yaml:"retry"`
	DLQ        RabbitMQDLQConfig   `mapstructure:"dlq" json:"dlq" yaml:"dlq"`
}

// RabbitMQRetryConfig defines retry queue settings
type RabbitMQRetryConfig struct {
	Enabled             bool    `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Suffix              string  `mapstructure:"suffix" json:"suffix" yaml:"suffix"`                                        // e.g., "retry"
	TTLMs               int     `mapstructure:"ttlMs" json:"ttlMs" yaml:"ttlMs"`                                           // Time-to-live in milliseconds
	MaxAttempts         int     `mapstructure:"maxAttempts" json:"maxAttempts" yaml:"maxAttempts"`                         // Max retry attempts
	ExponentialMultiply float64 `mapstructure:"exponentialMultiply" json:"exponentialMultiply" yaml:"exponentialMultiply"` // Exponential backoff multiplier
	Exchange            string  `mapstructure:"exchange" json:"exchange" yaml:"exchange"`                                  // Exchange to route back to
}

// RabbitMQDLQConfig defines dead letter queue settings
type RabbitMQDLQConfig struct {
	Enabled  bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Suffix   string `mapstructure:"suffix" json:"suffix" yaml:"suffix"`       // e.g., "dlq"
	Exchange string `mapstructure:"exchange" json:"exchange" yaml:"exchange"` // Exchange for DLQ routing
}

// RabbitMQProducerConfig defines how to publish messages
type RabbitMQProducerConfig struct {
	Name       string                      `mapstructure:"name" json:"name" yaml:"name"`
	Exchange   string                      `mapstructure:"exchange" json:"exchange" yaml:"exchange"`
	RoutingKey string                      `mapstructure:"routingKey" json:"routingKey" yaml:"routingKey"`
	Confirm    bool                        `mapstructure:"confirm" json:"confirm" yaml:"confirm"`          // Publisher confirms
	Mandatory  bool                        `mapstructure:"mandatory" json:"mandatory" yaml:"mandatory"`    // Return unroutable messages
	Persistent bool                        `mapstructure:"persistent" json:"persistent" yaml:"persistent"` // Message durability
	Retry      RabbitMQProducerRetryConfig `mapstructure:"retry" json:"retry" yaml:"retry"`
}

// RabbitMQProducerRetryConfig defines retry settings for producers
type RabbitMQProducerRetryConfig struct {
	Enabled             bool    `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	MaxAttempts         int     `mapstructure:"maxAttempts" json:"maxAttempts" yaml:"maxAttempts"`
	BackoffMs           int     `mapstructure:"backoffMs" json:"backoffMs" yaml:"backoffMs"`
	ExponentialMultiply float64 `mapstructure:"exponentialMultiply" json:"exponentialMultiply" yaml:"exponentialMultiply"`
}

// RabbitMQConsumerConfig defines how to consume messages
type RabbitMQConsumerConfig struct {
	Name                string  `mapstructure:"name" json:"name" yaml:"name"`
	Queue               string  `mapstructure:"queue" json:"queue" yaml:"queue"`
	Prefetch            int     `mapstructure:"prefetch" json:"prefetch" yaml:"prefetch"`                   // QoS: max unacked messages
	Concurrency         int     `mapstructure:"concurrency" json:"concurrency" yaml:"concurrency"`          // Number of worker goroutines
	AutoAck             bool    `mapstructure:"autoAck" json:"autoAck" yaml:"autoAck"`                      // Auto-acknowledgment
	RequeueOnError      bool    `mapstructure:"requeueOnError" json:"requeueOnError" yaml:"requeueOnError"` // Requeue on error vs DLQ
	BackoffMs           int     `mapstructure:"backoffMs" json:"backoffMs" yaml:"backoffMs"`
	ExponentialMultiply float64 `mapstructure:"exponentialMultiply" json:"exponentialMultiply" yaml:"exponentialMultiply"`
}

// loadRabbitMQConfig loads the RabbitMQ advanced configuration from rabbitmq-config.yaml
func loadRabbitMQConfig(configPath string) error {
	// Create a new viper instance for RabbitMQ config
	rabbitViper := viper.New()
	rabbitViper.AddConfigPath(configPath)
	rabbitViper.SetConfigName("rabbitmq_config")
	rabbitViper.SetConfigType("yaml")

	if err := rabbitViper.ReadInConfig(); err != nil {
		// If file doesn't exist, that's okay - return nil (optional config)
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return fmt.Errorf("rabbitmq_config.yaml not found (this is optional)")
		}
		return fmt.Errorf("error reading rabbitmq_config.yaml: %w", err)
	}

	var rabbitMQConfig RabbitMQConfig
	if err := rabbitViper.Unmarshal(&rabbitMQConfig); err != nil {
		return fmt.Errorf("unable to decode rabbitmq_config.yaml into struct: %w", err)
	}

	applyRabbitMQConfigDefaults(&rabbitMQConfig)

	appConfig.RabbitMQ.Topology = rabbitMQConfig.Topology
	appConfig.RabbitMQ.Producers = rabbitMQConfig.Producers
	appConfig.RabbitMQ.Consumers = rabbitMQConfig.Consumers

	return nil
}

// applyRabbitMQConfigDefaults sets default values for RabbitMQ configuration
func applyRabbitMQConfigDefaults(cfg *RabbitMQConfig) {
	for i := range cfg.Topology.Exchanges {
		ex := &cfg.Topology.Exchanges[i]
		if ex.Type == "" {
			ex.Type = "topic"
		}
		for j := range ex.Queues {
			q := &ex.Queues[j]

			// Queue-level defaults
			if q.Prefetch <= 0 {
				q.Prefetch = 1
			}

			// Retry defaults
			if q.Retry.Enabled {
				if q.Retry.Suffix == "" {
					q.Retry.Suffix = "retry"
				}
				if q.Retry.TTLMs == 0 {
					q.Retry.TTLMs = 60000
				}
				if q.Retry.MaxAttempts == 0 {
					q.Retry.MaxAttempts = 3
				}
				if q.Retry.ExponentialMultiply == 0 {
					q.Retry.ExponentialMultiply = 1.5
				}
				if q.Retry.Exchange == "" {
					q.Retry.Exchange = ex.Name
				}
			}

			// DLQ defaults
			if q.DLQ.Enabled {
				if q.DLQ.Suffix == "" {
					q.DLQ.Suffix = "dlq"
				}
				// Usually DLQs don’t need an exchange unless explicitly set
			}
		}
	}

	// Producer defaults
	for i := range cfg.Producers {
		p := &cfg.Producers[i]
		if p.Retry.Enabled {
			if p.Retry.MaxAttempts == 0 {
				p.Retry.MaxAttempts = 3
			}
			if p.Retry.BackoffMs == 0 {
				p.Retry.BackoffMs = 1000
			}
			if p.Retry.ExponentialMultiply == 0 {
				p.Retry.ExponentialMultiply = 2.0
			}
		}
	}

	// Consumer defaults
	for i := range cfg.Consumers {
		c := &cfg.Consumers[i]
		if c.Prefetch <= 0 {
			c.Prefetch = 1
		}
		if c.BackoffMs == 0 {
			c.BackoffMs = 2000
		}
		if c.ExponentialMultiply == 0 {
			c.ExponentialMultiply = 1.5
		}
	}
}
