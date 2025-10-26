package middleware

import (
	"context"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

type ctxKey string

const (
	userIDKey ctxKey = "user_id"
)

type AuthMiddleware struct {
	jwtService iservice.JWTService
}

func NewAuthMiddleware(jwtService iservice.JWTService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
	}
}

// RequireAuth validates JWT token and sets user context
func (a *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, responses.ErrorResponse(
				"MISSING_TOKEN: Authorization header is required",
				http.StatusUnauthorized,
			))
			c.Abort()
			return
		}

		// Extract Bearer token
		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, responses.ErrorResponse(
				"INVALID_TOKEN_FORMAT: Authorization header must be Bearer token",
				http.StatusUnauthorized,
			))
			c.Abort()
			return
		}

		token := tokenParts[1]
		claims, err := a.jwtService.ValidateAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, responses.ErrorResponse(
				"INVALID_TOKEN: "+err.Error(),
				http.StatusUnauthorized,
			))
			c.Abort()
			return
		}

		// Set user context
		c.Set("user_id", claims.UserID)
		c.Set("subject", claims.Subject)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Set("claims", claims)

		// Add user ID to request context
		currentContext := c.Request.Context()
		newContext := context.WithValue(currentContext, userIDKey, claims.UserID)
		c.Request = c.Request.WithContext(newContext)

		c.Next()
	}
}

// RequireRole validates user has specific role
func (a *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, responses.ErrorResponse(
				"MISSING_TOKEN: Authorization header is required",
				http.StatusUnauthorized,
			))
			c.Abort()
			return
		}

		// Extract Bearer token
		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, responses.ErrorResponse(
				"INVALID_TOKEN_FORMAT: Authorization header must be Bearer token",
				http.StatusUnauthorized,
			))
			c.Abort()
			return
		}

		token := tokenParts[1]
		claims, err := a.jwtService.ValidateAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, responses.ErrorResponse(
				"INVALID_TOKEN: "+err.Error(),
				http.StatusUnauthorized,
			))
			c.Abort()
			return
		}

		// Set user context
		c.Set("user_id", claims.UserID)
		c.Set("subject", claims.Subject)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Set("claims", claims)

		// Add user ID to request context
		currentContext := c.Request.Context()
		newContext := context.WithValue(currentContext, userIDKey, claims.UserID)
		c.Request = c.Request.WithContext(newContext)

		userRole := claims.Roles

		if slices.Contains(roles, userRole) {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, responses.ErrorResponse(
			"INSUFFICIENT_PERMISSIONS: User does not have required permissions",
			http.StatusForbidden,
		))
		c.Abort()
	}
}

func (a *AuthMiddleware) RequireAuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// Extract Bearer token
		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Next()
			return
		}

		token := tokenParts[1]
		claims, err := a.jwtService.ValidateAccessToken(token)
		if err != nil {
			return
		}

		// Set user context
		c.Set("user_id", claims.UserID)
		c.Set("subject", claims.Subject)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Set("claims", claims)

		// Add user ID to request context
		currentContext := c.Request.Context()
		newContext := context.WithValue(currentContext, userIDKey, claims.UserID)
		c.Request = c.Request.WithContext(newContext)

		c.Next()
	}
}
