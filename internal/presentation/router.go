package presentation

import (
	"core-backend/internal/presentation/handler"
	"core-backend/internal/presentation/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	handlerRegistry    *handler.HandlerRegistry
	middlewareRegistry *middleware.MiddlewareRegistry
}

func NewRouter(
	handlerRegistry *handler.HandlerRegistry,
	middlewareRegistry *middleware.MiddlewareRegistry,
) *Router {
	return &Router{
		handlerRegistry:    handlerRegistry,
		middlewareRegistry: middlewareRegistry,
	}
}

func (r *Router) SetupRoutes(engine *gin.Engine) {
	r.middlewareRegistry.ApplyGlobalMiddlewares(engine)

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Service is healthy",
		})
	})
}

func (r *Router) SetupV1Routes(engine *gin.Engine) {
	v1 := engine.Group("/api/v1")
	{
		// Public auth routes (no authentication required)
		authHandler := r.handlerRegistry.AuthHandler
		authGroup := v1.
			Group("/auth")
		{
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/signup", authHandler.SignUp)
			authGroup.POST("/refresh", authHandler.RefreshToken)
		}

		// Protected auth routes (authentication required)
		authProtectedGroup := v1.
			Group("/auth").
			Use(r.middlewareRegistry.Auth.RequireAuth())
		{
			authProtectedGroup.POST("/logout", authHandler.Logout)
			authProtectedGroup.POST("/logout-all", authHandler.LogoutAll)
			authProtectedGroup.GET("/sessions", authHandler.GetActiveSessions)
			authProtectedGroup.DELETE("/sessions/:sessionId", authHandler.RevokeSession)
		}

		// User routes
		userHandler := r.handlerRegistry.UserHandler
		userGroup := v1.Group("/users")
		userGroup.Use(r.middlewareRegistry.Auth.RequireAuth()) // All user routes require authentication
		{
			// Current user profile routes (accessible by all authenticated users)
			userGroup.GET("/profile", userHandler.GetProfile)
			userGroup.PUT("/profile", userHandler.UpdateProfile)

			// Admin only routes
			adminUserGroup := userGroup.Group("/")
			adminUserGroup.Use(r.middlewareRegistry.Auth.RequireRole("admin"))
			{
				adminUserGroup.GET("", userHandler.GetUsers)
				adminUserGroup.GET("/:id", userHandler.GetUserByID)
				adminUserGroup.PUT("/:id/status", userHandler.UpdateUserStatus)
				adminUserGroup.PUT("/:id/role", userHandler.UpdateUserRole)
				adminUserGroup.DELETE("/:id", userHandler.DeleteUser)
			}
		}

		// FUTURE ROUTES FOR OTHER RESOURCES CAN BE ADDED HERE
	}
}
