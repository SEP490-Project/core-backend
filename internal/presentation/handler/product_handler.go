package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/pkg/utils"
	"net/http"
	"os"
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
//
//	@Summary		Get All Products
//	@Description	Get paginated list of products with optional search
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int																		false	"Number of items per page"	default(10)
//	@Param			offset	query		int																		false	"Number of items to skip"	default(0)
//	@Param			search	query		string																	false	"Search term for product name"
//	@Param			category_id	query		string																	false	"Filter category of products"
//	@Param			type	query		string																	false	"Filter type of products"
//	@Success		200		{object}	object{data=[]responses.ProductResponse,total=int,limit=int,offset=int}	"Products retrieved successfully"
//	@Failure		500		{object}	object{error=string}													"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/products [get]
func (h *ProductHandler) GetAllProducts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	search := c.DefaultQuery("search", "")
	category := c.DefaultQuery("category_id", "")
	ptype := c.DefaultQuery("type", "")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	products, total, err := (h.productService).GetProductsPagination(limit, offset, search, category, ptype)
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
//
//	@Summary		Get Products By Task
//	@Description	Get paginated products (overview) belonging to a task. Authorization: staff roles or owning brand user.
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			taskId	path		string	true	"Task ID (UUID)"
//	@Param			limit	query		int		false	"Number of items per page"	default(10)
//	@Param			offset	query		int		false	"Number of items to skip"	default(0)
//	@Success		200		{object}	object{data=[]responses.ProductOverviewResponse,total=int,limit=int,offset=int}
//	@Failure		400		{object}	object{error=string}
//	@Failure		403		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Security		BearerAuth
//	@Router			/api/v1/tasks/{taskId}/products [get]
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

// CreateStandardProduct godoc
//
//	@Summary		Create Product
//	@Description	Create a new product (initial state DRAFT)
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			body	body		requests.CreateStandardProductRequest true	"Product to create"
//	@Success		201		{object}	responses.ProductResponse
//	@Failure		400		{object}	object{error=string}
//	@Failure		401		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Security		BearerAuth
//	@Router			/api/v1/products/standard [post]
func (h *ProductHandler) CreateStandardProduct(c *gin.Context) {
	var req requests.CreateStandardProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
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

	product, err := h.productService.CreateStandardProduct(&req, creatorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, product)
}

// CreateLimitedProduct godoc
//
//	@Summary		Create a limited product
//	@Description	Create a limited product with stock/availability constraints. Requires authenticated user (creatorID lấy từ context).
//	@Tags			Products
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.CreateLimitedProductRequest	true	"Limited product payload"
//	@Success		201		{object}	map[string]interface{}				"Created product"
//	@Failure		400		{object}	map[string]string					"invalid request / validation failed"
//	@Failure		401		{object}	map[string]string					"missing or invalid user id"
//	@Failure		500		{object}	map[string]string					"internal server error"
//	@Router			/api/v1/products/limited [post]
func (h *ProductHandler) CreateLimitedProduct(c *gin.Context) {

	var req requests.CreateLimitedProductRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
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

	product, err := h.productService.CreateLimitedProduct(&req, creatorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, product)
}

// CreateProductVariant godoc
//
//	@Summary		Create Product Variant
//	@Description	Create a new product variant with story and attributes
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			productId	path		string						true	"Product ID (UUID)"
//
//	@Param			body		body		requests.BulkVariantRequest	true	"Variant data to create"	example({
//	  "price": 29.99,
//	  "current_stock": 100,
//	  "capacity": 500,
//	  "capacity_unit": "ML",
//	  "container_type": "BOTTLE",
//	  "dispenser_type": "SPRAY",
//	  "uses": "For daily use",
//	  "manufacturing_date": "2023-10-01T00:00:00Z",
//	  "expiry_date": "2025-10-01T00:00:00Z",
//	  "instructions": "Shake well before use",
//	  "is_default": true,
//	  "story": {
//	    "content": {
//	      "description": "This is a sample story",
//	      "details": "More details here"
//	    }
//	  },
//	  "attributes": [
//	    {
//	      "attribute_id": "66de757a-6f2b-420c-8aac-2937596e8706",
//	      "value": 10.5,
//	      "unit": "MG"
//	    }
//	  ]
//	})
//
//	@Success		201			{object}	responses.ProductVariantResponse
//	@Failure		400			{object}	object{error=string}
//	@Failure		401			{object}	object{error=string}
//	@Failure		500			{object}	object{error=string}
//	@Security		BearerAuth
//	@Router			/api/v1/products/{productId}/variants [post]
func (h *ProductHandler) CreateProductVariant(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	productID, err := uuid.Parse(c.Param("productId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid product ID: "+err.Error(), http.StatusBadRequest))
		return
	}

	var req requests.BulkVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	uow := h.unitOfWork.Begin()
	defer func() {
		// Ensure rollback on panic; ignore rollback error
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	// 1. Tạo variant
	variant, err := h.productService.CreateProductVariance(ctx, userID, productID, req.CreateProductVariantRequest, uow)
	if err != nil {
		uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse(err.Error(), http.StatusBadRequest))
		return
	}

	// 2. Tạo story
	_, err = h.productService.CreateProductStory(ctx, variant.ID, req.Story, uow)
	if err != nil {
		uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse(err.Error(), http.StatusBadRequest))
		return
	}

	// 3. Thêm attribute values
	attributeIDs := make(map[uuid.UUID]bool)
	for _, attrValue := range req.Attributes {
		if attributeIDs[attrValue.AttributeID] {
			uow.Rollback()
			c.JSON(http.StatusBadRequest, responses.ErrorResponse("AttributeID cannot be duplicate", http.StatusBadRequest))
			return
		}
		attributeIDs[attrValue.AttributeID] = true

		if _, err = h.productService.AddVariantAttributeValue(ctx, variant.ID, attrValue.AttributeID, attrValue, uow); err != nil {
			uow.Rollback()
			c.JSON(http.StatusBadRequest, responses.ErrorResponse(err.Error(), http.StatusBadRequest))
			return
		}
	}

	// 4. Commit transaction
	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Commit error: "+err.Error(), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusCreated, responses.SuccessResponse("Created successful", utils.IntPtr(http.StatusCreated), variant))
}

