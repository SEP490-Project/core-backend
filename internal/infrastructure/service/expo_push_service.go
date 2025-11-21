package service

import (
	"bytes"
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	iservice_third_party "core-backend/internal/application/interfaces/iservice_third_party"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// region: ========== Variables ==========

const (
	ExpoPushURL         = "https://exp.host/--/api/v2/push/send"
	ExpoReceiptURL      = "https://exp.host/--/api/v2/push/getReceipts"
	ExpoMaxBatchSize    = 100
	ExpoTokenPrefix     = "ExponentPushToken["
	ExpoMaxPayloadSize  = 4096 // bytes
	ExpoDefaultPriority = "default"
	ExpoHighPriority    = "high"
	ExpoNormalPriority  = "normal"
	ExpoDefaultSound    = "default"
)

var (
	// ErrInvalidExpoToken indicates the Expo push token is invalid
	ErrInvalidExpoToken = errors.New("invalid Expo push token format")
	// ErrExpoServiceUnavailable indicates Expo service is unreachable
	ErrExpoServiceUnavailable = errors.New("expo push notification service unavailable")
	// ErrExpoPushFailed indicates the push notification failed on Expo side
	ErrExpoPushFailed = errors.New("expo push notification failed")
)

// endregion

// ExpoPushService handles Expo push notifications
type ExpoPushService struct {
	httpClient  *http.Client
	rateLimiter *expoRateLimiter
}

// expoRateLimiter implements token bucket rate limiting for Expo Push API
type expoRateLimiter struct {
	tokens         int
	maxTokens      int
	refillRate     time.Duration
	lastRefillTime time.Time
	mu             sync.Mutex
}

// NewExpoPushService creates a new Expo push notification service
func NewExpoPushService(cfg *config.AppConfig) *ExpoPushService {
	// Expo recommends 600 requests per minute per IP
	pushPerMinute := 500 // Conservative limit
	if cfg != nil && cfg.Notification.RateLimits.PushPerMinute > 0 {
		pushPerMinute = cfg.Notification.RateLimits.PushPerMinute
	}

	refillRate := time.Minute / time.Duration(pushPerMinute)

	zap.L().Info("Expo Push Service initialized",
		zap.Int("rate_limit", pushPerMinute),
		zap.Duration("refill_rate", refillRate))

	return &ExpoPushService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: &expoRateLimiter{
			tokens:         pushPerMinute,
			maxTokens:      pushPerMinute,
			refillRate:     refillRate,
			lastRefillTime: time.Now(),
		},
	}
}

// IsExpoToken checks if a token is an Expo push token
func (s *ExpoPushService) IsExpoToken(token string) bool {
	return strings.HasPrefix(token, ExpoTokenPrefix)
}

// ValidateExpoToken validates an Expo push token format
func (s *ExpoPushService) ValidateExpoToken(token string) error {
	if !s.IsExpoToken(token) {
		return ErrInvalidExpoToken
	}
	if !strings.HasSuffix(token, "]") {
		return ErrInvalidExpoToken
	}
	return nil
}

// SendToToken sends a push notification to a single Expo token
func (s *ExpoPushService) SendToToken(ctx context.Context, token, title, body string, data map[string]string) error {
	if err := s.ValidateExpoToken(token); err != nil {
		return err
	}

	// Convert map[string]string to map[string]any
	dataInterface := make(map[string]any)
	for k, v := range data {
		dataInterface[k] = v
	}

	message := dtos.ExpoMessage{
		To:       token,
		Title:    title,
		Body:     body,
		Data:     dataInterface,
		Sound:    ExpoDefaultSound,
		Priority: ExpoDefaultPriority,
	}

	tickets, err := s.sendMessages(ctx, []dtos.ExpoMessage{message})
	if err != nil {
		return err
	}

	if len(tickets) == 0 {
		return errors.New("no response from Expo service")
	}

	ticket := tickets[0]
	if ticket.Status == "error" {
		// Check for specific error types
		if ticket.Details != nil {
			if errorType, ok := ticket.Details["error"].(string); ok {
				if errorType == "DeviceNotRegistered" {
					return ErrInvalidExpoToken
				}
			}
		}
		return fmt.Errorf("%w: %s", ErrExpoPushFailed, ticket.Message)
	}

	zap.L().Info("Expo push notification sent successfully",
		zap.String("ticket_id", ticket.ID),
		zap.String("token", token[:20]+"..."))

	return nil
}

