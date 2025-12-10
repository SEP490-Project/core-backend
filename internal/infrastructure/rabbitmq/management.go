// Package rabbitmq provides RabbitMQ management functionality.
package rabbitmq

import (
	"context"
	"core-backend/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

// ManagementService provides RabbitMQ management API functionality
type ManagementService struct {
	httpClient     *http.Client
	baseURL        string
	username       string
	password       string
	vhost          string
	topologyConfig *config.RabbitMQTopologyConfig
}

// QueueInfo represents information about a RabbitMQ queue
type QueueInfo struct {
	Name                 string                 `json:"name"`
	VHost                string                 `json:"vhost"`
	Durable              bool                   `json:"durable"`
	AutoDelete           bool                   `json:"auto_delete"`
	Messages             int64                  `json:"messages"`
	MessagesReady        int64                  `json:"messages_ready"`
	MessagesUnacked      int64                  `json:"messages_unacknowledged"`
	Consumers            int                    `json:"consumers"`
	State                string                 `json:"state"`
	Arguments            map[string]interface{} `json:"arguments,omitempty"`
	MessageStats         *MessageStats          `json:"message_stats,omitempty"`
	BackingQueueStatus   *BackingQueueStatus    `json:"backing_queue_status,omitempty"`
	QueueType            string                 `json:"type,omitempty"`
	Policy               string                 `json:"policy,omitempty"`
	IdleSince            string                 `json:"idle_since,omitempty"`
	ExclusiveConsumerTag string                 `json:"exclusive_consumer_tag,omitempty"`
}

// MessageStats contains message statistics for a queue
type MessageStats struct {
	Ack               int64      `json:"ack,omitempty"`
	AckDetails        RateDetail `json:"ack_details,omitempty"`
	Deliver           int64      `json:"deliver,omitempty"`
	DeliverDetails    RateDetail `json:"deliver_details,omitempty"`
	DeliverGet        int64      `json:"deliver_get,omitempty"`
	DeliverGetDetails RateDetail `json:"deliver_get_details,omitempty"`
	Publish           int64      `json:"publish,omitempty"`
	PublishDetails    RateDetail `json:"publish_details,omitempty"`
	Redeliver         int64      `json:"redeliver,omitempty"`
	RedeliverDetails  RateDetail `json:"redeliver_details,omitempty"`
}

// RateDetail contains rate information
type RateDetail struct {
	Rate float64 `json:"rate"`
}

// BackingQueueStatus contains backing queue information
type BackingQueueStatus struct {
	Mode              string  `json:"mode,omitempty"`
	Q1                int64   `json:"q1,omitempty"`
	Q2                int64   `json:"q2,omitempty"`
	Q3                int64   `json:"q3,omitempty"`
	Q4                int64   `json:"q4,omitempty"`
	Len               int64   `json:"len,omitempty"`
	TargetRAMCount    string  `json:"target_ram_count,omitempty"`
	NextSeqID         int64   `json:"next_seq_id,omitempty"`
	AvgIngressRate    float64 `json:"avg_ingress_rate,omitempty"`
	AvgEgressRate     float64 `json:"avg_egress_rate,omitempty"`
	AvgAckIngressRate float64 `json:"avg_ack_ingress_rate,omitempty"`
	AvgAckEgressRate  float64 `json:"avg_ack_egress_rate,omitempty"`
}

// ExchangeInfo represents information about a RabbitMQ exchange
type ExchangeInfo struct {
	Name       string                 `json:"name"`
	VHost      string                 `json:"vhost"`
	Type       string                 `json:"type"`
	Durable    bool                   `json:"durable"`
	AutoDelete bool                   `json:"auto_delete"`
	Internal   bool                   `json:"internal"`
	Arguments  map[string]interface{} `json:"arguments,omitempty"`
}

// BindingInfo represents information about a queue binding
type BindingInfo struct {
	Source          string                 `json:"source"`
	VHost           string                 `json:"vhost"`
	Destination     string                 `json:"destination"`
	DestinationType string                 `json:"destination_type"`
	RoutingKey      string                 `json:"routing_key"`
	Arguments       map[string]interface{} `json:"arguments,omitempty"`
	PropertiesKey   string                 `json:"properties_key,omitempty"`
}

// Message represents a message retrieved from a queue
type Message struct {
	PayloadBytes    int64             `json:"payload_bytes"`
	Redelivered     bool              `json:"redelivered"`
	Exchange        string            `json:"exchange"`
	RoutingKey      string            `json:"routing_key"`
	MessageCount    int64             `json:"message_count"`
	Payload         string            `json:"payload"`
	PayloadEncoding string            `json:"payload_encoding"`
	Properties      MessageProperties `json:"properties"`
}

// MessageProperties contains message metadata
type MessageProperties struct {
	ContentType     string                 `json:"content_type,omitempty"`
	ContentEncoding string                 `json:"content_encoding,omitempty"`
	Headers         map[string]interface{} `json:"headers,omitempty"`
	DeliveryMode    int                    `json:"delivery_mode,omitempty"`
	Priority        int                    `json:"priority,omitempty"`
	CorrelationID   string                 `json:"correlation_id,omitempty"`
	ReplyTo         string                 `json:"reply_to,omitempty"`
	Expiration      string                 `json:"expiration,omitempty"`
	MessageID       string                 `json:"message_id,omitempty"`
	Timestamp       int64                  `json:"timestamp,omitempty"`
	Type            string                 `json:"type,omitempty"`
	UserID          string                 `json:"user_id,omitempty"`
	AppID           string                 `json:"app_id,omitempty"`
}

// ShovelDefinition represents a shovel configuration for moving messages
type ShovelDefinition struct {
	SrcURI          string `json:"src-uri"`
	SrcQueue        string `json:"src-queue"`
	DestURI         string `json:"dest-uri"`
	DestQueue       string `json:"dest-queue,omitempty"`
	DestExchange    string `json:"dest-exchange,omitempty"`
	DestExchangeKey string `json:"dest-exchange-key,omitempty"`
	AckMode         string `json:"ack-mode"`
	SrcDeleteAfter  string `json:"src-delete-after,omitempty"`
}

// ShovelStatus represents the status of a shovel
type ShovelStatus struct {
	Name      string `json:"name"`
	VHost     string `json:"vhost"`
	Type      string `json:"type"`
	State     string `json:"state"`
	Timestamp string `json:"timestamp,omitempty"`
}

// NewManagementService creates a new RabbitMQ management service
func NewManagementService(config *config.AppConfig) *ManagementService {
	// Parse the RabbitMQ URL to extract credentials
	rabbitURL := config.RabbitMQ.URL
	parsedURL, err := url.Parse(rabbitURL)
	if err != nil {
		zap.L().Error("Failed to parse RabbitMQ URL", zap.Error(err))
		return nil
	}

	username := "guest"
	password := "guest"
	if parsedURL.User != nil {
		username = parsedURL.User.Username()
		if pwd, ok := parsedURL.User.Password(); ok {
			password = pwd
		}
	}

	// Build management API URL (typically on port 15672)
	host := parsedURL.Hostname()
	managementURL := fmt.Sprintf("http://%s:15672", host)

	// Use custom management URL if configured
	if config.RabbitMQ.ManagementURL != "" {
		managementURL = config.RabbitMQ.ManagementURL
	}

	vhost := "/"
	if config.RabbitMQ.VHost != "" {
		vhost = config.RabbitMQ.VHost
	}

	zap.L().Info("RabbitMQ Management Service initialized",
		zap.String("management_url", managementURL),
		zap.String("vhost", vhost))

	return &ManagementService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:        managementURL,
		username:       username,
		password:       password,
		vhost:          vhost,
		topologyConfig: &config.RabbitMQ.Topology,
	}
}

