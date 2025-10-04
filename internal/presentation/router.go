package presentation

import (
	"core-backend/docs"
	"core-backend/internal/domain/enum"
	"core-backend/internal/presentation/handler"
	"core-backend/internal/presentation/middleware"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
	marketing string = string(enum.UserRoleMarketingStaff)
	sales     string = string(enum.UserRoleSalesStaff)
	content   string = string(enum.UserRoleContentStaff)
	admin     string = string(enum.UserRoleAdmin)
	customer  string = string(enum.UserRoleCustomer)
	brand     string = string(enum.UserRoleBrandPartner)
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

	// Swagger documentation route
	swaggerHandler := func() gin.HandlerFunc {
		handler := ginSwagger.WrapHandler(swaggerFiles.Handler)

		return func(c *gin.Context) {
			host := c.Request.Host

			docs.SwaggerInfo.Host = host
			if strings.Contains(host, "localhost") {
				docs.SwaggerInfo.Schemes = []string{"http"}
			} else {
				docs.SwaggerInfo.Schemes = []string{"https"}
			}

			handler(c)
		}
	}
	engine.GET("/swagger/*any", swaggerHandler())

	// Handle favicon to avoid 404 errors
	engine.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(204) // No Content
	})

	// Health check routes
	healthHandler := r.handlerRegistry.HealthHandler
	engine.GET("/health", healthHandler.HealthCheck)
	engine.GET("/health/ready", healthHandler.ReadinessCheck)
	engine.GET("/health/live", healthHandler.LivenessCheck)

	// Setup version 1 API routes
	r.SetupV1Routes(engine)
}

// SetupV1Routes sets up version 1 API routes
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
			adminUserGroup.Use(r.middlewareRegistry.Auth.RequireRole(admin))
			{
				adminUserGroup.GET("", userHandler.GetUsers)
				adminUserGroup.GET("/:id", userHandler.GetUserByID)
				adminUserGroup.PUT("/:id/status", userHandler.UpdateUserStatus)
				adminUserGroup.PUT("/:id/role", userHandler.UpdateUserRole)
				adminUserGroup.DELETE("/:id", userHandler.DeleteUser)
				adminUserGroup.PATCH("/:id/activate-brand", userHandler.ActivateBrandUser)
			}
		}

		r.setupBrandRoutes(v1)
		r.SetupContractRoutes(v1)

		// Product routes
		productHandler := r.handlerRegistry.ProductHandler
		v1.GET("/products", productHandler.GetAllProducts)

		// Product state routes (protected)
		stateHandler := r.handlerRegistry.TaskHandler
		productStateGroup := v1.Group("/products")
		//productStateGroup.Use(r.middlewareRegistry.Auth.RequireRole(sales))
		productStateGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
		{
			productStateGroup.PATCH("/:id/state", stateHandler.UpdateProductState)
		}

		// Task routes
		taskHandler := r.handlerRegistry.TaskHandler
		taskGroup := v1.Group("/tasks")
		//taskGroup.Use(r.middlewareRegistry.Auth.RequireRole(sales, marketing, content, admin, brand))
		taskGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
		{
			taskGroup.PATCH(":id/state", taskHandler.UpdateTaskState)
		}

		// PayOS payment route
		payOsHandler := r.handlerRegistry.PayOsHandler
		v1.POST("/payos/payment", payOsHandler.GeneratePaymentLink)

		// File upload routes
		s3Handler := r.handlerRegistry.FileHandler
		fileGroup := v1.Group("/files")
		fileGroup.Use(r.middlewareRegistry.Auth.RequireAuth()) // All file routes require authentication
		{
			fileGroup.POST("/upload", s3Handler.UploadFile)
			//fileGroup.DELETE(":filename", s3Handler.DeleteFile)
		}

		// FUTURE ROUTES FOR OTHER RESOURCES CAN BE ADDED HERE
	}
}

// setupBrandRoutes sets up routes for brand management
func (r *Router) setupBrandRoutes(group *gin.RouterGroup) {
	brandHandler := r.handlerRegistry.BrandHandler

	brandGroup := group.Group("/brands")
	{
		brandGroup.GET("", brandHandler.GetBrandsByFilter)
		brandGroup.GET("/:id", brandHandler.GetBrandByID)
		brandGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin)).
			POST("", brandHandler.CreateBrand)
		brandGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing)).
			POST("/with-users", brandHandler.CreateBrandWithInActiveUsers)
		brandGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing)).
			PUT("/:id", brandHandler.UpdateBrand)
		brandGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin)).
			PATCH("/:id/status", brandHandler.UpdateBrandStatus)
	}
}

// SetupContractRoutes sets up routes for contract management
func (r *Router) SetupContractRoutes(group *gin.RouterGroup) {
	contractHandler := r.handlerRegistry.ContractHandler

	contractGroup := group.Group("/contracts")
	{
		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(brand, marketing, admin)).
			GET("", contractHandler.GetContracts)
		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing, brand)).
			GET("/:id", contractHandler.GetContractByID)
		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(brand)).
			GET("/brands/:brand_id", contractHandler.GetContractsByBrandID)

		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin)).
			POST("", contractHandler.CreateContract)

		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin)).
			PATCH("/:id/approve", contractHandler.ApproveContract)

		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing)).
			PUT("/:id", contractHandler.UpdateContract)

		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin)).
			DELETE("/:id", contractHandler.DeleteContract)
	}
}

// SetupContractRoutes sets up routes for contract management
func (r *Router) SetupContractRoutes(group *gin.RouterGroup) {
	contractHandler := r.handlerRegistry.ContractHandler

	contractGroup := group.Group("/contracts")
	{
		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(brand)).
			GET("/brands/:brand_id", contractHandler.GetContractsByBrandID)
		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(brand, marketing, admin)).
			GET("", contractHandler.GetContracts)
		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing, brand)).
			GET("/:id", contractHandler.GetContractByID)

		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin)).
			POST("", contractHandler.CreateContract)

		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin)).
			PATCH("/:id/approve", contractHandler.ApproveContract)

		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing)).
			PUT("/:id", contractHandler.UpdateContract)

		contractGroup.
			Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin)).
			DELETE("/:id", contractHandler.DeleteContract)
	}
}

// SetupWebSocketRoutes sets up WebSocket routes
func (r *Router) SetupWebSocketRoutes(engine *gin.Engine, wsServer *WebSocketServer) {
	// WebSocket endpoint (requires authentication)
	engine.GET("/ws",
		r.middlewareRegistry.Auth.RequireAuth(),
		wsServer.HandleWebSocket,
	)
}
