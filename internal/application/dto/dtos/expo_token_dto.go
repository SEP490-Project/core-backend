package dtos

import (
	"sync"
	"time"
)

// ExpoMessage represents a push notification message for Expo
type ExpoMessage struct {
	To        any            `json:"to"` // string or []string
	Title     string         `json:"title,omitempty"`
	Body      string         `json:"body,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Sound     string         `json:"sound,omitempty"`    // "default" or null
	TTL       int            `json:"ttl,omitempty"`      // seconds
	Priority  string         `json:"priority,omitempty"` // "default", "normal", "high"
	Badge     *int           `json:"badge,omitempty"`
	ChannelID string         `json:"channelId,omitempty"` // Android notification channel
}

// ExpoResponse represents the response from Expo push API
type ExpoResponse struct {
	Data []ExpoTicket `json:"data"`
}

// ExpoTicket represents a single push notification ticket
type ExpoTicket struct {
	Status  string         `json:"status"` // "ok" or "error"
	ID      string         `json:"id,omitempty"`
	Message string         `json:"message,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

// ExpoReceiptRequest represents a request for push receipts
type ExpoReceiptRequest struct {
	IDs []string `json:"ids"`
}

// ExpoReceiptResponse represents receipts for push notifications
type ExpoReceiptResponse struct {
	Data map[string]ExpoReceipt `json:"data"`
}

// ExpoReceipt represents the delivery status of a push notification
type ExpoReceipt struct {
	Status  string         `json:"status"` // "ok" or "error"
	Message string         `json:"message,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

// ExpoRateLimiter implements token bucket rate limiting for Expo Push API
type ExpoRateLimiter struct {
	tokens         int
	maxTokens      int
	refillRate     time.Duration
	lastRefillTime time.Time
	mu             sync.Mutex
}
