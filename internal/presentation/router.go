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

	//File test
	engine.Static("/tmp", "./tmp")

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

		// ---------- Routes Setups from functions ----------
		r.setupUserRoutes(v1)
		r.setupBrandRoutes(v1)
		r.setupContractRoutes(v1)
		r.setupCampaignRoutes(v1)
		r.SetupContractPaymentRoutes(v1)
		r.SetupModifiedHistoryRouter(v1)
		r.SetupAdminConfigRouter(v1)
		r.SetupChannelRoutes(v1)
		r.SetupContentRoutes(v1)
		r.SetupTaskRoutes(v1)
		r.SetupDeviceTokenRoutes(v1)
		r.SetupNotificationRoutes(v1)
		r.SetupTagRoutes(v1)

		// ---------- PRODUCTS & VARIANTS ----------
		productHandler := r.handlerRegistry.ProductHandler
		stateHandler := r.handlerRegistry.StateHandler

		productsGroup := v1.Group("/products")
		{
			// Public
			productsGroup.GET("", productHandler.GetAllProducts)

			// Optional
			optionalGroup := productsGroup.Group("")
			optionalGroup.Use(r.middlewareRegistry.Auth.RequireAuthOptional())
			{
				optionalGroup.GET("/v2", productHandler.GetAllProductsV2)
			}

			productsGroup.GET("/:id", productHandler.GetProductDetail)

			// Protected (Sales, Brand, Admin)
			protectedProducts := productsGroup.Group("")
			protectedProducts.Use(r.middlewareRegistry.Auth.RequireRole(sales, brand, admin))
			{
				protectedProducts.POST("/standard", productHandler.CreateStandardProduct)
				protectedProducts.POST("/limited", productHandler.CreateLimitedProduct)
				protectedProducts.POST("/limited/:limited-id/concept/:concept-id", productHandler.AddConceptToLimitedProduct)
				protectedProducts.POST("/:productId/variants", productHandler.CreateProductVariant)
				protectedProducts.POST("/variants/:variantId/images", productHandler.CreateVariantImage)
			}

			// State update (Sales, Brand only)
			stateGroup := productsGroup.Group("")
			stateGroup.Use(r.middlewareRegistry.Auth.RequireRole(sales, admin))
			{
				stateGroup.PATCH("/:id/state", stateHandler.UpdateProductState)
			}
		}

		variantAttributeGroup := v1.Group("/variant-attributes")
		{
			//Public

			//Optional
			optionalGroup := variantAttributeGroup.Group("")
			optionalGroup.Use(r.middlewareRegistry.Auth.RequireAuthOptional())
			{
				optionalGroup.GET("", productHandler.GetVariantAttributePagination)
			}
			//Rules

		}
		// Variant Attributes (Sales, Brand, Admin)
		v1.POST("/variant-attributes",
			r.middlewareRegistry.Auth.RequireRole(sales, admin),
			productHandler.CreateVariantAttribute,
		)

		// ---------- CATEGORIES ----------
		categoryHandler := r.handlerRegistry.CategoryHandler
		categoriesGroup := v1.Group("/categories")
		{
			categoriesGroup.GET("", categoryHandler.GetAllCategories)
			categoriesGroup.POST("",
				r.middlewareRegistry.Auth.RequireRole(sales, admin),
				categoryHandler.CreateCategory,
			)
			categoriesGroup.PATCH("/:id/parent",
				r.middlewareRegistry.Auth.RequireRole(sales, admin),
				categoryHandler.AssignParentCategory,
			)
			categoriesGroup.DELETE("/:id",
				r.middlewareRegistry.Auth.RequireRole(sales, admin),
				categoryHandler.DeleteCategory,
			)
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
		v1.POST("/payos/cancel", payOsHandler.CancelCallback)

		// ---------- FILES ----------
		fileHandler := r.handlerRegistry.FileHandler
		filesGroup := v1.Group("/files")
		//filesGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
		{
			filesGroup.POST("/upload", fileHandler.UploadFile)
			filesGroup.DELETE(":filename", fileHandler.DeleteFile)

			// ---------------- Videos ----------------
			videosGroup := filesGroup.Group("/videos")
			{
				// Upload chunk video (stream)
				videosGroup.POST("/upload-chunk", fileHandler.UploadVideoChunk)

				// Xóa video
				videosGroup.DELETE("", fileHandler.DeleteVideo)
			}
		}

		// ---------- ORDERS ----------
		orderHandler := r.handlerRegistry.OrderHandler
		ordersGroup := v1.Group("/orders")
		ordersGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
		{
			// Get orders for current user with pagination
			ordersGroup.GET("", orderHandler.GetOrdersByUserIDWithPagination)
			ordersGroup.POST("", orderHandler.PlaceOrder)
			ordersGroup.POST("/:id/pay", orderHandler.PayOrder)
		}

		// ---------- CONCEPTS ----------
		conceptHandler := r.handlerRegistry.ConceptHandler
		conceptsGroup := v1.Group("/concepts")
		{
			// Public list
			conceptsGroup.GET("", conceptHandler.GetConcepts)

			// Protected (marketing, admin)
			protected := conceptsGroup.Group("")
			protected.Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin, sales, content))
			{
				protected.POST("", conceptHandler.CreateConcept)
				protected.DELETE("/:id", conceptHandler.DeleteConcept)
			}
		}

		// ---------- LOCATIONS ----------
		locationHandler := r.handlerRegistry.LocationHandler
		locationGroup := v1.Group("/location")
		locationGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
		{
			locationGroup.GET("/provinces", locationHandler.GetProvinces)
			locationGroup.GET("/districts/:province-id", locationHandler.GetDistricts)
			locationGroup.GET("/wards/:district-id", locationHandler.GetWards)
			// Address management for authenticated users
			locationGroup.POST("/address", locationHandler.InputUserAddress)
			locationGroup.PATCH("/address/:address-id/default", locationHandler.SetAddressAsDefault)
			locationGroup.GET("/addresses", locationHandler.GetUserAddresses)
		}

		// FUTURE ROUTES FOR OTHER RESOURCES CAN BE ADDED HERE
	}

}

