package middleware

import (
	"net/http"
	"slices"
	"strings"
	"core-backend/internal/application/service"
	"core-backend/internal/presentation/dto/response"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	jwtService *service.JWTService
}

func NewAuthMiddleware(jwtService *service.JWTService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
	}
}

// RequireAuth validates JWT token and sets user context
func (a *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, response.APIResponse{
				Success: false,
				Error: &response.ErrorInfo{
					Code:    "MISSING_TOKEN",
					Message: "Authorization header is required",
				},
			})
			c.Abort()
			return
		}

		// Extract Bearer token
		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, response.APIResponse{
				Success: false,
				Error: &response.ErrorInfo{
					Code:    "INVALID_TOKEN_FORMAT",
					Message: "Authorization header must be Bearer token",
				},
			})
			c.Abort()
			return
		}

		token := tokenParts[1]
		claims, err := a.jwtService.ValidateAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, response.APIResponse{
				Success: false,
				Error: &response.ErrorInfo{
					Code:    "INVALID_TOKEN",
					Message: "Token is invalid or expired",
				},
			})
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

		c.Next()
	}
}

// RequireRole validates user has specific role
func (a *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, response.APIResponse{
				Success: false,
				Error: &response.ErrorInfo{
					Code:    "NO_ROLE_INFO",
					Message: "User role information not found",
				},
			})
			c.Abort()
			return
		}

		userRole := userRoles.(string)
		if slices.Contains(roles, userRole) {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, response.APIResponse{
			Success: false,
			Error: &response.ErrorInfo{
				Code:    "INSUFFICIENT_PERMISSIONS",
				Message: "User does not have required permissions",
			},
		})
		c.Abort()
	}
}
