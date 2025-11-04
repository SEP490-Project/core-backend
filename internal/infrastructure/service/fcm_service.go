package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"errors"
	"sync"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

var (
	// ErrInvalidToken indicates the FCM token is invalid or unregistered
	ErrInvalidToken = errors.New("invalid or unregistered FCM token")
	// ErrFCMNotInitialized indicates FCM service was not properly initialized
	ErrFCMNotInitialized = errors.New("FCM service not initialized")
)

// fcmService handles Firebase Cloud Messaging operations with rate limiting
type fcmService struct {
	client      *messaging.Client
	rateLimiter *fcmRateLimiter
}

// fcmRateLimiter implements token bucket rate limiting for FCM
type fcmRateLimiter struct {
	tokens         int
	maxTokens      int
	refillRate     time.Duration
	lastRefillTime time.Time
	mu             sync.Mutex
}

// NewFCMService creates a new FCM service instance with rate limiting
// serviceAccountPath: path to Firebase service account JSON file
func NewFCMService(serviceAccountPath string, cfg *config.AppConfig) (iservice_third_party.FCMService, error) {
	if serviceAccountPath == "" {
		zap.L().Warn("Firebase service account path not configured, FCM service will not be initialized")
		return &fcmService{client: nil}, nil
	}

	opt := option.WithCredentialsFile(serviceAccountPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		zap.L().Error("Failed to initialize Firebase app", zap.Error(err))
		return nil, err
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		zap.L().Error("Failed to get Firebase messaging client", zap.Error(err))
		return nil, err
	}

	// Calculate refill rate based on rate limit (pushes per minute)
	pushPerMinute := cfg.Notification.RateLimits.PushPerMinute
	if pushPerMinute <= 0 {
		pushPerMinute = 500 // Default to 500 per minute
	}
	refillRate := time.Minute / time.Duration(pushPerMinute)

	zap.L().Info("FCM service initialized successfully",
		zap.Int("rate_limit", pushPerMinute),
		zap.Duration("refill_rate", refillRate))

	return &fcmService{
		client: client,
		rateLimiter: &fcmRateLimiter{
			tokens:         pushPerMinute,
			maxTokens:      pushPerMinute,
			refillRate:     refillRate,
			lastRefillTime: time.Now(),
		},
	}, nil
}

// SendToToken sends a push notification to a single device token
func (s *fcmService) SendToToken(ctx context.Context, token, title, body string, data map[string]string) error {
	if s.client == nil {
		return ErrFCMNotInitialized
	}

	// Rate limiting
	if err := s.rateLimiter.waitForToken(ctx); err != nil {
		zap.L().Warn("FCM rate limit exceeded or context cancelled",
			zap.Error(err))
		return err
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:  data,
		Token: token,
	}

	messageID, err := s.client.Send(ctx, message)
	if err != nil {
		// Check for invalid token errors
		if messaging.IsUnregistered(err) || messaging.IsInvalidArgument(err) {
			zap.L().Warn("Invalid FCM token detected",
				zap.String("token", token),
				zap.Error(err))
			return ErrInvalidToken
		}
		zap.L().Error("Failed to send FCM message",
			zap.String("token", token),
			zap.Error(err))
		return err
	}

	zap.L().Info("FCM message sent successfully",
		zap.String("message_id", messageID),
		zap.String("token", token))
	return nil
}

// SendMulticast sends a push notification to multiple device tokens
func (s *fcmService) SendMulticast(ctx context.Context, tokens []string, title, body string, data map[string]string) (*messaging.BatchResponse, error) {
	if s.client == nil {
		return nil, ErrFCMNotInitialized
	}

	if len(tokens) == 0 {
		return &messaging.BatchResponse{}, nil
	}

	// Rate limiting - wait for one token per batch (batch operations are counted as single API call)
	if err := s.rateLimiter.waitForToken(ctx); err != nil {
		zap.L().Warn("FCM rate limit exceeded for multicast",
			zap.Error(err),
			zap.Int("token_count", len(tokens)))
		return nil, err
	}

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:   data,
		Tokens: tokens,
	}

	batchResponse, err := s.client.SendEachForMulticast(ctx, message)
	if err != nil {
		zap.L().Error("Failed to send FCM multicast",
			zap.Int("token_count", len(tokens)),
			zap.Error(err))
		return nil, err
	}

	zap.L().Info("FCM multicast sent",
		zap.Int("success_count", batchResponse.SuccessCount),
		zap.Int("failure_count", batchResponse.FailureCount),
		zap.Int("total_tokens", len(tokens)))

	return batchResponse, nil
}