// setupUserRoutes sets up routes for user management
func (r *Router) setupUserRoutes(group *gin.RouterGroup) {
	userHandler := r.handlerRegistry.UserHandler
	userGroup := group.Group("/users")
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

		preferenceGroup := userGroup.Group("/notification-preferences")
		{
			preferenceGroup.GET("", userHandler.GetUserPreference)
			preferenceGroup.PUT("", userHandler.UpdateUserPreferences)
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
	stateHandler := r.handlerRegistry.StateHandler
	contracts := group.Group("/contracts")

	// View routes with their specific role requirements
	contracts.GET("", r.middlewareRegistry.Auth.RequireRole(brand, marketing, admin), contractHandler.GetContracts)
	contracts.GET("/:id", r.middlewareRegistry.Auth.RequireRole(marketing, brand), contractHandler.GetContractByID)
	contracts.GET("/brands/profile", r.middlewareRegistry.Auth.RequireRole(brand), contractHandler.GetContractsByBrandProfile)
	contracts.GET("/brands/:brand_id", r.middlewareRegistry.Auth.RequireRole(brand), contractHandler.GetContractsByBrandID)
	contracts.PATCH("/:id/state", r.middlewareRegistry.Auth.RequireRole(brand, marketing, admin), stateHandler.UpdateContractState)

	// Write/Modify routes for Marketing and Admins
	adminAndMarketing := contracts.Group("")
	adminAndMarketing.Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin))
	{
		adminAndMarketing.POST("", contractHandler.CreateContract)
		adminAndMarketing.POST("/async", contractHandler.CreateContractAsync)
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

	suggestGroup := campaigns.Group("")
	suggestGroup.Use(r.middlewareRegistry.Auth.RequireRole(marketing))
	{
		suggestGroup.GET("/:campaign_id/suggest", campaignHandler.SuggestCampaign)
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

// SetupModifiedHistoryRouter sets up routes for modified history management
func (r *Router) SetupModifiedHistoryRouter(group *gin.RouterGroup) {
}

// SetupAdminConfigRouter sets up routes for admin configuration management
func (r *Router) SetupAdminConfigRouter(group *gin.RouterGroup) {
	adminConfigHandler := r.handlerRegistry.AdminConfigHandler
	configGroup := group.Group("configs")
	{
		writeGroup := configGroup.Group("").Use(r.middlewareRegistry.Auth.RequireRole(admin))
		{
			writeGroup.GET("", adminConfigHandler.GetAllConfigValues)
		}

		readGroup := configGroup.Group("").Use(r.middlewareRegistry.Auth.RequireRole(admin, marketing, sales, content))
		{
			readGroup.GET("/representative", adminConfigHandler.GetRepresentativeConfigs)
		}
	}
}

// SetupContractPaymentRoutes sets up routes for contract payment management
func (r *Router) SetupContractPaymentRoutes(group *gin.RouterGroup) {
	contractPaymentHandler := r.handlerRegistry.ContractPaymentHandler
	cPaymentGroup := group.Group("contract_payments")
	{
		marketingGroup := cPaymentGroup.Group("").Use(r.middlewareRegistry.Auth.RequireRole(marketing))
		{
			marketingGroup.POST("/contract/:contract_id", contractPaymentHandler.CreateContractPaymentsFromContract)
		}
	}
}

// SetupChannelRoutes sets up routes for channel management
func (r *Router) SetupChannelRoutes(group *gin.RouterGroup) {
	channelHandler := r.handlerRegistry.ChannelHandler
	channelGroup := group.Group("/channels")
	{
		channelGroup.GET("", channelHandler.GetAllChannels)
		channelGroup.GET("/:id", channelHandler.GetChannelByID)

		authenticatedGroup := channelGroup.Group("").Use(r.middlewareRegistry.Auth.RequireRole(admin, marketing, sales))
		{
			authenticatedGroup.POST("", channelHandler.CreateChannel)
			authenticatedGroup.PUT("/:id", channelHandler.UpdateChannel)
			authenticatedGroup.DELETE("/:id", channelHandler.DeleteChannel)
		}
	}
}

// SetupContentRoutes sets up routes for content management
func (r *Router) SetupContentRoutes(group *gin.RouterGroup) {
	contentHandler := r.handlerRegistry.ContentHandler
	blogHandler := r.handlerRegistry.BlogHandler
	contentGroup := group.Group("/contents")
	{
		viewGroup := contentGroup.Group("").Use(r.middlewareRegistry.Auth.RequireRole(customer, brand, marketing, sales, content, admin))
		{
			viewGroup.GET("", contentHandler.List)
			viewGroup.GET("/:id", contentHandler.GetByID)
		}

		editGroup := contentGroup.Group("").Use(r.middlewareRegistry.Auth.RequireRole(content))
		{
			editGroup.POST("", contentHandler.Create)
			editGroup.PUT("/:id", contentHandler.Update)
			editGroup.DELETE("/:id", contentHandler.Delete)
			editGroup.PATCH("/:id/submit", contentHandler.Submit)
			editGroup.PATCH("/:id/publish", contentHandler.Publish)
			editGroup.PUT("/:id/blog", blogHandler.UpdateBlogDetails)
		}

		reviewGroup := contentGroup.Group("").Use(r.middlewareRegistry.Auth.RequireRole(admin, brand, marketing))
		{
			reviewGroup.PATCH("/:id/approve", contentHandler.Approve)
			reviewGroup.PATCH("/:id/reject", contentHandler.Reject)
		}
	}
}

// SetupTaskRoutes sets up routes for task management
func (r *Router) SetupTaskRoutes(group *gin.RouterGroup) {
	taskHandler := r.handlerRegistry.TaskHandler
	stateHandler := r.handlerRegistry.StateHandler
	productHandler := r.handlerRegistry.ProductHandler

	taskGroup := group.Group("/tasks")
	taskGroup.Use(r.middlewareRegistry.Auth.RequireRole(marketing, sales, content, admin, brand))
	{
		viewGroup := taskGroup.Group("")
		viewGroup.Use(r.middlewareRegistry.Auth.RequireRole(marketing, sales, content, admin, brand))
		{
			viewGroup.GET("", taskHandler.GetTasksByFilter)
			viewGroup.GET("/:task_id", taskHandler.GetTaskByID)
			viewGroup.GET("/:task_id/products", productHandler.GetProductsByTask)
			viewGroup.GET("/contract/:contract_id", taskHandler.GetTasksByContractID)
			viewGroup.GET("/profile", taskHandler.GetTasksByProfile)
		}

		editGroup := taskGroup.Group("")
		editGroup.Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin))
		{
			editGroup.PATCH("/:task_id/assign/:assigned_to_id", taskHandler.AssignTask)
			editGroup.POST("", taskHandler.CreateTask)
			editGroup.PUT("/:task_id", taskHandler.UpdateTaskByID)
			editGroup.DELETE("/:task_id", taskHandler.DeleteTask)
		}

		stateGroup := taskGroup.Group("")
		stateGroup.Use(r.middlewareRegistry.Auth.RequireRole(marketing, sales, content, admin, brand))
		{
			stateGroup.PATCH("/:task_id/state", stateHandler.UpdateTaskState)
		}
	}
}

