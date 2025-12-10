// Package responses defines DTOs for RabbitMQ management API responses.
package responses

// RabbitMQOverviewResponse represents the RabbitMQ system overview
type RabbitMQOverviewResponse struct {
	TotalQueues          int                      `json:"total_queues"`
	TotalExchanges       int                      `json:"total_exchanges"`
	TotalMessages        int64                    `json:"total_messages"`
	TotalMessagesReady   int64                    `json:"total_messages_ready"`
	TotalMessagesUnacked int64                    `json:"total_messages_unacked"`
	ConnectionStatus     string                   `json:"connection_status"`
	QueueSummary         RabbitMQQueueSummary     `json:"queue_summary"`
	ConfiguredTopology   *RabbitMQTopologySummary `json:"configured_topology,omitempty"`
}

// RabbitMQQueueSummary summarizes queue statistics by type
type RabbitMQQueueSummary struct {
	MainQueues       int   `json:"main_queues"`
	RetryQueues      int   `json:"retry_queues"`
	DeadLetterQueues int   `json:"dead_letter_queues"`
	DelayedQueues    int   `json:"delayed_queues"`
	TotalDLQMessages int64 `json:"total_dlq_messages"`
}

// RabbitMQTopologySummary summarizes the configured topology
type RabbitMQTopologySummary struct {
	ExchangeCount int      `json:"exchange_count"`
	QueueCount    int      `json:"queue_count"`
	ExchangeNames []string `json:"exchange_names"`
}

// RabbitMQQueueResponse represents a queue with additional metadata
type RabbitMQQueueResponse struct {
	Name            string                 `json:"name"`
	Type            string                 `json:"type"` // main, retry, dead_letter, delayed
	MainQueueName   string                 `json:"main_queue_name,omitempty"`
	VHost           string                 `json:"vhost"`
	Durable         bool                   `json:"durable"`
	AutoDelete      bool                   `json:"auto_delete"`
	Messages        int64                  `json:"messages"`
	MessagesReady   int64                  `json:"messages_ready"`
	MessagesUnacked int64                  `json:"messages_unacknowledged"`
	Consumers       int                    `json:"consumers"`
	State           string                 `json:"state"`
	Arguments       map[string]interface{} `json:"arguments,omitempty"`
	HasRetryQueue   bool                   `json:"has_retry_queue,omitempty"`
	HasDLQ          bool                   `json:"has_dlq,omitempty"`
	RetryQueueName  string                 `json:"retry_queue_name,omitempty"`
	DLQName         string                 `json:"dlq_name,omitempty"`
	MessageRate     *RabbitMQRateInfo      `json:"message_rate,omitempty"`
}

// RabbitMQRateInfo contains message rate information
type RabbitMQRateInfo struct {
	PublishRate   float64 `json:"publish_rate"`
	DeliverRate   float64 `json:"deliver_rate"`
	AckRate       float64 `json:"ack_rate"`
	RedeliverRate float64 `json:"redeliver_rate"`
}

// RabbitMQExchangeResponse represents an exchange
type RabbitMQExchangeResponse struct {
	Name       string                    `json:"name"`
	VHost      string                    `json:"vhost"`
	Type       string                    `json:"type"`
	Durable    bool                      `json:"durable"`
	AutoDelete bool                      `json:"auto_delete"`
	Internal   bool                      `json:"internal"`
	Arguments  map[string]interface{}    `json:"arguments,omitempty"`
	Bindings   []RabbitMQBindingResponse `json:"bindings,omitempty"`
}

// RabbitMQBindingResponse represents a binding between exchange and queue
type RabbitMQBindingResponse struct {
	Source          string `json:"source"`
	Destination     string `json:"destination"`
	DestinationType string `json:"destination_type"`
	RoutingKey      string `json:"routing_key"`
}

// RabbitMQMessageResponse represents a message from a queue
type RabbitMQMessageResponse struct {
	PayloadBytes    int64                             `json:"payload_bytes"`
	Redelivered     bool                              `json:"redelivered"`
	Exchange        string                            `json:"exchange"`
	RoutingKey      string                            `json:"routing_key"`
	MessageCount    int64                             `json:"message_count"`
	Payload         string                            `json:"payload"`
	PayloadEncoding string                            `json:"payload_encoding"`
	Properties      RabbitMQMessagePropertiesResponse `json:"properties"`
}

// RabbitMQMessagePropertiesResponse represents message properties
type RabbitMQMessagePropertiesResponse struct {
	ContentType   string                 `json:"content_type,omitempty"`
	Headers       map[string]interface{} `json:"headers,omitempty"`
	DeliveryMode  int                    `json:"delivery_mode,omitempty"`
	Priority      int                    `json:"priority,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	MessageID     string                 `json:"message_id,omitempty"`
	Timestamp     int64                  `json:"timestamp,omitempty"`
	Type          string                 `json:"type,omitempty"`
	AppID         string                 `json:"app_id,omitempty"`
}

// RabbitMQShovelStatusResponse represents a shovel status
type RabbitMQShovelStatusResponse struct {
	Name      string `json:"name"`
	VHost     string `json:"vhost"`
	Type      string `json:"type"`
	State     string `json:"state"`
	Timestamp string `json:"timestamp,omitempty"`
}

// RabbitMQDLQRetryResponse represents the result of a DLQ retry operation
type RabbitMQDLQRetryResponse struct {
	SourceQueue      string `json:"source_queue"`
	DestinationQueue string `json:"destination_queue"`
	ShovelName       string `json:"shovel_name"`
	Status           string `json:"status"`
	Message          string `json:"message"`
}

// RabbitMQPurgeResponse represents the result of a queue purge operation
type RabbitMQPurgeResponse struct {
	QueueName       string `json:"queue_name"`
	MessagesDeleted int64  `json:"messages_deleted"`
	Status          string `json:"status"`
}

// RabbitMQQueueGroupResponse groups queues by their main queue
type RabbitMQQueueGroupResponse struct {
	MainQueue     *RabbitMQQueueResponse `json:"main_queue"`
	RetryQueue    *RabbitMQQueueResponse `json:"retry_queue,omitempty"`
	DLQ           *RabbitMQQueueResponse `json:"dlq,omitempty"`
	DelayQueue    *RabbitMQQueueResponse `json:"delay_queue,omitempty"`
	TotalMessages int64                  `json:"total_messages"`
}

// RabbitMQHealthResponse represents the health status of RabbitMQ
type RabbitMQHealthResponse struct {
	Connected       bool                   `json:"connected"`
	ManagementAPI   bool                   `json:"management_api"`
	ClusterName     string                 `json:"cluster_name,omitempty"`
	RabbitMQVersion string                 `json:"rabbitmq_version,omitempty"`
	ErlangVersion   string                 `json:"erlang_version,omitempty"`
	NodeName        string                 `json:"node_name,omitempty"`
	Details         map[string]interface{} `json:"details,omitempty"`
}
