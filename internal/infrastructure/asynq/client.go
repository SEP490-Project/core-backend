// Package asynq provides a general-purpose Asynq client wrapper for scheduling and managing tasks.
// It uses the existing Valkey (Redis-compatible) instance for task queue storage.
package asynq

import (
	"context"
	"core-backend/config"
	"core-backend/pkg/logging"
	"fmt"
	"strconv"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// AsynqClient wraps the Asynq client and server for task scheduling and processing
type AsynqClient struct {
	client    *asynq.Client
	inspector *asynq.Inspector
	server    *asynq.Server
	mux       *asynq.ServeMux
	config    *config.AsynqConfig
	redisOpt  asynq.RedisClientOpt
}

// NewAsynqClient creates a new Asynq client using the existing Valkey/Redis configuration
func NewAsynqClient(cacheConfig *config.CacheConfig, asynqConfig *config.AsynqConfig) (*AsynqClient, error) {
	zap.L().Info("Initializing Asynq client",
		zap.String("host", cacheConfig.Host),
		zap.Int("port", cacheConfig.Port),
		zap.Int("db", asynqConfig.DB))

	redisOpt := asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%s", cacheConfig.Host, strconv.Itoa(cacheConfig.Port)),
		Password: cacheConfig.Password,
		DB:       asynqConfig.DB,
	}

	client := asynq.NewClient(redisOpt)
	inspector := asynq.NewInspector(redisOpt)

	// Create server with configuration
	serverConfig := asynq.Config{
		Concurrency: asynqConfig.Concurrency,
		Queues: map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		},
		RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
			// Exponential backoff: 1min, 2min, 4min, 8min, 16min...
			return time.Duration(1<<uint(n)) * time.Minute
		},
		Logger:   &logging.AsynqLogger{},
		LogLevel: asynq.InfoLevel,
	}

	server := asynq.NewServer(redisOpt, serverConfig)
	mux := asynq.NewServeMux()

	return &AsynqClient{
		client:    client,
		inspector: inspector,
		server:    server,
		mux:       mux,
		config:    asynqConfig,
		redisOpt:  redisOpt,
	}, nil
}

// RegisterHandler registers a task handler for a specific task type
func (ac *AsynqClient) RegisterHandler(taskType string, handler asynq.Handler) {
	ac.mux.Handle(taskType, handler)
	zap.L().Info("Registered Asynq task handler", zap.String("task_type", taskType))
}

// RegisterHandlerFunc registers a handler function for a specific task type
func (ac *AsynqClient) RegisterHandlerFunc(taskType string, handler func(context.Context, *asynq.Task) error) {
	ac.mux.HandleFunc(taskType, handler)
	zap.L().Info("Registered Asynq task handler function", zap.String("task_type", taskType))
}

// Start starts the Asynq server to process tasks
func (ac *AsynqClient) Start() error {
	zap.L().Info("Starting Asynq server", zap.Int("concurrency", ac.config.Concurrency))
	return ac.server.Start(ac.mux)
}

// Shutdown gracefully shuts down the Asynq server and client
func (ac *AsynqClient) Shutdown() {
	zap.L().Info("Shutting down Asynq client")
	ac.server.Shutdown()
	if err := ac.client.Close(); err != nil {
		zap.L().Error("Failed to close Asynq client", zap.Error(err))
	}
}

// GetTaskInfo retrieves information about a task
func (ac *AsynqClient) GetTaskInfo(queue, taskID string) (*asynq.TaskInfo, error) {
	return ac.inspector.GetTaskInfo(queue, taskID)
}

// ListScheduledTasks lists all scheduled tasks in a queue
func (ac *AsynqClient) ListScheduledTasks(queue string, opts ...asynq.ListOption) ([]*asynq.TaskInfo, error) {
	return ac.inspector.ListScheduledTasks(queue, opts...)
}

// ListPendingTasks lists all pending tasks in a queue
func (ac *AsynqClient) ListPendingTasks(queue string, opts ...asynq.ListOption) ([]*asynq.TaskInfo, error) {
	return ac.inspector.ListPendingTasks(queue, opts...)
}

// GetQueueInfo retrieves information about a queue
func (ac *AsynqClient) GetQueueInfo(queue string) (*asynq.QueueInfo, error) {
	return ac.inspector.GetQueueInfo(queue)
}

// GetInspector returns the inspector for advanced operations
func (ac *AsynqClient) GetInspector() *asynq.Inspector {
	return ac.inspector
}

// HealthCheck performs a health check on the Asynq connection
func (ac *AsynqClient) HealthCheck() error {
	queues, err := ac.inspector.Queues()
	if err != nil {
		return fmt.Errorf("asynq health check failed: %w", err)
	}
	zap.L().Debug("Asynq health check passed", zap.Strings("queues", queues))
	return nil
}
