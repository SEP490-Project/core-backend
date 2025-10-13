package handler

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type ProductHandler struct {
	productService iservice.ProductService
	fileService    iservice.FileService
	unitOfWork     irepository.UnitOfWork
	validator      *validator.Validate
}

func NewProductHandler(
	productService iservice.ProductService,
	fileService iservice.FileService,
	unitOfWork irepository.UnitOfWork,
) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		fileService:    fileService,
		unitOfWork:     unitOfWork,
		validator:      validator.New(),
	}
}

// GetAllProducts godoc
// @Summary      Get All Products
// @Description  Get paginated list of products with optional search
// @Tags         Products
// @Accept       json
// @Produce      json
// @Param        limit query int false "Number of items per page" default(10)
// @Param        offset query int false "Number of items to skip" default(0)
// @Param        search query string false "Search term for product name"
// @Success      200 {object} object{data=[]responses.ProductResponse,total=int,limit=int,offset=int} "Products retrieved successfully"
// @Failure      500 {object} object{error=string} "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/products [get]
func (h *ProductHandler) GetAllProducts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	search := c.DefaultQuery("search", "")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	products, total, err := (h.productService).GetProductsPagination(limit, offset, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   products,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetProductsByTask godoc
// @Summary      Get Products By Task
// @Description  Get paginated products (overview) belonging to a task. Authorization: staff roles or owning brand user.
// @Tags         Products
// @Accept       json
// @Produce      json
// @Param        taskId path string true "Task ID (UUID)"
// @Param        limit  query int false "Number of items per page" default(10)
// @Param        offset query int false "Number of items to skip" default(0)
// @Success      200 {object} object{data=[]responses.ProductOverviewResponse,total=int,limit=int,offset=int}
// @Failure      400 {object} object{error=string}
// @Failure      403 {object} object{error=string}
// @Failure      500 {object} object{error=string}
// @Security     BearerAuth
// @Router       /api/v1/tasks/{taskId}/products [get]
func (h *ProductHandler) GetProductsByTask(c *gin.Context) {
	// Parse path param
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	// Pagination params
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Auth context
	userIDVal, ok := c.Get("user_id")
	if !ok || userIDVal == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "missing user id in context"})
		return
	}
	userIDStr, _ := userIDVal.(string)
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid user id in context"})
		return
	}
	roleVal, _ := c.Get("roles")
	roleStr, _ := roleVal.(string)

	overview, total, svcErr := h.productService.GetProductsByTask(taskID, userUUID, roleStr, limit, offset)
	if svcErr != nil {
		if strings.Contains(strings.ToLower(svcErr.Error()), "forbidden") {
			c.JSON(http.StatusForbidden, gin.H{"error": svcErr.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": svcErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   overview,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// CreateProduct godoc
// @Summary      Create Product
// @Description  Create a new product (initial state DRAFT)
// @Tags         Products
// @Accept       json
// @Produce      json
// @Param        body  body  requests.CreateProductRequest  true  "Product to create"
// @Success      201  {object} responses.ProductResponse
// @Failure      400  {object} object{error=string}
// @Failure      401  {object} object{error=string}
// @Failure      500  {object} object{error=string}
// @Security     BearerAuth
// @Router       /api/v1/products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req requests.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}
	if h.validator == nil {
		h.validator = validator.New()
	}
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed: " + err.Error()})
		return
	}

	uidVal, ok := c.Get("user_id")
	if !ok || uidVal == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id in context"})
		return
	}
	uidStr, _ := uidVal.(string)
	creatorID, err := uuid.Parse(uidStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in context"})
		return
	}

	product, err := h.productService.CreateProduct(&req, creatorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, product)
}

// CreateProductVariant godoc
// @Summary      Create Product Variant
// @Description  Create a new product variant with story and attributes
// @Tags         Products
// @Accept       json
// @Produce      json
// @Param        productId  path  string  true  "Product ID (UUID)"
// @Param        body       body  requests.BulkVariantRequest  true  "Variant data to create"
// @Success      201        {object} responses.ProductVariantResponse
// @Failure      400        {object} object{error=string}
// @Failure      401        {object} object{error=string}
// @Failure      500        {object} object{error=string}
// @Security     BearerAuth
// @Router       /api/v1/products/{productId}/variants [post]
func (h *ProductHandler) CreateProductVariant(c *gin.Context) {
	userIDData, exists := c.Get("user_id")
	if !exists || userIDData == "" {
		responses := responses.ErrorResponse("Unauthorized: user_id not found in context", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, responses)
		return
	}

	userIDStr := userIDData.(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in context"})
		return
	}

	productIDStr := c.Param("productId")
	productID, err := uuid.Parse(productIDStr)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var req requests.BulkVariantRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if h.validator == nil {
		h.validator = validator.New()
	}
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed: " + err.Error()})
		return
	}

	ctx := context.Background()
	uow := h.unitOfWork.Begin()

	//Create variant
	createdVariants := req.CreateProductVariantRequest
	variant, err := h.productService.CreateProductVariance(ctx, userID, productID, createdVariants, uow)
	if err != nil {
		err := uow.Rollback()
		if err != nil {
			zap.L().Info(err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	//Create story for variant
	productStory := req.Story
	_, err = h.productService.CreateProductStory(ctx, variant.ID, productStory, uow)
	if err != nil {
		err := uow.Rollback()
		if err != nil {
			zap.L().Info(err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	//File Upload and collect URLS
	uow.Rollback()

	//svcErr := h.productService.CreateProductVariance(userID, productID, req)
	//if svcErr != nil {
	//	c.JSON(http.StatusInternalServerError, gin.H{"error": svcErr.Error()})
	//	return
	//}
	//c.Status(http.StatusCreated)
}
