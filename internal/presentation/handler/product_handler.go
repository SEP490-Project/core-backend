package handler

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"go.uber.org/zap"

	"github.com/aws/smithy-go/ptr"

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
	// create validator and register custom datetime validation
	v := validator.New()
	_ = v.RegisterValidation("datetime", func(fl validator.FieldLevel) bool {
		param := fl.Param()
		// if no param provided, default to RFC3339
		if strings.TrimSpace(param) == "" {
			param = time.RFC3339
		}
		// get string value, support *string and string
		var s string
		fv := fl.Field()
		switch fv.Kind() {
		case reflect.Pointer:
			if fv.IsNil() {
				return true // let 'required' handle emptiness
			}
			s = fv.Elem().String()
		default:
			s = fv.String()
		}
		if s == "" {
			return true
		}
		// support multiple layouts separated by '|'
		layouts := strings.SplitSeq(param, "|")
		for layout := range layouts {
			layout = strings.TrimSpace(layout)
			if layout == "" {
				continue
			}
			if _, err := time.Parse(layout, s); err == nil {
				return true
			}
		}
		return false
	})

	return &ProductHandler{
		productService: productService,
		fileService:    fileService,
		unitOfWork:     unitOfWork,
		validator:      v,
	}
}

// GetAllProducts godoc
//
//	@Deprecated
//	@Summary		Get All Products
//	@Description	Get paginated list of products with optional search
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			limit		query		int																		false	"Number of items per page"	default(10)
//	@Param			page		query		int																		false	"Number of items to skip"	default(1)
//	@Param			search		query		string																	false	"Search term for product name"
//	@Param			category_id	query		string																	false	"Filter category of products"
//	@Param			type		query		string																	false	"Filter type of products"
//	@Success		200			{object}	object{data=[]responses.ProductResponse,total=int,limit=int,offset=int}	"Products retrieved successfully"
//	@Failure		500			{object}	object{error=string}													"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/products [get]
func (h *ProductHandler) GetAllProducts(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	search := c.DefaultQuery("search", "")
	category := c.DefaultQuery("category_id", "")
	prdType := c.DefaultQuery("type", "")

	products, total, err := h.productService.GetProductsPagination(page, limit, search, category, prdType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := int(total) / limit
	if total%limit != 0 {
		totalPages++
	}

	pagination := responses.Pagination{
		Page:       page,
		Limit:      limit,
		Total:      int64(total),
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	// --- Response  ---
	response := responses.NewPaginationResponse(
		"Products retrieved successfully",
		http.StatusOK,
		products,
		pagination,
	)
	c.JSON(http.StatusOK, response)
}

// GetAllProductsV2 godoc
//
//	@Summary		Get All Products ONLY TO ADMIN/SALES_STAFF. Other will viewed as Partial
//	@Description	Get paginated list of products with optional search
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			limit			query		int																			false	"Number of items per page"	default(10)
//	@Param			page			query		int																			false	"Number of items to skip"	default(1)
//	@Param			search			query		string																		false	"Search term for product name"
//	@Param			category_id		query		string																		false	"Filter category of products"
//	@Param			brand_id		query		string																		false	"Filter products by brand"
//	@Param			user_id			query		string																		false	"Filter products by brand user"
//	@Param			type			query		string																		false	"Filter type of products"	Enums(STANDARD, LIMITED)
//	@Param			status			query		string																		false	"Filter status of products"	Enums(DRAFT, SUBMITTED, REVISION, APPROVED, ACTIVED, INACTIVED)
//	@Param			filterPreOrder	query		boolean																		false	"Filter status of products"	false	"Find All PreOrder Products Only"
//	@Success		200				{object}	object{data=[]responses.ProductResponseV2,total=int,limit=int,offset=int}	"Products view for Others"
//
//	@Failure		500				{object}	object{error=string}														"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/products/v2 [get]
func (h *ProductHandler) GetAllProductsV2(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	search := c.DefaultQuery("search", "")
	category := c.DefaultQuery("category_id", "")
	prdType := c.DefaultQuery("type", "")
	prdStatusesParam := c.DefaultQuery("status", "")
	filterPreOrder := c.DefaultQuery("filterPreOrder", "false")
	isPreOrderOnly := strings.ToLower(filterPreOrder) == "true"

	//if type != nil, we have to validate it
	if prdType != "" {
		validTypes := map[string]bool{
			string(enum.ProductTypeStandard): true,
			string(enum.ProductTypeLimited):  true,
		}
		if !validTypes[prdType] {
			resp := responses.ErrorResponse("invalid product type filter", http.StatusBadRequest)
			c.JSON(http.StatusBadRequest, resp)
			return
		}
	}

	allowFullViewRoles := []enum.UserRole{enum.UserRoleAdmin, enum.UserRoleSalesStaff}

	// parse status list (comma separated)
	var prdStatuses []string
	if strings.TrimSpace(prdStatusesParam) != "" {
		parts := strings.Split(prdStatusesParam, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				prdStatuses = append(prdStatuses, p)
			}
		}
	}

	if len(prdStatuses) > 0 && IsAllowRole(c, allowFullViewRoles) {
		validStatuses := map[string]bool{
			string(enum.ProductStatusDraft):     true,
			string(enum.ProductStatusSubmitted): true,
			string(enum.ProductStatusRevision):  true,
			string(enum.ProductStatusActived):   true,
			string(enum.ProductStatusInactived): true,
		}
		for _, s := range prdStatuses {
			if !validStatuses[s] {
				resp := responses.ErrorResponse("invalid product status filter", http.StatusBadRequest)
				c.JSON(http.StatusBadRequest, resp)
				return
			}
		}
	}

	var (
		products any
		total    int
		svcErr   error
	)

	allowFullViewRoles = []enum.UserRole{enum.UserRoleAdmin, enum.UserRoleSalesStaff}
	if IsAllowRole(c, allowFullViewRoles) {
		var res []responses.ProductResponseV2
		res, total, svcErr = h.productService.GetProductsPaginationV2(page, limit, search, category, prdType, prdStatuses, isPreOrderOnly)
		products = res
	} else {
		var res []responses.ProductResponseV2Partial
		res, total, svcErr = h.productService.GetProductsPaginationV2Partial(page, limit, search, category, prdType, isPreOrderOnly)
		products = res
	}

	if svcErr != nil {
		resp := responses.ErrorResponse(svcErr.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	totalPages := int(total) / limit
	if total%limit != 0 {
		totalPages++
	}

	pagination := responses.Pagination{
		Page:       page,
		Limit:      limit,
		Total:      int64(total),
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	// --- Response  ---
	if IsAllowRole(c, allowFullViewRoles) {
		data, ok := products.([]responses.ProductResponseV2)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal type assertion error"})
			return
		}
		resp := responses.NewPaginationResponse(
			"Products retrieved successfully",
			http.StatusOK,
			data,
			pagination,
		)
		c.JSON(http.StatusOK, resp)
		return
	} else {
		dataPartial, ok := products.([]responses.ProductResponseV2Partial)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal type assertion error"})
			return
		}
		resp := responses.NewPaginationResponse(
			"Products retrieved successfully",
			http.StatusOK,
			dataPartial,
			pagination,
		)
		c.JSON(http.StatusOK, resp)
	}
}

// GetProductsByTask godoc
//
//	@Summary		Get Products By Task
//	@Description	Get paginated products (overview) belonging to a task. Authorization: staff roles or owning brand user.
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			task_id	path		string	true	"Task ID (UUID)"
//	@Param			limit	query		int		false	"Number of items per page"	default(10)
//	@Param			offset	query		int		false	"Number of items to skip"	default(0)
//	@Success		200		{object}	object{data=[]responses.ProductOverviewResponse,total=int,limit=int,offset=int}
//	@Failure		400		{object}	object{error=string}
//	@Failure		403		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Security		BearerAuth
//	@Router			/api/v1/tasks/{task_id}/products [get]
func (h *ProductHandler) GetProductsByTask(c *gin.Context) {
	// Parse path param
	taskIDStr := c.Param("task_id")
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
//	@Tags			Products.Standard
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.CreateStandardProductRequest	true	"Product to create"
//	@Success		201		{object}	responses.ProductResponseV2
//	@Failure		400		{object}	object{error=string}
//	@Failure		401		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Security		BearerAuth
//	@Router			/api/v1/products/standard [post]
func (h *ProductHandler) CreateStandardProduct(c *gin.Context) {
	var req requests.CreateStandardProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		response := processValidationError(err)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	uidVal, err := extractUserID(c)
	if err != nil {
		response := responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	product, err := h.productService.CreateStandardProduct(&req, uidVal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := responses.SuccessResponse("Create standard product successfully", ptr.Int(http.StatusCreated), product)
	c.JSON(http.StatusCreated, resp)
}

// CreateLimitedProduct godoc
//
//	@Summary		Create a limited product
//	@Description	Create a limited product with stock/availability constraints. Requires authenticated user (creatorID lấy từ context).
//	@Tags			Products.Limited
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.CreateLimitedProductRequest	true	"Limited product payload"
//	@Success		201		{object}	responses.ProductResponseV2				"Created limited product"
//	@Failure		400		{object}	map[string]string						"invalid request / validation failed"
//	@Failure		401		{object}	map[string]string						"missing or invalid user id"
//	@Failure		500		{object}	map[string]string						"internal server error"
//	@Router			/api/v1/products/limited [post]
func (h *ProductHandler) CreateLimitedProduct(c *gin.Context) {

	var req requests.CreateLimitedProductRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		response := processValidationError(err)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := h.createLimitedProductValidation(req); err != nil {
		response := responses.ErrorResponse("Validation error: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	creatorID, err := extractUserID(c)
	if err != nil {
		resp := responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, resp)
		return
	}

	product, err := h.productService.CreateLimitedProduct(&req, creatorID)
	if err != nil {
		response := responses.ErrorResponse("Failed to create Limited Product: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	resp := responses.SuccessResponse("Create limited product successfully", ptr.Int(http.StatusCreated), product)
	c.JSON(http.StatusCreated, resp)
}

func (h *ProductHandler) createLimitedProductValidation(req requests.CreateLimitedProductRequest) error {
	// Validate limited product date fields against now and relative ordering
	var (
		premiereDateStr          = req.LimitedAttribute.PremiereDate
		availabilityStartDateStr = req.LimitedAttribute.AvailabilityStartDate
		availabilityEndDateStr   = req.LimitedAttribute.AvailabilityEndDate
	)

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04-07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}

	parse := func(s *string) (time.Time, error) {
		if s == nil || strings.TrimSpace(*s) == "" {
			return time.Time{}, nil
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, *s); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("invalid datetime format: %s", *s)
	}

	now := time.Now()

	premiereDate, err := parse(premiereDateStr)
	if err != nil {
		return err
	}
	if (premiereDate != time.Time{}) && premiereDate.Before(now) {
		return fmt.Errorf("premiere_date must not be in the past")
	}

	availabilityStart, err := parse(availabilityStartDateStr)
	if err != nil {
		return err
	}
	if (availabilityStart != time.Time{}) && availabilityStart.Before(now) {
		return fmt.Errorf("availability_start_date must not be in the past")
	}

	availabilityEnd, err := parse(availabilityEndDateStr)
	if err != nil {
		return err
	}
	if (availabilityEnd != time.Time{}) && availabilityEnd.Before(now) {
		return fmt.Errorf("availability_end_date must not be in the past")
	}

	// Ordering validations
	if !premiereDate.IsZero() && !availabilityStart.IsZero() && availabilityStart.Before(premiereDate) {
		return fmt.Errorf("availability_start_date must be equal or after premiere_date")
	}
	if !availabilityStart.IsZero() && !availabilityEnd.IsZero() && availabilityEnd.Before(availabilityStart) {
		return fmt.Errorf("availability_end_date must be equal or after availability_start_date")
	}

	return nil
}

// CreateProductVariant godoc
//
//	@Summary		Create Product Variant
//	@Description	Create a new product variant with story and attributes
//	@Tags			Products.Variants
//	@Accept			json
//	@Produce		json
//	@Param			productId	path		string						true	"Product ID (UUID)"
//	@Param			data		body		requests.BulkVariantRequest	true	"Variant data to create"
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
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err = h.validator.Struct(&req); err != nil {
		response := processValidationError(err)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	ctx := c.Request.Context()
	uow := h.unitOfWork.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			_ = uow.Rollback()
			panic(r)
		}
	}()

	// 1. Tạo variant
	variant, err := h.productService.CreateProductVariance(ctx, userID, productID, req.CreateProductVariantRequest, uow)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse(err.Error(), http.StatusBadRequest))
		return
	}

	// 2. Tạo story
	var story *model.ProductStory
	if req.Story != nil {
		story, err = h.productService.CreateProductStory(ctx, variant.ID, *req.Story, uow)
		if err != nil {
			_ = uow.Rollback()
			c.JSON(http.StatusBadRequest, responses.ErrorResponse(err.Error(), http.StatusBadRequest))
			return
		}
	}

	// 3. Thêm attribute values
	attributeIDs := make(map[uuid.UUID]bool)
	var attributeValueList []responses.ProductAttributesResponse
	for _, attrValue := range req.Attributes {
		if attributeIDs[attrValue.AttributeID] {
			_ = uow.Rollback()
			c.JSON(http.StatusBadRequest, responses.ErrorResponse("AttributeID cannot be duplicate", http.StatusBadRequest))
			return
		}
		attributeIDs[attrValue.AttributeID] = true

		attributeValue, err := h.productService.AddVariantAttributeValue(ctx, variant.ID, attrValue.AttributeID, attrValue, uow)
		if err != nil {
			_ = uow.Rollback()
			c.JSON(http.StatusBadRequest, responses.ErrorResponse(err.Error(), http.StatusBadRequest))
			return
		}
		attributeValueList = append(attributeValueList,
			responses.ProductAttributesResponse{
				Ingredient:  attributeValue.Attribute.Ingredient,
				Value:       attributeValue.Value,
				Unit:        attributeValue.Unit,
				Description: attributeValue.Attribute.Description,
			})
	}

	// 4. Commit transaction
	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Commit error: "+err.Error(), http.StatusBadRequest))
		return
	}

	formatedVariant := responses.ProductVariantResponse{}.ToFullProductVariantResponse(variant, story, attributeValueList)

	if variant.Product != nil {
		formatedVariant.Name = variant.Product.Name
		formatedVariant.Description = variant.Product.Description
		formatedVariant.Type = variant.Product.Type
	}

	// return concrete value (not pointer) to ensure JSON body matches expected shape
	varResp := *formatedVariant

	c.JSON(http.StatusCreated, responses.SuccessResponse("Created successful", utils.IntPtr(http.StatusCreated), varResp))
}

// CreateVariantImage godoc
//
//	@Summary		Create Variant Image
//	@Description	Upload and create a new image for a product variant
//	@Tags			Products.Variants
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
//	@Router			/api/v1/products/variants/{variantId}/images [post]
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
	isPrimary := c.PostForm("is_primary") == "true"

	// Generate timestamp-based filename
	timestamp := time.Now().Format("20060102_150405")
	newFileName := fmt.Sprintf("%s_%s", timestamp, file.Filename)

	// 🔹 Create per-user tmp directory
	userTmpDir := fmt.Sprintf("./tmp/%s", userID)
	if err = os.MkdirAll(userTmpDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to create user tmp directory", http.StatusInternalServerError))
		return
	}

	finalPath := fmt.Sprintf("%s/%s", userTmpDir, newFileName)

	// 🔹 Predict image URL before upload
	key := fmt.Sprintf("%s/%s", userID, newFileName)
	predictedImgURL := h.productService.BuildFileURL(key)

	// 🔹 Save uploaded file (with cleanup on error)
	if err = c.SaveUploadedFile(file, finalPath); err != nil {
		_ = os.Remove(finalPath) // cleanup if save failed
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to save uploaded file", http.StatusInternalServerError))
		return
	}

	// Prepare request to create DB record
	req := requests.CreateVariantImagesRequest{
		VariantID: variantID,
		ImageURL:  predictedImgURL,
		AltText:   &altText,
		IsPrimary: isPrimary,
	}

	ctx := c.Request.Context()
	uow := h.unitOfWork.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			_ = uow.Rollback()
			_ = os.Remove(finalPath) // cleanup file on panic
			panic(r)
		}
	}()

	variantImage, err := h.productService.CreateVarianceImage(ctx, variantID, req, uow)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to create variant image: "+err.Error(), http.StatusInternalServerError))
		return
	}

	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to commit transaction", http.StatusInternalServerError))
		return
	}

	// Run async upload
	go func(filePath, fileName string, variantImageID uuid.UUID, userID uuid.UUID) {
		zap.L().Info("Start async")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

		uow := h.unitOfWork.Begin(ctx)
		defer cancel()
		defer os.Remove(filePath)

		if _, err := h.productService.UpdateVariantImageAsync(ctx, userID, variantImageID, &filePath, requests.UpdateVariantImagesRequest{}, uow); err != nil {
			zap.L().Error("async update variant image failed",
				zap.String("variantImageID", variantImageID.String()),
				zap.Error(err),
			)
			_ = uow.Rollback()
			return
		}

		if err := uow.Commit(); err != nil {
			zap.L().Error("failed to commit async update", zap.Error(err))
		}

	}(finalPath, newFileName, variantImage.ID, userID)

	c.JSON(http.StatusCreated, responses.SuccessResponse(
		"Variant image created successfully (uploading asynchronously)",
		utils.IntPtr(http.StatusCreated),
		variantImage,
	))
}