// SendWithPriority sends a push notification to a single Expo token with priority
func (s *ExpoPushService) SendWithPriority(ctx context.Context, token, title, body string, data map[string]string, priority string) error {
	if err := s.ValidateExpoToken(token); err != nil {
		return err
	}

	// Validate priority
	if priority != ExpoDefaultPriority && priority != ExpoHighPriority && priority != ExpoNormalPriority {
		priority = ExpoDefaultPriority
	}

	// Convert map[string]string to map[string]any
	dataInterface := make(map[string]any)
	for k, v := range data {
		dataInterface[k] = v
	}

	message := dtos.ExpoMessage{
		To:       token,
		Title:    title,
		Body:     body,
		Data:     dataInterface,
		Sound:    ExpoDefaultSound,
		Priority: priority,
	}

	tickets, err := s.sendMessages(ctx, []dtos.ExpoMessage{message})
	if err != nil {
		return err
	}

	if len(tickets) == 0 {
		return errors.New("no response from Expo service")
	}

	ticket := tickets[0]
	if ticket.Status == "error" {
		// Check for specific error types
		if ticket.Details != nil {
			if errorType, ok := ticket.Details["error"].(string); ok {
				if errorType == "DeviceNotRegistered" {
					return ErrInvalidExpoToken
				}
			}
		}
		return fmt.Errorf("%w: %s", ErrExpoPushFailed, ticket.Message)
	}

	return nil
}

// SendMulticast sends push notifications to multiple tokens (batch)
func (s *ExpoPushService) SendMulticast(ctx context.Context, tokens []string, title, body string, data map[string]string) (successCount, failureCount int, invalidTokens []string, err error) {
	if len(tokens) == 0 {
		return 0, 0, nil, errors.New("no tokens provided")
	}

	// Validate all tokens first
	validTokens := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if err := s.ValidateExpoToken(token); err != nil {
			invalidTokens = append(invalidTokens, token)
			failureCount++
			continue
		}
		validTokens = append(validTokens, token)
	}

	if len(validTokens) == 0 {
		return 0, failureCount, invalidTokens, errors.New("no valid Expo tokens")
	}

	// Convert data
	dataInterface := make(map[string]any)
	for k, v := range data {
		dataInterface[k] = v
	}

	// Split into batches of 100 (Expo's limit)
	for i := 0; i < len(validTokens); i += ExpoMaxBatchSize {
		end := min(i+ExpoMaxBatchSize, len(validTokens))

		batchTokens := validTokens[i:end]
		messages := make([]dtos.ExpoMessage, len(batchTokens))
		for j, token := range batchTokens {
			messages[j] = dtos.ExpoMessage{
				To:       token,
				Title:    title,
				Body:     body,
				Data:     dataInterface,
				Sound:    ExpoDefaultSound,
				Priority: ExpoDefaultPriority,
			}
		}

		tickets, batchErr := s.sendMessages(ctx, messages)
		if batchErr != nil {
			zap.L().Error("Failed to send batch to Expo",
				zap.Error(batchErr),
				zap.Int("batch_size", len(batchTokens)))
			failureCount += len(batchTokens)
			continue
		}

		// Process tickets
		for idx, ticket := range tickets {
			if ticket.Status == "ok" {
				successCount++
			} else {
				failureCount++
				// Check if token is invalid
				if ticket.Details != nil {
					if errorType, ok := ticket.Details["error"].(string); ok {
						if errorType == "DeviceNotRegistered" && idx < len(batchTokens) {
							invalidTokens = append(invalidTokens, batchTokens[idx])
						}
					}
				}
				zap.L().Warn("Expo push failed for token",
					zap.String("error", ticket.Message),
					zap.String("token", batchTokens[idx][:20]+"..."))
			}
		}
	}

	return successCount, failureCount, invalidTokens, nil
}