// doRequest performs an HTTP request to the RabbitMQ Management API
func (m *ManagementService) doRequest(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	reqURL := fmt.Sprintf("%s%s", m.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(m.username, m.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("RabbitMQ management API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ListQueues returns all queues in the vhost
func (m *ManagementService) ListQueues(ctx context.Context) ([]QueueInfo, error) {
	encodedVHost := url.PathEscape(m.vhost)
	path := fmt.Sprintf("/api/queues/%s", encodedVHost)

	body, err := m.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list queues: %w", err)
	}

	var queues []QueueInfo
	if err := json.Unmarshal(body, &queues); err != nil {
		return nil, fmt.Errorf("failed to parse queues response: %w", err)
	}

	return queues, nil
}

// GetQueue returns information about a specific queue
func (m *ManagementService) GetQueue(ctx context.Context, queueName string) (*QueueInfo, error) {
	encodedVHost := url.PathEscape(m.vhost)
	encodedQueue := url.PathEscape(queueName)
	path := fmt.Sprintf("/api/queues/%s/%s", encodedVHost, encodedQueue)

	body, err := m.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue %s: %w", queueName, err)
	}

	var queue QueueInfo
	if err := json.Unmarshal(body, &queue); err != nil {
		return nil, fmt.Errorf("failed to parse queue response: %w", err)
	}

	return &queue, nil
}

// ListExchanges returns all exchanges in the vhost
func (m *ManagementService) ListExchanges(ctx context.Context) ([]ExchangeInfo, error) {
	encodedVHost := url.PathEscape(m.vhost)
	path := fmt.Sprintf("/api/exchanges/%s", encodedVHost)

	body, err := m.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list exchanges: %w", err)
	}

	var exchanges []ExchangeInfo
	if err := json.Unmarshal(body, &exchanges); err != nil {
		return nil, fmt.Errorf("failed to parse exchanges response: %w", err)
	}

	return exchanges, nil
}

// GetExchange returns information about a specific exchange
func (m *ManagementService) GetExchange(ctx context.Context, exchangeName string) (*ExchangeInfo, error) {
	encodedVHost := url.PathEscape(m.vhost)
	encodedExchange := url.PathEscape(exchangeName)
	path := fmt.Sprintf("/api/exchanges/%s/%s", encodedVHost, encodedExchange)

	body, err := m.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange %s: %w", exchangeName, err)
	}

	var exchange ExchangeInfo
	if err := json.Unmarshal(body, &exchange); err != nil {
		return nil, fmt.Errorf("failed to parse exchange response: %w", err)
	}

	return &exchange, nil
}

