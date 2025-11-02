package middleware

import (
	"core-backend/internal/application/dto/responses"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CSRFMiddleware provides CSRF protection for state-changing operations (T111)
// Validates Origin/Referer headers to prevent cross-site request forgery
type CSRFMiddleware struct {
	trustedOrigins []string
	strictMode     bool
}

// NewCSRFMiddleware creates a new CSRF middleware instance
func NewCSRFMiddleware(trustedOrigins []string, strictMode bool) *CSRFMiddleware {
	if len(trustedOrigins) == 0 {
		// Default trusted origins (should come from config)
		trustedOrigins = []string{
			"http://localhost:8080",
			"http://localhost:3000",
			"https://your-domain.com", // Replace with actual production domain
		}
	}

	return &CSRFMiddleware{
		trustedOrigins: trustedOrigins,
		strictMode:     strictMode,
	}
}

// Protect validates CSRF tokens for state-changing HTTP methods
func (m *CSRFMiddleware) Protect() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only validate state-changing methods (POST, PUT, PATCH, DELETE)
		if m.isReadOnlyMethod(c.Request.Method) {
			c.Next()
			return
		}

		// Check Origin header (preferred)
		origin := c.GetHeader("Origin")
		if origin != "" {
			if !m.isOriginTrusted(origin) {
				zap.L().Warn("CSRF: Untrusted Origin header",
					zap.String("origin", origin),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method))

				response := responses.ErrorResponse("CSRF validation failed: untrusted origin", http.StatusForbidden)
				c.AbortWithStatusJSON(http.StatusForbidden, response)
				return
			}
			c.Next()
			return
		}

		// Fallback to Referer header (less reliable)
		referer := c.GetHeader("Referer")
		if referer != "" {
			if !m.isRefererTrusted(referer) {
				zap.L().Warn("CSRF: Untrusted Referer header",
					zap.String("referer", referer),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method))

				response := responses.ErrorResponse("CSRF validation failed: untrusted referer", http.StatusForbidden)
				c.AbortWithStatusJSON(http.StatusForbidden, response)
				return
			}
			c.Next()
			return
		}

		// No Origin or Referer header
		if m.strictMode {
			// In strict mode, reject requests without Origin/Referer
			zap.L().Warn("CSRF: Missing Origin and Referer headers",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("user_agent", c.Request.UserAgent()))

			response := responses.ErrorResponse("CSRF validation failed: missing origin/referer", http.StatusForbidden)
			c.AbortWithStatusJSON(http.StatusForbidden, response)
			return
		}

		// In non-strict mode, allow (for backwards compatibility with API clients)
		zap.L().Debug("CSRF: No Origin/Referer header, allowing in non-strict mode",
			zap.String("path", c.Request.URL.Path))
		c.Next()
	}
}

// isReadOnlyMethod checks if HTTP method is read-only
func (m *CSRFMiddleware) isReadOnlyMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// isOriginTrusted validates the Origin header against trusted origins
func (m *CSRFMiddleware) isOriginTrusted(origin string) bool {
	origin = strings.ToLower(strings.TrimSpace(origin))

	for _, trusted := range m.trustedOrigins {
		if strings.ToLower(trusted) == origin {
			return true
		}

		// Support wildcard subdomains (e.g., "https://*.your-domain.com")
		if strings.Contains(trusted, "*") {
			pattern := strings.ReplaceAll(trusted, "*", ".*")
			if matched := strings.Contains(origin, strings.Replace(pattern, ".*", "", 1)); matched {
				return true
			}
		}
	}

	return false
}

// isRefererTrusted validates the Referer header
func (m *CSRFMiddleware) isRefererTrusted(referer string) bool {
	referer = strings.ToLower(strings.TrimSpace(referer))

	for _, trusted := range m.trustedOrigins {
		// Referer includes full URL, so check if it starts with trusted origin
		if strings.HasPrefix(referer, strings.ToLower(trusted)) {
			return true
		}
	}

	return false
}
