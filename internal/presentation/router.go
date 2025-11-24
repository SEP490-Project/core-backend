package presentation

import (
	"core-backend/config"
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
	config             *config.AppConfig
	handlerRegistry    *handler.HandlerRegistry
	middlewareRegistry *middleware.MiddlewareRegistry
}

func NewRouter(
	config *config.AppConfig,
	handlerRegistry *handler.HandlerRegistry,
	middlewareRegistry *middleware.MiddlewareRegistry,
) *Router {
	return &Router{
		config:             config,
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

	// Affiliate link redirect (PUBLIC endpoint - no authentication required)
	redirectHandler := r.handlerRegistry.RedirectHandler
	engine.GET("/r/:hash", redirectHandler.Redirect)

	// API v1
	r.SetupV1Routes(engine)

	//File test
	engine.Static("/tmp", "./tmp")
	engine.Static("/html", "./templates/public")

	// Fallback route for undefined paths
	engine.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "Route not found"})
	})
}

// SetupV1Routes sets up version 1 API routes
func (r *Router) SetupV1Routes(engine *gin.Engine) {
	v1 := engine.Group("/api/v1")
	v1.Use(r.middlewareRegistry.Timeout)
	{
		// ---------- Routes Setups from functions ----------
		r.setupAuthRoutes(v1)
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
		r.SetupAffiliateLinkRoutes(v1)
		r.SetupAffiliateLinkAnalyticsRoutes(v1)
		r.SetupMarketingAnalyticsRoutes(v1)
		r.SetupPayOSRoutes(v1)
		r.setupFacebookSocialRoutes(v1)
		r.setupTikTokSocialRoutes(v1)
		r.setupPaymentTransactionsRoutes(v1)
		r.setupFileRoutes(v1)
		if r.config.IsDevelopmentDebugging() {
			r.setupTestRoutes(v1)
		}

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
				//Debt: do not allow brand to active this
				protectedProducts.PATCH("/publish/:id/:is-active", productHandler.PublishProduct)
			}

			// State update (Sales, Brand only)
			stateGroup := productsGroup.Group("")
			stateGroup.Use(r.middlewareRegistry.Auth.RequireRole(sales, brand, admin))
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

		// ---------- ORDERS ----------
		orderHandler := r.handlerRegistry.OrderHandler
		ordersGroup := v1.Group("/orders")
		ordersGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
		{
			// Get orders for current user with pagination
			ordersGroup.GET("", orderHandler.GetOrdersByUserIDWithPagination)
			//ordersGroup.POST(":id/pay", orderHandler.PayOrder)
			// Place and immediately pay
			ordersGroup.POST("", orderHandler.CreateOrder)
			ordersGroup.POST("/limited", orderHandler.CreateLimitedOrder)
			ordersGroup.PATCH("/received/:orderID", orderHandler.MarkAsReceived)

			// Customer: request early refund for an order
			ordersGroup.POST("/:orderID/refund", orderHandler.RequestRefund)
			// Customer: request compensation for an order
			ordersGroup.POST("/:orderID/compensation", orderHandler.RequestCompensation)
		}

		// Staffs
		staffOrdersGroup := v1.Group("/orders/staff")
		staffOrdersGroup.Use(r.middlewareRegistry.Auth.RequireRole(sales, admin))
		{
			staffOrdersGroup.GET("", orderHandler.GetStaffAvailableOrdersWithPagination)
			staffOrdersGroup.GET("/self-delivering", orderHandler.GetSelfDeliveringOrdersWithPagination)
			staffOrdersGroup.POST("/:orderID/censorship", orderHandler.OrderCensorship)
			staffOrdersGroup.PATCH("/readyToPickedUp/:orderID", orderHandler.MarkAsReadyToPickedUp)
			staffOrdersGroup.PATCH("/receivedAfterPickup/:orderID", orderHandler.MarkAsReceivedAfterPickedUp)
			// Self-delivering flow (LIMITED, not self pick-up)
			staffOrdersGroup.PATCH("/self-delivering/in-transit/:orderID", orderHandler.MarkSelfDeliveringOrderAsInTransit)
			staffOrdersGroup.PATCH("/self-delivering/delivered/:orderID", orderHandler.MarkSelfDeliveringOrderAsDelivered)

			// Staff: approve early refund and attach confirmation image
			staffOrdersGroup.POST("/:orderID/refund/approve", orderHandler.ApproveRefund)

			// Staff: process compensation requests (approve/reject)
			staffOrdersGroup.POST("/:orderID/compensation", orderHandler.ProcessCompensation)
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

		// ---------- LOCATIONS ----------/
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

		locationAdminGroup := v1.Group("/location")
		locationAdminGroup.Use(r.middlewareRegistry.Auth.RequireRole(admin))
		{
			// Use POST to match handler swagger and to allow triggering via POST
			locationAdminGroup.POST("/sync", locationHandler.TriggerLocationSync)
		}

		// ---------- GHN INTEGRATION ----------
		ghnHandler := r.handlerRegistry.GHNHandler
		ghnGroup := v1.Group("/ghn")
		ghnGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
		{
			ghnGroup.GET("/order/:order-id/shipping-services", ghnHandler.GetAvailableDeliveryServicesByOrderID)
			ghnGroup.POST("/order/:order-id/calculate", ghnHandler.CalculateDeliveryPriceByOrderID)
			// GHN order info (protected)
			ghnGroup.GET("/order/info/:order-id", ghnHandler.GetOrderInfo)
		}
		ghnPublicGroup := v1.Group("/ghn")
		{
			ghnPublicGroup.GET("/:district-id/shipping-services", ghnHandler.GetAvailableDeliveryServicesByDistrictID)
			ghnPublicGroup.POST("/delivery/calculate-by-dimension", ghnHandler.CalculateDeliveryPriceByDimension)
			// Public endpoint for expected delivery time
			ghnPublicGroup.GET("/expected-delivery-time", ghnHandler.GetExpectedDeliveryTime)

			//TET
			ghnPublicGroup.POST("/order/status", ghnHandler.UpdateGHNDeliveryStatus)
		}
		ghnMockingGroup := v1.Group("/ghn/mocking")
		{
			ghnMockingGroup.GET("/session", ghnHandler.GetGHNSession)
			ghnMockingGroup.POST("/order/:order-code/update-status", ghnHandler.UpdateGHNDeliveryStatus)
			ghnMockingGroup.GET("/service-token", ghnHandler.GetGHNServiceToken)
			ghnMockingGroup.GET("/gso-token", ghnHandler.GetGHNGSOToken)
		}

		// ---------- PRE-ORDERS ----------
		preOrderHandler := r.handlerRegistry.PreOrderHandler
		preOrderGroup := v1.Group("/preorders")
		preOrderGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
		{
			preOrderGroup.POST("", preOrderHandler.CreatePreOrderAndPay)
			preOrderGroup.GET("", preOrderHandler.GetAllPreorders)
			preOrderGroup.PATCH(":id/state", stateHandler.UpdatePreOrderState)
		}

		// Staffs
		staffPreOrderGroup := v1.Group("/preorders/staff")
		staffPreOrderGroup.Use(r.middlewareRegistry.Auth.RequireRole(sales, admin))
		{
			staffPreOrderGroup.GET("", preOrderHandler.GetStaffAvailablePreOrdersWithPagination)
			staffPreOrderGroup.POST("/:orderID/censorship", preOrderHandler.PreOrderCensorship)
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

		// Route cho cả ADMIN và MARKETING_STAFF
		userGroup.GET("", r.middlewareRegistry.Auth.RequireRole(admin, marketing), userHandler.GetUsers)

		// Admin only routes
		adminUserGroup := userGroup.Group("")
		adminUserGroup.Use(r.middlewareRegistry.Auth.RequireRole(admin))
		{
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

		//Brands only
		brandGroup := brands.Group("")
		brandGroup.Use(r.middlewareRegistry.Auth.RequireRole(brand))
		{
			brandGroup.GET("/my-product", brandHandler.MyProductsByFilter)
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
	contracts.GET("/brands/:brand_id", r.middlewareRegistry.Auth.RequireRole(brand, marketing), contractHandler.GetContractsByBrandID)
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
		editGroup.POST("/internal", campaignHandler.CreateInternalCampaign)
		editGroup.PUT("/id/:id", campaignHandler.UpdateCampaign)
		editGroup.DELETE("/id/:id", campaignHandler.DeleteCampaign)
		editGroup.GET("/:id/suggest", campaignHandler.SuggestCampaign)
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
	brandGroup.Use(r.middlewareRegistry.Auth.RequireRole(brand, admin))
	{
		brandGroup.GET("/brand/profile", campaignHandler.GetCampaignsByBrandProfile)
		brandGroup.PATCH("/id/:id/approve", campaignHandler.ApproveCampaign)
		brandGroup.PATCH("/id/:id/reject", campaignHandler.RejectCampaign)
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

		viewGroup := cPaymentGroup.Group("").Use(r.middlewareRegistry.Auth.RequireRole(admin, marketing, sales, brand))
		{
			viewGroup.GET("", contractPaymentHandler.GetContractPaymentsByFilter)
			viewGroup.GET("/:contract_payment_id", contractPaymentHandler.GetContractPaymentByID)
			viewGroup.POST("/:contract_payment_id/payment-link", contractPaymentHandler.GeneratePaymentLink)
		}

		brandGroup := cPaymentGroup.Group("").Use(r.middlewareRegistry.Auth.RequireRole(brand))
		{
			brandGroup.GET("/profile", contractPaymentHandler.GetContractPaymentByProfile)
		}
	}
}

// SetupChannelRoutes sets up routes for channel management
func (r *Router) SetupChannelRoutes(group *gin.RouterGroup) {
	channelHandler := r.handlerRegistry.ChannelHandler
	channelGroup := group.Group("/channels")
	{
		optionalAuthGroup := channelGroup.Group("")
		optionalAuthGroup.Use(r.middlewareRegistry.Auth.OptionalAuth())
		{
			optionalAuthGroup.GET("", channelHandler.GetAllChannels)
			optionalAuthGroup.GET("/:id", channelHandler.GetChannelByID)
		}

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
		publicGroup := contentGroup.Group("/public")
		{
			publicGroup.GET("", contentHandler.ListPublic)
			publicGroup.GET("/:id", contentHandler.GetByIDPublic)
		}
		viewGroup := contentGroup.Group("").Use(r.middlewareRegistry.Auth.RequireRole(customer, brand, marketing, sales, content, admin))
		{
			viewGroup.GET("", contentHandler.List)
			viewGroup.GET("/assigned_to", contentHandler.ListByAssignedUser)
			viewGroup.GET("/:id", contentHandler.GetByID)
		}

		editGroup := contentGroup.Group("").Use(r.middlewareRegistry.Auth.RequireRole(content, admin))
		{
			editGroup.POST("", contentHandler.Create)
			editGroup.PUT("/:id", contentHandler.Update)
			editGroup.DELETE("/:id", contentHandler.Delete)
			editGroup.PATCH("/:id/submit", contentHandler.Submit)
			editGroup.PUT("/:id/blog", blogHandler.UpdateBlogDetails)
			editGroup.POST("/:id/publish/channel/:channel_id", contentHandler.PublishToChannel)
			editGroup.POST("/:id/publish", contentHandler.PublishToAllChannels)
		}

		reviewGroup := contentGroup.Group("").Use(r.middlewareRegistry.Auth.RequireRole(admin, brand, marketing))
		{
			reviewGroup.PATCH("/:id/approve", contentHandler.Approve)
			reviewGroup.PATCH("/:id/reject", contentHandler.Reject)
		}
	}

	// Content channel status route (view publishing status)
	contentChannelGroup := group.Group("/content-channels")
	{
		statusGroup := contentChannelGroup.Group("").Use(r.middlewareRegistry.Auth.RequireRole(content, admin, brand, marketing))
		{
			statusGroup.GET("/:content_channel_id/status", contentHandler.GetPublishingStatus)
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
	notificationGroup.Use(r.middlewareRegistry.Auth.RequireRole(admin, content, marketing, sales, brand, customer))
	{
		// Read-only endpoints
		notificationGroup.GET("", notificationHandler.List)
		notificationGroup.GET("/failed", notificationHandler.GetFailedNotifications)
		notificationGroup.GET("/:id", notificationHandler.GetByID)

		// Write endpoints
		notificationGroup.PUT("/:id/read", notificationHandler.MarkAsRead)
		notificationGroup.PUT("/read-all", notificationHandler.MarkAllAsRead)

		// Testing/Publishing endpoints (Admin only)
		notificationGroup.POST("/publish", notificationHandler.PublishNotification)
		notificationGroup.POST("/publish/email", notificationHandler.PublishEmail)
		notificationGroup.POST("/publish/push", notificationHandler.PublishPush)
		notificationGroup.POST("/republish-failed", notificationHandler.RepublishFailed)

		// Broadcast endpoints (Admin only)
		notificationGroup.POST("/broadcast/user", notificationHandler.BroadcastToUser)
		notificationGroup.POST("/broadcast/all", notificationHandler.BroadcastToAll)
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

func (r *Router) SetupSSERoutes(engine *gin.Engine) {
	notificationHandler := r.handlerRegistry.NotificationHandler
	sseGroup := engine.Group("").Use(
		r.middlewareRegistry.Recovery,
		r.middlewareRegistry.RequestID,
		r.middlewareRegistry.Logging,
		r.middlewareRegistry.CORS,
		r.middlewareRegistry.Auth.RequireAuth(),
	)
	{
		sseGroup.GET("/api/v1/notifications/sse",
			notificationHandler.SubscribeSSE)
	}
}

// SetupAffiliateLinkRoutes sets up routes for affiliate link management
func (r *Router) SetupAffiliateLinkRoutes(group *gin.RouterGroup) {
	affiliateLinkHandler := r.handlerRegistry.AffiliateLinkHandler
	affiliateLinksGroup := group.Group("/affiliate-links")
	{
		// Protected routes (Sales, Admin only)
		protectedGroup := affiliateLinksGroup.Group("")
		protectedGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
		protectedGroup.Use(r.middlewareRegistry.Auth.RequireRole(sales, admin))
		protectedGroup.Use(r.middlewareRegistry.CSRF.Protect()) // T111: CSRF protection
		{
			protectedGroup.POST("", affiliateLinkHandler.Create)
			protectedGroup.GET("", affiliateLinkHandler.List)
			protectedGroup.GET("/:id", affiliateLinkHandler.GetByID)
			protectedGroup.PUT("/:id", affiliateLinkHandler.Update)
			protectedGroup.DELETE("/:id", affiliateLinkHandler.Delete)
		}
	}
}

// SetupAffiliateLinkAnalyticsRoutes sets up routes for affiliate link analytics
func (r *Router) SetupAffiliateLinkAnalyticsRoutes(group *gin.RouterGroup) {
	analyticsHandler := r.handlerRegistry.AffiliateLinkAnalyticsHandler
	analyticsGroup := group.Group("/analytics/affiliate-links")
	{
		// Protected routes (Admin, Marketing, Sales can view analytics)
		protectedGroup := analyticsGroup.Group("")
		protectedGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
		protectedGroup.Use(r.middlewareRegistry.Auth.RequireRole(admin, marketing, sales))
		{
			protectedGroup.GET("/by-contract/:contract_id", analyticsHandler.GetMetricsByContract)
			protectedGroup.GET("/by-channel", analyticsHandler.GetMetricsByChannel)
			protectedGroup.GET("/time-series/:affiliate_link_id", analyticsHandler.GetTimeSeriesData)
			protectedGroup.GET("/top-performers", analyticsHandler.GetTopPerformers)
			protectedGroup.GET("/dashboard", analyticsHandler.GetDashboard)
		}
	}
}

// SetupMarketingAnalyticsRoutes sets up routes for marketing analytics dashboard
func (r *Router) SetupMarketingAnalyticsRoutes(group *gin.RouterGroup) {
	marketingAnalyticsHandler := r.handlerRegistry.MarketingAnalyticsHandler
	analyticsGroup := group.Group("/analytics/marketing")
	{
		// Protected routes (Admin and Marketing Staff can view analytics)
		protectedGroup := analyticsGroup.Group("")
		protectedGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
		protectedGroup.Use(r.middlewareRegistry.Auth.RequireRole(admin, marketing))
		{
			protectedGroup.GET("/active-brands", marketingAnalyticsHandler.GetActiveBrandsCount)
			protectedGroup.GET("/active-campaigns", marketingAnalyticsHandler.GetActiveCampaignsCount)
			protectedGroup.GET("/draft-campaigns", marketingAnalyticsHandler.GetDraftCampaignsCount)
			protectedGroup.GET("/monthly-revenue", marketingAnalyticsHandler.GetMonthlyContractRevenue)
			protectedGroup.GET("/top-brands", marketingAnalyticsHandler.GetTopBrandsByRevenue)
			protectedGroup.GET("/revenue-by-type", marketingAnalyticsHandler.GetRevenueByContractType)
			protectedGroup.GET("/upcoming-deadlines", marketingAnalyticsHandler.GetUpcomingDeadlineCampaigns)
			protectedGroup.GET("/dashboard", marketingAnalyticsHandler.GetDashboard)
		}
	}
}

func (r *Router) SetupPayOSRoutes(group *gin.RouterGroup) {
	// ---------- PAYOS ----------
	payOsHandler := r.handlerRegistry.PayOsHandler

	// Public webhook endpoint (no authentication required for PayOS callbacks)
	group.POST("/payos/webhook", payOsHandler.HandleWebhook)
	group.GET("/payos/cancel-callback", payOsHandler.HandleCancelCallback)

	// Admin-protected PayOS routes
	payosGroup := group.Group("/payos")
	payosGroup.Use(r.middlewareRegistry.Auth.RequireRole(admin))
	{
		payosGroup.POST("/payment", payOsHandler.GeneratePaymentLink)
		payosGroup.GET("/payment/:orderCode", payOsHandler.GetPaymentInfo)
		payosGroup.POST("/cancel", payOsHandler.CancelExpiredLinks)
		payosGroup.POST("/confirm-webhook", payOsHandler.ConfirmWebhookURL)
	}
}

func (r *Router) setupPaymentTransactionsRoutes(group *gin.RouterGroup) {
	payOsHandler := r.handlerRegistry.PaymentTransactionsHandler

	viewGroup := group.Group("/payments")
	viewGroup.Use(r.middlewareRegistry.Auth.RequireRole(admin, marketing, sales, brand, customer))
	{
		viewGroup.GET("", payOsHandler.GetByFilter)
		viewGroup.GET("/profile", payOsHandler.GetByProfileFilter)
		viewGroup.GET("/id/:id", payOsHandler.GetByID)
		viewGroup.GET("/order-code/:order_code", payOsHandler.GetByOrderCode)
	}
}

func (r *Router) setupFacebookSocialRoutes(group *gin.RouterGroup) {
	facebookHandler := r.handlerRegistry.FacebookSocialHandler

	authFacebookGroup := group.Group("/auth/facebook")
	{
		authFacebookGroup.GET("/login", r.middlewareRegistry.Auth.OptionalAuth(), facebookHandler.HandleLogin)
		authFacebookGroup.GET("/callback", facebookHandler.HandleCallback)
	}
}

func (r *Router) setupTikTokSocialRoutes(group *gin.RouterGroup) {
	tiktokHandler := r.handlerRegistry.TikTokSocialHandler

	authTikTokGroup := group.Group("/auth/tiktok")
	{
		authTikTokGroup.GET("/login", r.middlewareRegistry.Auth.OptionalAuth(), tiktokHandler.HandleLogin)
		authTikTokGroup.GET("/callback", tiktokHandler.HandleCallback)
	}

	tiktokInfoGroup := group.Group("/social/tiktok")
	tiktokInfoGroup.Use(r.middlewareRegistry.Auth.RequireRole(admin))
	{
		tiktokInfoGroup.GET("/system-user-profile", tiktokHandler.GetSystemUserProfile)
		tiktokInfoGroup.GET("/creator-info", tiktokHandler.GetCreatorInfo)
	}
}

func (r *Router) setupAuthRoutes(group *gin.RouterGroup) {
	authHandler := r.handlerRegistry.AuthHandler
	authGroup := group.Group("/auth")
	{
		// Public
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/signup", authHandler.SignUp)
		authGroup.POST("/refresh", authHandler.RefreshToken)
		authGroup.POST("/forgot-password", authHandler.ForgotPassword)
		authGroup.POST("/reset-password", authHandler.ResetPassword)

		// Protected
		authProtectedGroup := authGroup.Group("")
		authProtectedGroup.Use(r.middlewareRegistry.Auth.RequireAuth())
		{
			authProtectedGroup.POST("/logout", authHandler.Logout)
			authProtectedGroup.POST("/logout-all", authHandler.LogoutAll)
			authProtectedGroup.GET("/sessions", authHandler.GetActiveSessions)
			authProtectedGroup.DELETE("/sessions/:sessionId", authHandler.RevokeSession)
			authProtectedGroup.POST("/change-password", authHandler.ChangePassword)
		}
	}
}

func (r *Router) setupTestRoutes(group *gin.RouterGroup) {
	testHandler := r.handlerRegistry.TestHandler
	testGroup := group.Group("/test")
	testGroup.Use(r.middlewareRegistry.Auth.RequireRole(admin))
	{
		testGroup.GET("/tiktok/exchange-code-for-token", testHandler.TikTokExchangeCodeForToken)
		testGroup.GET("/tiktok/refresh-access-token", testHandler.TikTokRefreshAccessToken)
		testGroup.GET("/tiktok/get-user-profile", testHandler.TikTokGetUserProfile)
		testGroup.GET("/tiktok/get-system-user-profile", testHandler.TikTokGetSystemUserProfile)
		testGroup.GET("/tiktok/get-creator-info", testHandler.TikTokGetCreatorInfo)
	}
}

func (r *Router) setupFileRoutes(group *gin.RouterGroup) {
	fileHandler := r.handlerRegistry.FileHandler

	filesGroup := group.Group("/files")
	{
		uploadGroup := filesGroup.Group("")
		{
			uploadFilesGroup := uploadGroup.Group("")
			{
				uploadFilesGroup.POST("/upload", fileHandler.UploadFile)
				uploadFilesGroup.DELETE(":filename", fileHandler.DeleteFile)
			}

			videosGroup := uploadGroup.Group("/videos")
			{
				// Upload chunk video (stream)
				videosGroup.POST("/upload-chunk", fileHandler.UploadVideoChunk)
				// Delete video
				videosGroup.DELETE("", fileHandler.DeleteVideo)
			}
		}

		getGroup := filesGroup.Group("")
		{
			getGroup.GET("/:key", fileHandler.GetFileDetailByS3Key)
			getGroup.GET("", fileHandler.GetFileByFilter)
		}
	}
}
