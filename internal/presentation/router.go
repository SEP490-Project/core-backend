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
	marketing = string(enum.UserRoleMarketingStaff)
	sales     = string(enum.UserRoleSalesStaff)
	content   = string(enum.UserRoleContentStaff)
	admin     = string(enum.UserRoleAdmin)
	customer  = string(enum.UserRoleCustomer)
	brand     = string(enum.UserRoleBrandPartner)
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

	// Swagger docs
	engine.GET("/swagger/*any", func(c *gin.Context) {
		handler := ginSwagger.WrapHandler(swaggerFiles.Handler)
		host := c.Request.Host
		docs.SwaggerInfo.Host = host
		if strings.Contains(host, "localhost") {
			docs.SwaggerInfo.Schemes = []string{"http"}
		} else {
			docs.SwaggerInfo.Schemes = []string{"https"}
		}
		handler(c)
	})

	// Favicon
	engine.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(204)
	})

	// Health check
	healthHandler := r.handlerRegistry.HealthHandler
	engine.GET("/health", healthHandler.HealthCheck)
	engine.GET("/health/ready", healthHandler.ReadinessCheck)
	engine.GET("/health/live", healthHandler.LivenessCheck)

	// API v1
	r.SetupV1Routes(engine)

	// Fallback route for undefined paths
	engine.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "Route not found"})
	})
}

// SetupV1Routes sets up version 1 API routes
func (r *Router) SetupV1Routes(engine *gin.Engine) {
	v1 := engine.Group("/api/v1")
	{
		// ---------- AUTH ----------
		authHandler := r.handlerRegistry.AuthHandler
		authGroup := v1.Group("/auth")
		{
			// Public
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/signup", authHandler.SignUp)
			authGroup.POST("/refresh", authHandler.RefreshToken)

			// Protected
			authProtectedGroup := authGroup.Group("")
			authProtectedGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
			{
				authProtectedGroup.POST("/logout", authHandler.Logout)
				authProtectedGroup.POST("/logout-all", authHandler.LogoutAll)
				authProtectedGroup.GET("/sessions", authHandler.GetActiveSessions)
				authProtectedGroup.DELETE("/sessions/:sessionId", authHandler.RevokeSession)
			}
		}

		// ---------- USERS ----------
		userHandler := r.handlerRegistry.UserHandler
		userGroup := v1.Group("/users")
		userGroup.Use(r.middlewareRegistry.Auth.RequireAuth()) // All user routes require authentication
		{
			// Current user profile routes (accessible by all authenticated users)
			userGroup.GET("/profile", userHandler.GetProfile)
			userGroup.PUT("/profile", userHandler.UpdateProfile)

			// Admin only routes
			adminUserGroup := userGroup.Group("")
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

		// ---------- Routes Setups from functions ----------
		r.setupBrandRoutes(v1)
		r.setupContractRoutes(v1)
		r.setupCampaignRoutes(v1)

		// ---------- PRODUCTS ----------
		productHandler := r.handlerRegistry.ProductHandler
		// Protected create route (sales, brand partner, admin can create)
		v1.POST("/products",
			r.middlewareRegistry.Auth.RequireRole(sales, brand, admin),
			productHandler.CreateProduct,
		)
		// Protected create variant route
		v1.POST("/products/:productId/variants",
			r.middlewareRegistry.Auth.RequireRole(sales, brand, admin),
			productHandler.CreateProductVariant,
		)
		stateHandler := r.handlerRegistry.StateHandler
		productsGroup := v1.Group("/products")
		{
			// Public
			productsGroup.GET("", productHandler.GetAllProducts)

			// Sales / Brand restricted
			productStateGroup := productsGroup.Group("")
			productStateGroup.Use(r.middlewareRegistry.Auth.RequireRole(sales, brand))
			{
				productStateGroup.PATCH("/:id/state", stateHandler.UpdateProductState)
			}

			// ---------- TASKS ----------
			taskGroup := v1.Group("/tasks")
			taskGroup.Use(r.middlewareRegistry.Auth.RequireRole(sales, content, admin, brand))
			{
				taskGroup.PATCH("/:id/state", stateHandler.UpdateTaskState)
				taskGroup.GET("/:taskId/products", productHandler.GetProductsByTask)
			}

			// Milestone routes (state transitions)
			milestoneGroup := v1.Group("/milestones")
			milestoneGroup.Use(r.middlewareRegistry.Auth.RequireRole(sales, content, admin, brand))
			{
				milestoneGroup.PATCH("/:id/state", stateHandler.UpdateMilestoneState)
			}

			// ---------- PAYOS ----------
			payOsHandler := r.handlerRegistry.PayOsHandler
			v1.POST("/payos/payment", payOsHandler.GeneratePaymentLink)

			// ---------- FILES ----------
			fileHandler := r.handlerRegistry.FileHandler
			filesGroup := v1.Group("/files")
			filesGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
			{
				filesGroup.POST("/upload", fileHandler.UploadFile)
				//filesGroup.DELETE(":filename", fileHandler.DeleteFile)
			}

			// FUTURE ROUTES FOR OTHER RESOURCES CAN BE ADDED HERE
		}
	}
}

// setupBrandRoutes sets up routes for brand management
func (r *Router) setupBrandRoutes(group *gin.RouterGroup) {
	brandHandler := r.handlerRegistry.BrandHandler
	brands := group.Group("/brands")
	{
		// Public
		brands.GET("", brandHandler.GetBrandsByFilter)
		brands.GET("/:id", brandHandler.GetBrandByID)

		// Marketing + Admin
		marketingAdmin := brands.Group("")
		marketingAdmin.Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin))
		{
			marketingAdmin.POST("", brandHandler.CreateBrand)
			marketingAdmin.PATCH("/:id/status", brandHandler.UpdateBrandStatus)
		}

		// Marketing only
		marketingGroup := brands.Group("")
		marketingGroup.Use(r.middlewareRegistry.Auth.RequireRole(marketing))
		{
			marketingGroup.POST("/with-users", brandHandler.CreateBrandWithInActiveUsers)
			marketingGroup.PUT("/:id", brandHandler.UpdateBrand)
		}
	}
}