// CreateVariantAttribute godoc
//
//	@Summary		Create Variant Attribute
//	@Description	Create a new variant attribute
//	@Tags			Products.Variants.Attributes
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.CreateVariantAttributeRequest	true	"Attribute data"
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
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err = h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	uow := h.unitOfWork.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			_ = uow.Rollback()
			panic(r)
		}
	}()

	variantAttribute, err := h.productService.CreateVariantAttribute(ctx, userID, req, uow)
	if err != nil {
		_ = uow.Rollback()
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
//
//	@Summary		Get Product Detail
//	@Description	Retrieve full product detail by ID
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Product ID (UUID)"
//	@Success		200	{object}	responses.ProductResponse
//	@Failure		400	{object}	object{error=string}
//	@Failure		404	{object}	object{error=string}
//	@Failure		500	{object}	object{error=string}
//	@Security		BearerAuth
//	@Router			/api/v1/products/{id} [get]
func (h *ProductHandler) GetProductDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		resp := responses.ErrorResponse("invalid product id: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	resp, err := h.productService.GetProductDetail(id)
	if err != nil {
		resp := responses.ErrorResponse(err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	c.JSON(http.StatusOK, responses.SuccessResponse("Fetched successful", utils.IntPtr(http.StatusOK), resp))

}

// AddConceptToLimitedProduct godoc
//
//	@Summary		Add Concept to Limited Product
//	@Description	Associate an existing concept to a limited product
//	@Tags			Products.Limited
//	@Accept			json
//	@Produce		json
//	@Param			limited-id	path		string	true	"Limited Product ID (UUID)"
//	@Param			concept-id	path		string	false	"Concept ID (UUID)"
//	@Success		200			{object}	map[string]any
//	@Failure		400			{object}	object{error=string}
//	@Failure		401			{object}	object{error=string}
//	@Failure		500			{object}	object{error=string}
//	@Security		BearerAuth
//	@Router			/api/v1/products/limited/{limited-id}/concept/{concept-id} [post]
func (h *ProductHandler) AddConceptToLimitedProduct(c *gin.Context) {
	limitedIDStr := c.Param("limited-id")
	limitedID, err := uuid.Parse(limitedIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid limited product id: "+err.Error(), http.StatusBadRequest))
		return
	}

	conceptIDStr := c.Param("concept-id")
	conceptID, err := uuid.Parse(conceptIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid concept id: "+err.Error(), http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	uow := h.unitOfWork.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			_ = uow.Rollback()
			panic(r)
		}
	}()

	res, err := h.productService.AddConceptToLimitedProduct(ctx, limitedID, conceptID, uow)
	if err != nil {
		_ = uow.Rollback()
		// If service returned not found-like errors, map to 400/404 accordingly
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			c.JSON(http.StatusBadRequest, responses.ErrorResponse(err.Error(), http.StatusBadRequest))
			return
		}
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("commit error: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Concept associated successfully", utils.IntPtr(http.StatusOK), res))
}

// GetVariantAttributePagination godoc
//
//	@Summary		List Variant Attributes (Public)
//	@Description	Get paginated list of variant attributes (public view). Returns lightweight attribute responses suitable for front-end listing.
//	@Tags			Products.Variants.Attributes
//	@Accept			json
//	@Produce		json
//	@Param			page	query		int		false	"Page number"		default(1)
//	@Param			limit	query		int		false	"Items per page"	default(10)
//	@Param			search	query		string	false	"Search term for name"
//	@Success		200		{object}	responses.APIResponse{data=[]responses.VariantAttributeResponse,pagination=responses.Pagination}
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/variant-attributes [get]
func (h *ProductHandler) GetVariantAttributePagination(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	search := c.DefaultQuery("search", "")

	// Fetch variant attributes according to role (service returns model.VariantAttribute)
	var (
		attrs  any
		total  int
		svcErr error
	)

	allowFullViewRoles := []enum.UserRole{enum.UserRoleAdmin, enum.UserRoleSalesStaff}
	if IsAllowRole(c, allowFullViewRoles) {
		attrs, total, svcErr = h.productService.GetVariantAttributePagination(page, limit, search)
	} else {
		attrs, total, svcErr = h.productService.GetVariantAttributePaginationPartial(page, limit, search)
	}

	if svcErr != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Error fetching variant attributes: "+svcErr.Error(), http.StatusInternalServerError))
		return
	}

	data := make([]any, 0)
	switch v := attrs.(type) {
	case []model.VariantAttribute:
		for i := range v {
			data = append(data, v[i])
		}
	case []responses.VariantAttributeResponse:
		for i := range v {
			data = append(data, v[i])
		}
	default:
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Unexpected type", http.StatusInternalServerError))
		return
	}

	totalPages := int(total) / limit
	if total%limit != 0 {
		totalPages++
	}

	pagination := responses.Pagination{
		Page:       page,
		Limit:      limit,
		Total:      int64(total),
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	// --- Response  ---
	resp := responses.NewPaginationResponse(
		"Variant attributes retrieved successfully",
		http.StatusOK,
		data,
		pagination,
	)
	c.JSON(http.StatusOK, resp)
}

// PublishProduct godoc
//
//	@Summary		Publish or unpublish a product
//	@Description	Toggle a product's active status (publish/unpublish). Requires Sales, Brand, or Admin role.
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string	true	"Product ID (UUID)"
//	@Param			is-active	path		boolean	true	"Publish payload"
//	@Success		200			{object}	responses.ProductResponseV2
//	@Failure		400			{object}	object{error=string}
//	@Failure		401			{object}	object{error=string}
//	@Failure		404			{object}	object{error=string}
//	@Failure		500			{object}	object{error=string}
//	@Security		BearerAuth
//	@Router			/api/v1/products/publish/{id}/{is-active} [patch]
func (h *ProductHandler) PublishProduct(c *gin.Context) {
	idParam := c.Param("id")
	activeParam := c.Param("is-active")

	productID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	isActive, err := strconv.ParseBool(activeParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid is-active value"})
		return
	}

	prd, svcErr := h.productService.PublishProduct(productID, isActive)
	if svcErr != nil {
		if errors.Is(svcErr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		zap.L().Error("publish product failed", zap.String("product_id", productID.String()), zap.Error(svcErr))
		resp := responses.ErrorResponse(svcErr.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	resp := responses.SuccessResponse("Product publish status updated successfully", ptr.Int(http.StatusOK), prd)
	c.JSON(http.StatusOK, resp)
}