// SendWithPlatformConfig sends a push notification with platform-specific configuration
func (s *fcmService) SendWithPlatformConfig(ctx context.Context, token, title, body string, data map[string]string, apnsConfig *messaging.APNSConfig, androidConfig *messaging.AndroidConfig) error {
	if s.client == nil {
		return ErrFCMNotInitialized
	}

	// Rate limiting
	if err := s.rateLimiter.waitForToken(ctx); err != nil {
		zap.L().Warn("FCM rate limit exceeded for platform config message",
			zap.Error(err))
		return err
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:    data,
		Token:   token,
		APNS:    apnsConfig,
		Android: androidConfig,
	}

	messageID, err := s.client.Send(ctx, message)
	if err != nil {
		// Check for invalid token errors
		if messaging.IsUnregistered(err) || messaging.IsInvalidArgument(err) {
			zap.L().Warn("Invalid FCM token detected with platform config",
				zap.String("token", token),
				zap.Error(err))
			return ErrInvalidToken
		}
		zap.L().Error("Failed to send FCM message with platform config",
			zap.String("token", token),
			zap.Error(err))
		return err
	}

	zap.L().Info("FCM message with platform config sent successfully",
		zap.String("message_id", messageID),
		zap.String("token", token))
	return nil
}

// SendMulticastWithPlatformConfig sends a push notification to multiple tokens with platform-specific configuration
func (s *fcmService) SendMulticastWithPlatformConfig(ctx context.Context, tokens []string, title, body string, data map[string]string, apnsConfig *messaging.APNSConfig, androidConfig *messaging.AndroidConfig) (*messaging.BatchResponse, error) {
	if s.client == nil {
		return nil, ErrFCMNotInitialized
	}

	if len(tokens) == 0 {
		return &messaging.BatchResponse{}, nil
	}

	// Rate limiting - wait for one token per batch
	if err := s.rateLimiter.waitForToken(ctx); err != nil {
		zap.L().Warn("FCM rate limit exceeded for multicast with platform config",
			zap.Error(err),
			zap.Int("token_count", len(tokens)))
		return nil, err
	}

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:    data,
		Tokens:  tokens,
		APNS:    apnsConfig,
		Android: androidConfig,
	}

	batchResponse, err := s.client.SendEachForMulticast(ctx, message)
	if err != nil {
		zap.L().Error("Failed to send FCM multicast with platform config",
			zap.Int("token_count", len(tokens)),
			zap.Error(err))
		return nil, err
	}

	zap.L().Info("FCM multicast with platform config sent",
		zap.Int("success_count", batchResponse.SuccessCount),
		zap.Int("failure_count", batchResponse.FailureCount),
		zap.Int("total_tokens", len(tokens)))

	return batchResponse, nil
}

// waitForToken blocks until a token is available in the rate limiter
func (rl *fcmRateLimiter) waitForToken(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Calculate tokens to add based on elapsed time
	elapsed := time.Since(rl.lastRefillTime)
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		rl.tokens = min(rl.tokens+tokensToAdd, rl.maxTokens)
		rl.lastRefillTime = rl.lastRefillTime.Add(time.Duration(tokensToAdd) * rl.refillRate)
	}

	// If tokens available, consume one
	if rl.tokens > 0 {
		rl.tokens--
		return nil
	}

	// Need to wait for next token
	waitTime := rl.refillRate
	rl.mu.Unlock() // Release lock while waiting

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		// Re-acquire lock and try again
		rl.mu.Lock()
		if rl.tokens > 0 {
			rl.tokens--
			return nil
		}
		return errors.New("rate limit exceeded")
	}
}

// Health returns the health status of the FCM service
func (s *fcmService) Health(ctx context.Context) iservice_third_party.ServiceHealth {
	health := iservice_third_party.ServiceHealth{
		Name:          "FCMService",
		LastCheckTime: time.Now(),
		Details:       make(map[string]any),
	}

	if s.client == nil {
		health.IsHealthy = false
		health.LastError = ErrFCMNotInitialized
		zap.L().Debug("FCM service health check failed - client not initialized")
	} else {
		health.IsHealthy = true
		zap.L().Debug("FCM service health check passed")
	}

	return health
}
