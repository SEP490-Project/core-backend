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
}

// SetupV1Routes sets up the API v1 routes with appropriate handlers and middleware.
func (r *Router) SetupV1Routes(engine *gin.Engine) {
	v1 := engine.Group("/api/v1")

	// ---------- AUTH ----------
	authHandler := r.handlerRegistry.AuthHandler
	auth := v1.Group("/auth")
	{
		// Public
		auth.POST("/login", authHandler.Login)
		auth.POST("/signup", authHandler.SignUp)
		auth.POST("/refresh", authHandler.RefreshToken)

		// Protected
		authProtected := auth.Group("/")
		authProtected.Use(r.middlewareRegistry.Auth.RequireAuth())
		{
			authProtected.POST("/logout", authHandler.Logout)
			authProtected.POST("/logout-all", authHandler.LogoutAll)
			authProtected.GET("/sessions", authHandler.GetActiveSessions)
			authProtected.DELETE("/sessions/:sessionId", authHandler.RevokeSession)
		}
	}

	// ---------- USERS ----------
	userHandler := r.handlerRegistry.UserHandler
	users := v1.Group("/users")
	users.Use(r.middlewareRegistry.Auth.RequireAuth())
	{
		// Profile (any authenticated user)
		users.GET("/profile", userHandler.GetProfile)
		users.PUT("/profile", userHandler.UpdateProfile)

		// Admin-only
		adminGroup := users.Group("/")
		adminGroup.Use(r.middlewareRegistry.Auth.RequireRole(admin))
		{
			adminGroup.GET("", userHandler.GetUsers)
			adminGroup.GET("/:id", userHandler.GetUserByID)
			adminGroup.PUT("/:id/status", userHandler.UpdateUserStatus)
			adminGroup.PUT("/:id/role", userHandler.UpdateUserRole)
			adminGroup.DELETE("/:id", userHandler.DeleteUser)
			adminGroup.PATCH("/:id/activate-brand", userHandler.ActivateBrandUser)
		}
	}

	// ---------- Routes Setups from functions ----------
	r.setupBrandRoutes(v1)
	r.setupContractRoutes(v1)
	r.setupCampaignRoutes(v1)

	// ---------- PRODUCTS ----------
	productHandler := r.handlerRegistry.ProductHandler
	stateHandler := r.handlerRegistry.StateHandler
	products := v1.Group("/products")
	{
		// Public
		products.GET("", productHandler.GetAllProducts)

		// Sales / Brand restricted
		productState := products.Group("/")
		productState.Use(r.middlewareRegistry.Auth.RequireRole(sales, brand))
		{
			productState.PATCH("/:id/state", stateHandler.UpdateProductState)
		}
	}

	// ---------- TASKS ----------
	tasks := v1.Group("/tasks")
	tasks.Use(r.middlewareRegistry.Auth.RequireRole(sales, content, admin, brand))
	{
		tasks.PATCH("/:id/state", stateHandler.UpdateTaskState)
	}

	// ---------- PAYOS ----------
	payOsHandler := r.handlerRegistry.PayOsHandler
	v1.POST("/payos/payment", payOsHandler.GeneratePaymentLink)

	// ---------- FILES ----------
	fileHandler := r.handlerRegistry.FileHandler
	files := v1.Group("/files")
	files.Use(r.middlewareRegistry.Auth.RequireAuth())
	{
		files.POST("/upload", fileHandler.UploadFile)
	}
}