// setupContractRoutes sets up routes for contract management
func (r *Router) setupContractRoutes(group *gin.RouterGroup) {
	contractHandler := r.handlerRegistry.ContractHandler
	contracts := group.Group("/contracts")

	// View routes with their specific role requirements
	contracts.GET("", r.middlewareRegistry.Auth.RequireRole(brand, marketing, admin), contractHandler.GetContracts)
	contracts.GET("/:id", r.middlewareRegistry.Auth.RequireRole(marketing, brand), contractHandler.GetContractByID)
	contracts.GET("/brands/profile", r.middlewareRegistry.Auth.RequireRole(brand), contractHandler.GetContractsByBrandProfile)
	contracts.GET("/brands/:brand_id", r.middlewareRegistry.Auth.RequireRole(brand), contractHandler.GetContractsByBrandID)

	// Write/Modify routes for Marketing and Admins
	adminAndMarketing := contracts.Group("")
	adminAndMarketing.Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin))
	{
		adminAndMarketing.POST("", contractHandler.CreateContract)
		adminAndMarketing.PATCH("/:id/approve", contractHandler.ApproveContract)
		adminAndMarketing.DELETE("/:id", contractHandler.DeleteContract)
	}

	// Update route for Marketing ONLY
	marketingOnly := contracts.Group("")
	marketingOnly.Use(r.middlewareRegistry.Auth.RequireRole(marketing))
	{
		marketingOnly.PUT("/:id", contractHandler.UpdateContract)
	}
}

// setupCampaignRoutes sets up routes for campaign management
func (r *Router) setupCampaignRoutes(group *gin.RouterGroup) {
	campaignHandler := r.handlerRegistry.CampaignHandler
	campaigns := group.Group("/campaigns")

	editGroup := campaigns.Group("")
	editGroup.Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin))
	{
		editGroup.POST("", campaignHandler.CreateCampaignFromContract)
		editGroup.DELETE("/id/:id", campaignHandler.DeleteCampaign)
	}

	viewGroup := campaigns.Group("")
	viewGroup.Use(r.middlewareRegistry.Auth.RequireRole(marketing, sales, content, admin, brand))
	{
		viewGroup.GET("/id/:id", campaignHandler.GetCampaignInfoByID)
		viewGroup.GET("/id/:id/details", campaignHandler.GetCampaignDetailsByID)
		viewGroup.GET("/contract/:contract_id", campaignHandler.GetCampaignInfoByContractID)
		viewGroup.GET("/contract/:contract_id/details", campaignHandler.GetCampaignDetailsByContractID)
		viewGroup.GET("/brand/:brand_id", campaignHandler.GetCampaignsInfoByBrandID)
		viewGroup.GET("", campaignHandler.GetCampaignsByFilter)
	}

	brandGroup := campaigns.Group("")
	brandGroup.Use(r.middlewareRegistry.Auth.RequireRole(brand))
	{
		brandGroup.GET("/brand/profile", campaignHandler.GetCampaignsByBrandProfile)
	}
}

// SetupWebSocketRoutes sets up WebSocket routes
func (r *Router) SetupWebSocketRoutes(engine *gin.Engine, wsServer *WebSocketServer) {
	engine.GET("/ws",
		r.middlewareRegistry.Auth.RequireAuth(),
		wsServer.HandleWebSocket,
	)
}