func (r *Router) SetupDeviceTokenRoutes(group *gin.RouterGroup) {
	deviceTokenHandler := r.handlerRegistry.DeviceTokenHandler
	deviceTokenGroup := group.Group("/device-tokens")
	deviceTokenGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
	{
		deviceTokenGroup.POST("", deviceTokenHandler.Register)
		deviceTokenGroup.GET("", deviceTokenHandler.List)
		deviceTokenGroup.PUT("/:id", deviceTokenHandler.Update)
		deviceTokenGroup.DELETE("/:id", deviceTokenHandler.Delete)
		deviceTokenGroup.DELETE("", deviceTokenHandler.DeleteAll)
	}
}

func (r *Router) SetupNotificationRoutes(group *gin.RouterGroup) {
	notificationHandler := r.handlerRegistry.NotificationHandler
	notificationGroup := group.Group("/notifications")
	notificationGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
	notificationGroup.Use(r.middlewareRegistry.Auth.RequireRole("ADMIN"))
	{
		notificationGroup.GET("", notificationHandler.List)
		notificationGroup.GET("/failed", notificationHandler.GetFailedNotifications)
		notificationGroup.GET("/:id", notificationHandler.GetByID)
	}
}

func (r *Router) SetupTagRoutes(group *gin.RouterGroup) {
	tagHandler := r.handlerRegistry.TagHandler
	tagsGroup := group.Group("/tags")
	{
		editGroup := tagsGroup.Group("")
		editGroup.Use(r.middlewareRegistry.Auth.RequireRole(admin, content, marketing, sales))
		{
			editGroup.GET("/:tag_id", tagHandler.GetByID)
			editGroup.GET("/name/:name", tagHandler.GetByName)
			editGroup.POST("", tagHandler.Create)
			editGroup.PUT("/:tag_id", tagHandler.UpdateByID)
			editGroup.DELETE("/:tag_id", tagHandler.DeleteByID)
		}

		viewGroup := tagsGroup.Group("")
		viewGroup.Use(r.middlewareRegistry.Auth.RequireRole(admin, content, marketing, sales, brand, customer))
		{
			viewGroup.GET("", tagHandler.GetByFilter)
		}
	}
}

// SetupWebSocketRoutes sets up WebSocket routes
func (r *Router) SetupWebSocketRoutes(engine *gin.Engine, wsServer *WebSocketServer) {
	engine.GET("/ws",
		r.middlewareRegistry.Auth.RequireAuth(),
		wsServer.HandleWebSocket,
	)
}
