package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"core-backend/config"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// Task types
const (
	TaskEmailNotification = "email:notification"
	TaskFileCleanup       = "file:cleanup"
	TaskDataExport        = "data:export"
	TaskReportGeneration  = "report:generation"
)

type AsynqClient struct {
	client *asynq.Client
}

type AsynqServer struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

// Task payload structures
type EmailNotificationPayload struct {
	To      string            `json:"to"`
	Subject string            `json:"subject"`
	Body    string            `json:"body"`
	Data    map[string]string `json:"data,omitempty"`
}

type FileCleanupPayload struct {
	FilePaths []string `json:"file_paths"`
}

type DataExportPayload struct {
	UserID     string `json:"user_id"`
	ExportType string `json:"export_type"`
	Format     string `json:"format"`
}

type ReportGenerationPayload struct {
	ReportID   string                 `json:"report_id"`
	UserID     string                 `json:"user_id"`
	ReportType string                 `json:"report_type"`
	Parameters map[string]interface{} `json:"parameters"`
}

func NewAsynqClient() *AsynqClient {
	cfg := config.GetAppConfig().Asynq
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		DB:       cfg.RedisDB,
		Password: cfg.RedisPassword,
	})

	zap.L().Info("Asynq client created successfully",
		zap.String("redis_addr", cfg.RedisAddr),
		zap.Int("redis_db", cfg.RedisDB))

	return &AsynqClient{client: client}
}

func NewAsynqServer() *AsynqServer {
	cfg := config.GetAppConfig().Asynq
	server := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.RedisAddr,
			DB:       cfg.RedisDB,
			Password: cfg.RedisPassword,
		},
		asynq.Config{
			Concurrency: cfg.Concurrency,
			Queues:      cfg.Queues,
			Logger:      NewAsynqLogger(),
		},
	)

	mux := asynq.NewServeMux()
	asynqServer := &AsynqServer{
		server: server,
		mux:    mux,
	}

	// Register task handlers
	asynqServer.registerHandlers()

	zap.L().Info("Asynq server created successfully",
		zap.String("redis_addr", cfg.RedisAddr),
		zap.Int("redis_db", cfg.RedisDB),
		zap.Int("concurrency", cfg.Concurrency))

	return asynqServer
}

// EnqueueEmailNotification enqueues an email notification task
func (c *AsynqClient) EnqueueEmailNotification(payload EmailNotificationPayload, delay time.Duration) (*asynq.TaskInfo, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TaskEmailNotification, payloadBytes)
	opts := []asynq.Option{
		asynq.Queue("default"),
		asynq.MaxRetry(3),
	}

	if delay > 0 {
		opts = append(opts, asynq.ProcessIn(delay))
	}

	return c.client.Enqueue(task, opts...)
}

// EnqueueFileCleanup enqueues a file cleanup task
func (c *AsynqClient) EnqueueFileCleanup(payload FileCleanupPayload, delay time.Duration) (*asynq.TaskInfo, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TaskFileCleanup, payloadBytes)
	opts := []asynq.Option{
		asynq.Queue("default"),
		asynq.MaxRetry(2),
	}

	if delay > 0 {
		opts = append(opts, asynq.ProcessIn(delay))
	}

	return c.client.Enqueue(task, opts...)
}

// EnqueueDataExport enqueues a data export task
func (c *AsynqClient) EnqueueDataExport(payload DataExportPayload) (*asynq.TaskInfo, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TaskDataExport, payloadBytes)
	opts := []asynq.Option{
		asynq.Queue("critical"),
		asynq.MaxRetry(1),
		asynq.Timeout(30 * time.Minute),
	}

	return c.client.Enqueue(task, opts...)
}

// EnqueueReportGeneration enqueues a report generation task
func (c *AsynqClient) EnqueueReportGeneration(payload ReportGenerationPayload) (*asynq.TaskInfo, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TaskReportGeneration, payloadBytes)
	opts := []asynq.Option{
		asynq.Queue("critical"),
		asynq.MaxRetry(2),
		asynq.Timeout(45 * time.Minute),
	}

	return c.client.Enqueue(task, opts...)
}