// CreateVariantImage godoc
//
//	@Summary		Create Variant Image
//	@Description	Upload and create a new image for a product variant
//	@Tags			Products
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			variantId	path		string	true	"Variant ID (UUID)"
//	@Param			file		formData	file	true	"Image file to upload"
//	@Param			alt_text	formData	string	false	"Alt text for the image"
//	@Param			is_primary	formData	bool	false	"Is this the primary image"	default(false)
//	@Success		201			{object}	responses.VariantImageResponse
//	@Failure		400			{object}	object{error=string}
//	@Failure		401			{object}	object{error=string}
//	@Failure		500			{object}	object{error=string}
//	@Security		BearerAuth
//	@Router			/api/v1/products/{productId}/variants/{variantId}/images [post]
func (h *ProductHandler) CreateVariantImage(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	variantID, err := uuid.Parse(c.Param("variantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid variant ID: "+err.Error(), http.StatusBadRequest))
		return
	}

	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("File is required", http.StatusBadRequest))
		return
	}

	// Parse other form fields
	altText := c.PostForm("alt_text")
	isPrimaryStr := c.PostForm("is_primary")
	isPrimary := false
	if isPrimaryStr == "true" {
		isPrimary = true
	}

	// Save uploaded file temporarily
	tempPath := "/tmp/" + file.Filename
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to save uploaded file", http.StatusInternalServerError))
		return
	}
	defer func() { _ = os.Remove(tempPath) }()

	// Upload to S3 and get URL
	imageURL, err := h.fileService.UploadFile(userID.String(), tempPath, file.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to upload file: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// Prepare request
	req := requests.CreateVariantImagesRequest{
		VariantID: variantID,
		ImageURL:  imageURL,
		AltText:   &altText,
		IsPrimary: isPrimary,
	}

	ctx := c.Request.Context()
	uow := h.unitOfWork.Begin()
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	variantImage, err := h.productService.CreateVarianceImage(ctx, variantID, req, uow)
	if err != nil {
		uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to create variant image: "+err.Error(), http.StatusInternalServerError))
		return
	}

	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Commit error: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusCreated, responses.SuccessResponse("Variant image created successfully", utils.IntPtr(http.StatusCreated), variantImage))
}

// CreateVariantAttribute godoc
//
//	@Summary		Create Variant Attribute
//	@Description	Create a new variant attribute
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			body	body		requests.CreateVariantAttributeRequest	true	"Attribute data"
//	@Success		201		{object}	model.VariantAttribute
//	@Failure		400		{object}	object{error=string}
//	@Failure		401		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Security		BearerAuth
//	@Router			/api/v1/variant-attributes [post]
func (h *ProductHandler) CreateVariantAttribute(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	var req requests.CreateVariantAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	uow := h.unitOfWork.Begin()
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	variantAttribute, err := h.productService.CreateVariantAttribute(ctx, userID, req, uow)
	if err != nil {
		uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to create variant attribute: "+err.Error(), http.StatusInternalServerError))
		return
	}

	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Commit error: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusCreated, responses.SuccessResponse("Variant attribute created successfully", utils.IntPtr(http.StatusCreated), variantAttribute))
}

// GetProductDetail godoc
// @Summary      Get Product Detail
// @Description  Retrieve full product detail by ID
// @Tags         Products
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Product ID (UUID)"
// @Success      200  {object}  responses.ProductResponse
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Failure      500  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /api/v1/products/{id} [get]
func (h *ProductHandler) GetProductDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id: " + err.Error()})
		return
	}

	// support both service signatures if available
	var resp *responses.ProductResponse
	// try concrete optional method first
	if svcWithDetail, ok := h.productService.(interface {
		GetProductDetail(uuid.UUID) (*responses.ProductResponse, error)
	}); ok {
		resp, err = svcWithDetail.GetProductDetail(id)
	} else if svcByID, ok := h.productService.(interface {
		GetProductByID(string) (*responses.ProductResponse, error)
	}); ok {
		resp, err = svcByID.GetProductByID(id.String())
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service does not implement required method"})
		return
	}

	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, responses.SuccessResponse("Fetched successful", utils.IntPtr(http.StatusOK), resp))

}
