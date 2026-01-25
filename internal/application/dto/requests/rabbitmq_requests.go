// Package requests defines DTOs for RabbitMQ management API requests.
package requests

// RabbitMQGetMessagesRequest represents a request to get messages from a queue
type RabbitMQGetMessagesRequest struct {
	Count    int    `form:"count" json:"count" validate:"omitempty,min=1,max=100"`
	AckMode  string `form:"ack_mode" json:"ack_mode" validate:"omitempty,oneof=ack_requeue_true ack_requeue_false reject_requeue_true reject_requeue_false"`
	Encoding string `form:"encoding" json:"encoding" validate:"omitempty,oneof=auto base64"`
}

// RabbitMQRetryDLQRequest represents a request to retry messages from a DLQ
type RabbitMQRetryDLQRequest struct {
	SourceQueue      string `json:"source_queue" validate:"required"`
	DestinationQueue string `json:"destination_queue,omitempty"`
	DestExchange     string `json:"dest_exchange,omitempty"`
	DestRoutingKey   string `json:"dest_routing_key,omitempty"`
}

// RabbitMQPublishMessageRequest represents a request to publish a message
type RabbitMQPublishMessageRequest struct {
	Exchange      string                 `json:"exchange" validate:"required"`
	RoutingKey    string                 `json:"routing_key" validate:"required"`
	Payload       string                 `json:"payload" validate:"required"`
	ContentType   string                 `json:"content_type,omitempty"`
	Headers       map[string]interface{} `json:"headers,omitempty"`
	DeliveryMode  int                    `json:"delivery_mode,omitempty" validate:"omitempty,oneof=1 2"` // 1=non-persistent, 2=persistent
	Priority      int                    `json:"priority,omitempty" validate:"omitempty,min=0,max=9"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	MessageID     string                 `json:"message_id,omitempty"`
}

// RabbitMQQueueFilterRequest represents filter options for listing queues
type RabbitMQQueueFilterRequest struct {
	Type        string `form:"type" json:"type" validate:"omitempty,oneof=main retry dead_letter delayed all"`
	Search      string `form:"search" json:"search"`
	HasMessages bool   `form:"has_messages" json:"has_messages"`
}