// registerHandlers registers all task handlers
func (s *AsynqServer) registerHandlers() {
	s.mux.HandleFunc(TaskEmailNotification, s.handleEmailNotification)
	s.mux.HandleFunc(TaskFileCleanup, s.handleFileCleanup)
	s.mux.HandleFunc(TaskDataExport, s.handleDataExport)
	s.mux.HandleFunc(TaskReportGeneration, s.handleReportGeneration)
}

// Task handlers
func (s *AsynqServer) handleEmailNotification(ctx context.Context, t *asynq.Task) error {
	var payload EmailNotificationPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	zap.L().Info("Processing email notification task",
		zap.String("to", payload.To),
		zap.String("subject", payload.Subject))

	// TODO: Implement actual email sending logic
	// For now, just log the task
	time.Sleep(100 * time.Millisecond) // Simulate work

	zap.L().Info("Email notification sent successfully", zap.String("to", payload.To))
	return nil
}

func (s *AsynqServer) handleFileCleanup(ctx context.Context, t *asynq.Task) error {
	var payload FileCleanupPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	zap.L().Info("Processing file cleanup task",
		zap.Int("file_count", len(payload.FilePaths)))

	// TODO: Implement actual file cleanup logic
	// For now, just log the task
	for _, filePath := range payload.FilePaths {
		zap.L().Debug("Cleaning up file", zap.String("path", filePath))
		// os.Remove(filePath) // Actual cleanup would go here
	}

	zap.L().Info("File cleanup completed successfully")
	return nil
}

func (s *AsynqServer) handleDataExport(ctx context.Context, t *asynq.Task) error {
	var payload DataExportPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	zap.L().Info("Processing data export task",
		zap.String("user_id", payload.UserID),
		zap.String("export_type", payload.ExportType),
		zap.String("format", payload.Format))

	// TODO: Implement actual data export logic
	// This is a long-running task simulation
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			time.Sleep(1 * time.Second) // Simulate work
			zap.L().Debug("Export progress", zap.Int("step", i+1))
		}
	}

	zap.L().Info("Data export completed successfully", zap.String("user_id", payload.UserID))
	return nil
}

func (s *AsynqServer) handleReportGeneration(ctx context.Context, t *asynq.Task) error {
	var payload ReportGenerationPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	zap.L().Info("Processing report generation task",
		zap.String("report_id", payload.ReportID),
		zap.String("user_id", payload.UserID),
		zap.String("report_type", payload.ReportType))

	// TODO: Implement actual report generation logic
	// This is a long-running task simulation
	for i := 0; i < 15; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			time.Sleep(1 * time.Second) // Simulate work
			zap.L().Debug("Report generation progress", zap.Int("step", i+1))
		}
	}

	zap.L().Info("Report generation completed successfully", zap.String("report_id", payload.ReportID))
	return nil
}

// Start starts the Asynq server
func (s *AsynqServer) Start() error {
	zap.L().Info("Starting Asynq server...")
	return s.server.Run(s.mux)
}

// Stop stops the Asynq server gracefully
func (s *AsynqServer) Stop() {
	zap.L().Info("Stopping Asynq server...")
	s.server.Shutdown()
}

// Close closes the Asynq client
func (c *AsynqClient) Close() error {
	return c.client.Close()
}

// Custom logger for Asynq
type AsynqLogger struct{}

func NewAsynqLogger() *AsynqLogger {
	return &AsynqLogger{}
}

func (l *AsynqLogger) Debug(args ...interface{}) {
	zap.L().Debug(fmt.Sprint(args...))
}

func (l *AsynqLogger) Info(args ...interface{}) {
	zap.L().Info(fmt.Sprint(args...))
}

func (l *AsynqLogger) Warn(args ...interface{}) {
	zap.L().Warn(fmt.Sprint(args...))
}

func (l *AsynqLogger) Error(args ...interface{}) {
	zap.L().Error(fmt.Sprint(args...))
}

func (l *AsynqLogger) Fatal(args ...interface{}) {
	zap.L().Fatal(fmt.Sprint(args...))
}