// sendMessages sends messages to Expo Push API
func (s *ExpoPushService) sendMessages(ctx context.Context, messages []dtos.ExpoMessage) ([]dtos.ExpoTicket, error) {
	// Rate limiting
	if err := s.rateLimiter.waitForToken(ctx); err != nil {
		return nil, err
	}

	// Prepare request
	bodyBytes, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal messages: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ExpoPushURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrExpoServiceUnavailable, err.Error())
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		zap.L().Error("Expo API returned error status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(respBody)))
		return nil, fmt.Errorf("%w: HTTP %d", ErrExpoServiceUnavailable, resp.StatusCode)
	}

	// Parse response
	var expoResp dtos.ExpoResponse
	if err := json.Unmarshal(respBody, &expoResp); err != nil {
		return nil, fmt.Errorf("failed to parse Expo response: %w", err)
	}

	return expoResp.Data, nil
}

// GetReceipts fetches delivery receipts for sent notifications
func (s *ExpoPushService) GetReceipts(ctx context.Context, ticketIDs []string) (map[string]iservice_third_party.ExpoReceipt, error) {
	if len(ticketIDs) == 0 {
		return nil, errors.New("no ticket IDs provided")
	}

	reqBody := dtos.ExpoReceiptRequest{IDs: ticketIDs}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ExpoReceiptURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrExpoServiceUnavailable, err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", ErrExpoServiceUnavailable, resp.StatusCode)
	}

	var receiptResp dtos.ExpoReceiptResponse
	if err := json.Unmarshal(respBody, &receiptResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert map[string]ExpoReceipt to map[string]iservice_third_party.ExpoReceipt
	result := make(map[string]iservice_third_party.ExpoReceipt, len(receiptResp.Data))
	for k, v := range receiptResp.Data {
		result[k] = iservice_third_party.ExpoReceipt{
			Status:  v.Status,
			Message: v.Message,
			Details: v.Details,
		}
	}

	return result, nil
}

// Health checks the health of the Expo push service
func (s *ExpoPushService) Health(ctx context.Context) iservice_third_party.ServiceHealth {
	checkTime := time.Now()

	// Create a test request to check service availability
	req, err := http.NewRequestWithContext(ctx, "GET", "https://exp.host", nil)
	if err != nil {
		return iservice_third_party.ServiceHealth{
			Name:          "ExpoService",
			IsHealthy:     false,
			LastCheckTime: checkTime,
			LastError:     fmt.Errorf("failed to create health check request: %w", err),
		}
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return iservice_third_party.ServiceHealth{
			Name:          "ExpoService",
			IsHealthy:     false,
			LastCheckTime: checkTime,
			LastError:     fmt.Errorf("expo service unreachable: %w", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return iservice_third_party.ServiceHealth{
			Name:          "ExpoService",
			IsHealthy:     true,
			LastCheckTime: checkTime,
			Details: map[string]any{
				"status_code": resp.StatusCode,
				"message":     "Expo push service is operational",
			},
		}
	}

	return iservice_third_party.ServiceHealth{
		Name:          "ExpoService",
		IsHealthy:     false,
		LastCheckTime: checkTime,
		LastError:     fmt.Errorf("expo service returned status code: %d", resp.StatusCode),
	}
}

// Rate limiter implementation
func (rl *expoRateLimiter) waitForToken(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens based on time passed
	now := time.Now()
	elapsed := now.Sub(rl.lastRefillTime)
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefillTime = now
	}

	// Check if token available
	if rl.tokens > 0 {
		rl.tokens--
		return nil
	}

	// Wait for next token
	waitTime := rl.refillRate
	rl.mu.Unlock()

	select {
	case <-time.After(waitTime):
		rl.mu.Lock()
		if rl.tokens > 0 {
			rl.tokens--
			return nil
		}
		return errors.New("rate limit exceeded")
	case <-ctx.Done():
		rl.mu.Lock()
		return ctx.Err()
	}
}