// ListBindings returns all bindings for a queue
func (m *ManagementService) ListBindings(ctx context.Context, queueName string) ([]BindingInfo, error) {
	encodedVHost := url.PathEscape(m.vhost)
	encodedQueue := url.PathEscape(queueName)
	path := fmt.Sprintf("/api/queues/%s/%s/bindings", encodedVHost, encodedQueue)

	body, err := m.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list bindings for queue %s: %w", queueName, err)
	}

	var bindings []BindingInfo
	if err := json.Unmarshal(body, &bindings); err != nil {
		return nil, fmt.Errorf("failed to parse bindings response: %w", err)
	}

	return bindings, nil
}

// GetMessages retrieves messages from a queue without consuming them (peek)
func (m *ManagementService) GetMessages(ctx context.Context, queueName string, count int, ackMode string, encoding string) ([]Message, error) {
	if count <= 0 {
		count = 10
	}
	if ackMode == "" {
		ackMode = "ack_requeue_true" // Don't consume, just peek
	}
	if encoding == "" {
		encoding = "auto"
	}

	encodedVHost := url.PathEscape(m.vhost)
	encodedQueue := url.PathEscape(queueName)
	path := fmt.Sprintf("/api/queues/%s/%s/get", encodedVHost, encodedQueue)

	payload := map[string]interface{}{
		"count":    count,
		"ackmode":  ackMode,
		"encoding": encoding,
	}
	payloadBytes, _ := json.Marshal(payload)

	body, err := m.doRequest(ctx, http.MethodPost, path, io.NopCloser(bytesReader(payloadBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to get messages from queue %s: %w", queueName, err)
	}

	var messages []Message
	if err := json.Unmarshal(body, &messages); err != nil {
		return nil, fmt.Errorf("failed to parse messages response: %w", err)
	}

	return messages, nil
}

// bytesReader helper function
func bytesReader(b []byte) io.Reader {
	return &bytesReaderImpl{data: b}
}

type bytesReaderImpl struct {
	data []byte
	pos  int
}

func (r *bytesReaderImpl) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// PurgeQueue removes all messages from a queue
func (m *ManagementService) PurgeQueue(ctx context.Context, queueName string) error {
	encodedVHost := url.PathEscape(m.vhost)
	encodedQueue := url.PathEscape(queueName)
	path := fmt.Sprintf("/api/queues/%s/%s/contents", encodedVHost, encodedQueue)

	_, err := m.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to purge queue %s: %w", queueName, err)
	}

	zap.L().Info("Queue purged successfully", zap.String("queue", queueName))
	return nil
}

// MoveMessages moves messages from source queue to destination (shovel)
// This is used to retry messages from DLQ back to the main queue
func (m *ManagementService) MoveMessages(ctx context.Context, srcQueue, destQueue string, destExchange string, destRoutingKey string) error {
	// Create a dynamic shovel to move messages
	shovelName := fmt.Sprintf("temp-shovel-%s-to-%s-%d", srcQueue, destQueue, time.Now().Unix())

	shovelDef := ShovelDefinition{
		SrcURI:         fmt.Sprintf("amqp:///%s", url.PathEscape(m.vhost)),
		SrcQueue:       srcQueue,
		DestURI:        fmt.Sprintf("amqp:///%s", url.PathEscape(m.vhost)),
		AckMode:        "on-confirm",
		SrcDeleteAfter: "queue-length", // Delete shovel after all messages are moved
	}

	if destExchange != "" {
		shovelDef.DestExchange = destExchange
		shovelDef.DestExchangeKey = destRoutingKey
	} else {
		shovelDef.DestQueue = destQueue
	}

	// Create the shovel parameter
	encodedVHost := url.PathEscape(m.vhost)
	path := fmt.Sprintf("/api/parameters/shovel/%s/%s", encodedVHost, url.PathEscape(shovelName))

	payload := map[string]interface{}{
		"value": shovelDef,
	}
	payloadBytes, _ := json.Marshal(payload)

	_, err := m.doRequest(ctx, http.MethodPut, path, io.NopCloser(bytesReader(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to create shovel from %s to %s: %w", srcQueue, destQueue, err)
	}

	zap.L().Info("Shovel created successfully",
		zap.String("shovel_name", shovelName),
		zap.String("src_queue", srcQueue),
		zap.String("dest_queue", destQueue))

	return nil
}

// PublishMessage publishes a message to an exchange
func (m *ManagementService) PublishMessage(ctx context.Context, exchange, routingKey string, payload string, properties map[string]interface{}) error {
	encodedVHost := url.PathEscape(m.vhost)
	encodedExchange := url.PathEscape(exchange)
	path := fmt.Sprintf("/api/exchanges/%s/%s/publish", encodedVHost, encodedExchange)

	message := map[string]interface{}{
		"routing_key":      routingKey,
		"payload":          payload,
		"payload_encoding": "string",
		"properties":       properties,
	}
	payloadBytes, _ := json.Marshal(message)

	_, err := m.doRequest(ctx, http.MethodPost, path, io.NopCloser(bytesReader(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to publish message to exchange %s: %w", exchange, err)
	}

	return nil
}

// GetTopologyConfig returns the configured topology from config
func (m *ManagementService) GetTopologyConfig() *config.RabbitMQTopologyConfig {
	return m.topologyConfig
}

// ClassifyQueue classifies a queue based on its name pattern
func (m *ManagementService) ClassifyQueue(queueName string) string {
	// Check queue suffixes to determine type
	suffixes := []struct {
		suffix string
		qType  string
	}{
		{".dlq", "dead_letter"},
		{".retry", "retry"},
		{".delay", "delayed"},
	}

	for _, s := range suffixes {
		if len(queueName) > len(s.suffix) && queueName[len(queueName)-len(s.suffix):] == s.suffix {
			return s.qType
		}
	}

	return "main"
}

// GetQueueMainName returns the main queue name from a DLQ or retry queue name
func (m *ManagementService) GetQueueMainName(queueName string) string {
	suffixes := []string{".dlq", ".retry", ".delay"}
	for _, suffix := range suffixes {
		if len(queueName) > len(suffix) && queueName[len(queueName)-len(suffix):] == suffix {
			return queueName[:len(queueName)-len(suffix)]
		}
	}
	return queueName
}

// ListShovels returns all shovels in the vhost
func (m *ManagementService) ListShovels(ctx context.Context) ([]ShovelStatus, error) {
	encodedVHost := url.PathEscape(m.vhost)
	path := fmt.Sprintf("/api/shovels/%s", encodedVHost)

	body, err := m.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list shovels: %w", err)
	}

	var shovels []ShovelStatus
	if err := json.Unmarshal(body, &shovels); err != nil {
		return nil, fmt.Errorf("failed to parse shovels response: %w", err)
	}

	return shovels, nil
}

// DeleteShovel deletes a shovel by name
func (m *ManagementService) DeleteShovel(ctx context.Context, shovelName string) error {
	encodedVHost := url.PathEscape(m.vhost)
	path := fmt.Sprintf("/api/parameters/shovel/%s/%s", encodedVHost, url.PathEscape(shovelName))

	_, err := m.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete shovel %s: %w", shovelName, err)
	}

	zap.L().Info("Shovel deleted successfully", zap.String("shovel_name", shovelName))
	return nil
}

// GetOverview returns RabbitMQ overview information
func (m *ManagementService) GetOverview(ctx context.Context) (map[string]interface{}, error) {
	body, err := m.doRequest(ctx, http.MethodGet, "/api/overview", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get overview: %w", err)
	}

	var overview map[string]interface{}
	if err := json.Unmarshal(body, &overview); err != nil {
		return nil, fmt.Errorf("failed to parse overview response: %w", err)
	}

	return overview, nil
}