// =============================
// ====== BRAND ROUTES =========
// =============================
func (r *Router) setupBrandRoutes(group *gin.RouterGroup) {
	brandHandler := r.handlerRegistry.BrandHandler
	brands := group.Group("/brands")
	{
		// Public
		brands.GET("", brandHandler.GetBrandsByFilter)
		brands.GET("/:id", brandHandler.GetBrandByID)

		// Marketing + Admin
		marketingAdmin := brands.Group("/")
		marketingAdmin.Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin))
		{
			marketingAdmin.POST("", brandHandler.CreateBrand)
			marketingAdmin.PATCH("/:id/status", brandHandler.UpdateBrandStatus)
		}

		// Marketing only
		marketingGroup := brands.Group("/")
		marketingGroup.Use(r.middlewareRegistry.Auth.RequireRole(marketing))
		{
			marketingGroup.POST("/with-users", brandHandler.CreateBrandWithInActiveUsers)
			marketingGroup.PUT("/:id", brandHandler.UpdateBrand)
		}
	}
}

// =============================
// ====== CONTRACT ROUTES ======
// =============================
func (r *Router) setupContractRoutes(group *gin.RouterGroup) {
	contractHandler := r.handlerRegistry.ContractHandler
	contracts := group.Group("/contracts")

	// View routes with their specific role requirements
	contracts.GET("", r.middlewareRegistry.Auth.RequireRole(brand, marketing, admin), contractHandler.GetContracts)
	contracts.GET("/:id", r.middlewareRegistry.Auth.RequireRole(marketing, brand), contractHandler.GetContractByID)
	contracts.GET("/brands/:brand_id", r.middlewareRegistry.Auth.RequireRole(brand), contractHandler.GetContractsByBrandID)

	// Write/Modify routes for Marketing and Admins
	adminAndMarketing := contracts.Group("/")
	adminAndMarketing.Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin))
	{
		adminAndMarketing.POST("", contractHandler.CreateContract)
		adminAndMarketing.PATCH("/:id/approve", contractHandler.ApproveContract)
		adminAndMarketing.DELETE("/:id", contractHandler.DeleteContract)
	}

	// Update route for Marketing ONLY
	marketingOnly := contracts.Group("/")
	marketingOnly.Use(r.middlewareRegistry.Auth.RequireRole(marketing))
	{
		marketingOnly.PUT("/:id", contractHandler.UpdateContract)
	}
}

// =============================
// ====== CAMPAIGN ROUTES ======
// =============================
func (r *Router) setupCampaignRoutes(group *gin.RouterGroup) {
	campaignHandler := r.handlerRegistry.CampaignHandler
	campaigns := group.Group("/campaigns")

	editGroup := campaigns.Group("/")
	editGroup.Use(r.middlewareRegistry.Auth.RequireRole(marketing, admin))
	{
		editGroup.POST("", campaignHandler.CreateCampaignFromContract)
		editGroup.DELETE("/id/:id", campaignHandler.DeleteCampaign)
	}

	viewGroup := campaigns.Group("/")
	viewGroup.Use(r.middlewareRegistry.Auth.RequireRole(marketing, sales, content, admin, brand))
	{
		viewGroup.GET("/id/:id", campaignHandler.GetCampaignInfoByID)
		viewGroup.GET("/id/:id/details", campaignHandler.GetCampaignDetailsByID)
		viewGroup.GET("/contract/:contract_id", campaignHandler.GetCampaignInfoByContractID)
		viewGroup.GET("/contract/:contract_id/details", campaignHandler.GetCampaignDetailsByContractID)
		viewGroup.GET("/brand/:brand_id", campaignHandler.GetCampaignsInfoByBrandID)
		viewGroup.GET("", campaignHandler.GetCampaignsByFilter)
	}

	brandGroup := campaigns.Group("/")
	brandGroup.Use(r.middlewareRegistry.Auth.RequireRole(brand))
	{
		brandGroup.GET("/brand/profile", campaignHandler.GetCampaignsByBrandProfile)
	}
}

// SetupWebSocketRoutes sets up the WebSocket routes with appropriate handlers and middleware.
func (r *Router) SetupWebSocketRoutes(engine *gin.Engine, wsServer *WebSocketServer) {
	engine.GET("/ws",
		r.middlewareRegistry.Auth.RequireAuth(),
		wsServer.HandleWebSocket,
	)
}
