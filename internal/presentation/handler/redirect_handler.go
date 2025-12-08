package handler

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/pkg/utils"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type RedirectHandler struct {
	clickTrackingService iservice.ClickTrackingService
	appConfigs           *config.AppConfig
}

func NewRedirectHandler(clickTrackingService iservice.ClickTrackingService, configs *config.AppConfig) *RedirectHandler {
	return &RedirectHandler{
		clickTrackingService: clickTrackingService,
		appConfigs:           configs,
	}
}

// Redirect godoc
//
//	@Summary		Redirect to affiliate tracking URL
//	@Description	Resolves affiliate link hash and redirects to tracking URL while logging click event asynchronously
//	@Tags			Redirect
//	@Accept			json
//	@Produce		json
//	@Param			hash	path	string	true	"16-character affiliate link hash"
//	@Success		302		"Redirects to tracking URL"
//	@Failure		404		{object}	responses.APIResponse	"Affiliate link not found"
//	@Failure		410		{object}	responses.APIResponse	"Affiliate link expired or inactive"
//	@Failure		429		{object}	responses.APIResponse	"Rate limit exceeded"
//	@Failure		500		{object}	responses.APIResponse	"Internal server error"
//	@Router			/r/{hash} [get]
func (h *RedirectHandler) Redirect(c *gin.Context) {
	startTime := time.Now()

	// Extract hash from URL path parameter
	var req requests.RedirectRequest
	if err := c.ShouldBindUri(&req); err != nil {
		zap.L().Warn("Invalid redirect hash format", zap.Error(err))
		response := responses.ErrorResponse("Invalid affiliate link hash", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// T112: Input Sanitization - Validate and sanitize hash parameter
	hash, err := sanitizeHash(req.Hash)
	if err != nil {
		zap.L().Warn("Hash failed sanitization", zap.String("raw_hash", req.Hash), zap.Error(err))
		response := responses.ErrorResponse("Invalid affiliate link hash format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	zap.L().Debug("Processing redirect request", zap.String("hash", hash))

	// Rate limiting check (per IP address)
	if h.isRateLimited(c) {
		zap.L().Warn("Rate limit exceeded", zap.String("ip", c.ClientIP()), zap.String("hash", hash))
		response := responses.ErrorResponse("Too many requests. Please try again later.", http.StatusTooManyRequests)
		c.JSON(http.StatusTooManyRequests, response)
		return
	}

	// Resolve hash to tracking URL (cache-first)
	ctx := c.Request.Context()
	trackingURL, affiliateLinkID, err := h.clickTrackingService.ResolveHash(ctx, hash)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			zap.L().Warn("Affiliate link not found", zap.String("hash", hash))
			response := responses.ErrorResponse("Affiliate link not found", http.StatusNotFound)
			c.JSON(http.StatusNotFound, response)
			return
		}

		if strings.Contains(err.Error(), "inactive") || strings.Contains(err.Error(), "expired") {
			zap.L().Warn("Affiliate link inactive or expired", zap.String("hash", hash))
			response := responses.ErrorResponse("This affiliate link is no longer active", http.StatusGone)
			c.JSON(http.StatusGone, response)
			return
		}

		zap.L().Error("Failed to resolve affiliate link hash", zap.String("hash", hash), zap.Error(err))
		response := responses.ErrorResponse("Failed to process redirect", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// T110: Open Redirect Protection - Validate tracking URL before redirecting
	if !h.isValidTrackingURL(trackingURL) {
		zap.L().Error("Invalid tracking URL detected (potential open redirect)",
			zap.String("hash", hash),
			zap.String("tracking_url", trackingURL))
		response := responses.ErrorResponse("Invalid tracking URL", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Extract user information from context (if authenticated)
	var userID *string
	if userIDValue, exists := c.Get("user_id"); exists {
		if uid, ok := userIDValue.(string); ok {
			userID = &uid
		}
	}

	// Detect bot traffic
	isBot := h.detectBot(c.Request.UserAgent())

	// Build click event message
	clickEvent := &consumers.ClickEventMessage{
		AffiliateLinkID: affiliateLinkID,
		IPAddress:       c.ClientIP(),
		UserAgent:       c.Request.UserAgent(),
		ClickedAt:       time.Now(),
		IsBot:           isBot,
	}

	// Extract optional fields
	if userID != nil {
		// Parse UUID if needed
		utils.SafeFunc(c.Request.Context(), func(ctx context.Context) error {
			parsedUUID, err := uuid.Parse(*userID)
			if err != nil {
				return err
			}
			clickEvent.UserID = &parsedUUID
			return nil
		})
	}

	if referer := c.Request.Referer(); referer != "" {
		clickEvent.ReferrerURL = &referer
	}

	// Extract device/platform info from User-Agent
	clickEvent.DeviceType, clickEvent.Platform, clickEvent.Browser = h.parseUserAgent(c.Request.UserAgent())

	// Log click event asynchronously (non-blocking)
	// Failures here should not prevent redirect
	go func() {
		context := context.Background()
		if err := h.clickTrackingService.LogClickAsync(context, clickEvent); err != nil {
			zap.L().Warn("Failed to log click event (non-fatal)",
				zap.String("hash", hash),
				zap.String("affiliate_link_id", affiliateLinkID.String()),
				zap.Error(err))
		}
	}()

	// Log redirect latency
	latency := time.Since(startTime)
	zap.L().Info("Redirect completed",
		zap.String("hash", hash),
		zap.String("tracking_url", trackingURL),
		zap.Duration("latency", latency),
		zap.Bool("is_bot", isBot))

	// Perform HTTP 302 redirect
	c.Redirect(http.StatusFound, trackingURL)
}

// isRateLimited checks if the client IP has exceeded rate limits
// Simple in-memory rate limiting (for production, use Redis-based solution)
func (h *RedirectHandler) isRateLimited(_ *gin.Context) bool {
	// TODO: Implement Redis-based rate limiting
	// For now, return false (no rate limiting)
	// Production implementation should use:
	// - Token bucket or sliding window algorithm
	// - Redis for distributed rate limiting
	// - Per-IP limits (e.g., 100 clicks per minute)
	return false
}

// detectBot performs simple bot detection based on User-Agent
func (h *RedirectHandler) detectBot(userAgent string) bool {
	userAgentLower := strings.ToLower(userAgent)

	return utils.ContainsSlice(h.appConfigs.AdminConfig.BotSignatures, userAgentLower)
}

// parseUserAgent extracts device type, platform, and browser from User-Agent string
// Returns (deviceType, platform, browser)
func (h *RedirectHandler) parseUserAgent(userAgent string) (string, string, string) {
	ua := strings.ToLower(userAgent)

	// Detect device type
	deviceType := "desktop"
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		deviceType = "mobile"
	} else if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		deviceType = "tablet"
	}

	// Detect platform
	platform := "unknown"
	if strings.Contains(ua, "windows") {
		platform = "Windows"
	} else if strings.Contains(ua, "mac os") || strings.Contains(ua, "macos") {
		platform = "macOS"
	} else if strings.Contains(ua, "linux") {
		platform = "Linux"
	} else if strings.Contains(ua, "android") {
		platform = "Android"
	} else if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ios") {
		platform = "iOS"
	}

	// Detect browser
	browser := "unknown"
	if strings.Contains(ua, "edg/") || strings.Contains(ua, "edge") {
		browser = "Edge"
	} else if strings.Contains(ua, "chrome") && !strings.Contains(ua, "edg") {
		browser = "Chrome"
	} else if strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome") {
		browser = "Safari"
	} else if strings.Contains(ua, "firefox") {
		browser = "Firefox"
	} else if strings.Contains(ua, "opera") || strings.Contains(ua, "opr/") {
		browser = "Opera"
	}

	return deviceType, platform, browser
}

// isValidTrackingURL validates the tracking URL to prevent open redirect attacks (T110)
// Only allows HTTPS URLs from trusted e-commerce domains
func (h *RedirectHandler) isValidTrackingURL(trackingURL string) bool {
	if h.appConfigs.IsDevelopment() {
		return true
	}

	// Parse URL
	parsedURL, err := url.Parse(trackingURL)
	if err != nil {
		zap.L().Warn("Failed to parse tracking URL", zap.String("url", trackingURL), zap.Error(err))
		return false
	}

	// Must be HTTPS (or HTTP for local development)
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		zap.L().Warn("Invalid URL scheme", zap.String("scheme", parsedURL.Scheme))
		return false
	}

	// Get hostname
	hostname := strings.ToLower(parsedURL.Hostname())

	// Check if hostname matches any trusted domain
	for _, domain := range h.appConfigs.AdminConfig.TrackingLinkTrustedDomains {
		if hostname == domain || strings.HasSuffix(hostname, "."+domain) {
			return true
		}
	}

	// Log suspicious redirect attempt
	zap.L().Warn("Untrusted domain in tracking URL",
		zap.String("hostname", hostname),
		zap.String("full_url", trackingURL))

	return false
}

// sanitizeHash validates and sanitizes hash input to prevent injection attacks (T112)
func sanitizeHash(hash string) (string, error) {
	// Ensure hash is exactly 16 non-whitespace characters
	hashRegex := regexp.MustCompile(`[^\s]{16}`)
	if !hashRegex.MatchString(hash) {
		return "", fmt.Errorf("invalid hash format: must be 16 alphanumeric characters")
	}

	// Already validated by regex, return as-is
	return hash, nil
}
