package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type ProductOptionHandler struct {
	productOptionService iservice.ProductOptionService
	unitOfWork           irepository.UnitOfWork
	validator            *validator.Validate
}

func NewProductOptionHandler(
	productOptionService iservice.ProductOptionService,
	unitOfWork irepository.UnitOfWork,
) *ProductOptionHandler {
	return &ProductOptionHandler{
		productOptionService: productOptionService,
		unitOfWork:           unitOfWork,
		validator:            validator.New(),
	}
}

// GetAll godoc
//
//	@Summary		Get product options
//	@Description	Retrieve product options with optional filtering by type and pagination. Public endpoint.
//	@Tags			ProductOptions
//	@Accept			json
//	@Produce		json
//	@Param			type		query		string															false	"Filter by option type"	Enums(CAPACITY_UNIT, CONTAINER_TYPE, DISPENSER_TYPE, ATTRIBUTE_UNIT)
//	@Param			active_only	query		bool															false	"Filter active options only (default: true)"
//	@Param			page		query		int																false	"Page number (default: 1)"
//	@Param			limit		query		int																false	"Items per page (default: 100, max: 100)"
//	@Success		200			{object}	responses.APIResponse{data=[]responses.ProductOptionResponse}	"Product options retrieved successfully"
//	@Failure		400			{object}	responses.APIResponse											"Invalid request parameters"
//	@Failure		500			{object}	responses.APIResponse											"Internal server error"
//	@Router			/api/v1/product-options [get]
func (h *ProductOptionHandler) GetAll(c *gin.Context) {
	var req requests.ProductOptionFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}

	options, total, err := h.productOptionService.GetAll(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get product options: "+err.Error(), http.StatusInternalServerError))
		return
	}

	pageSize := req.Limit
	if pageSize <= 0 {
		pageSize = 100
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}

	c.JSON(http.StatusOK, responses.NewPaginationResponse(
		"Product options retrieved successfully",
		http.StatusOK,
		options,
		responses.Pagination{
			Page:  page,
			Limit: pageSize,
			Total: total,
		},
	))
}

// GetByType godoc
//
//	@Summary		Get product options by type
//	@Description	Retrieve all active product options for a specific type. Public endpoint with caching.
//	@Tags			ProductOptions
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string															true	"Option type"	Enums(CAPACITY_UNIT, CONTAINER_TYPE, DISPENSER_TYPE, ATTRIBUTE_UNIT)
//	@Success		200		{object}	responses.APIResponse{data=[]responses.ProductOptionResponse}	"Product options retrieved successfully"
//	@Failure		400		{object}	responses.APIResponse											"Invalid option type"
//	@Failure		500		{object}	responses.APIResponse											"Internal server error"
//	@Router			/api/v1/product-options/type/{type} [get]
func (h *ProductOptionHandler) GetByType(c *gin.Context) {
	optionType := c.Param("type")
	if optionType == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Option type is required", http.StatusBadRequest))
		return
	}

	options, err := h.productOptionService.GetByType(c.Request.Context(), optionType)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "invalid option type" {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, responses.ErrorResponse(err.Error(), statusCode))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Product options retrieved successfully", utils.PtrOrNil(http.StatusOK), options))
}

// GetByID godoc
//
//	@Summary		Get product option by ID
//	@Description	Retrieve a specific product option by its ID
//	@Tags			ProductOptions
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string														true	"Product option ID"	format(uuid)
//	@Success		200	{object}	responses.APIResponse{data=responses.ProductOptionResponse}	"Product option retrieved successfully"
//	@Failure		400	{object}	responses.APIResponse										"Invalid product option ID"
//	@Failure		404	{object}	responses.APIResponse										"Product option not found"
//	@Failure		500	{object}	responses.APIResponse										"Internal server error"
//	@Router			/api/v1/product-options/{id} [get]
func (h *ProductOptionHandler) GetByID(c *gin.Context) {
	id, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid product option ID: "+err.Error(), http.StatusBadRequest))
		return
	}

	option, err := h.productOptionService.GetByID(c.Request.Context(), id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "product option not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, responses.ErrorResponse(err.Error(), statusCode))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Product option retrieved successfully", utils.PtrOrNil(http.StatusOK), option))
}

// Create godoc
//
//	@Summary		Create product option
//	@Description	Create a new product option. Admin only.
//	@Tags			ProductOptions
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.CreateProductOptionRequest							true	"Product option data"
//	@Success		201		{object}	responses.APIResponse{data=responses.ProductOptionResponse}	"Product option created successfully"
//	@Failure		400		{object}	responses.APIResponse										"Invalid request or validation error"
//	@Failure		401		{object}	responses.APIResponse										"Unauthorized"
//	@Failure		403		{object}	responses.APIResponse										"Forbidden - Admin only"
//	@Failure		409		{object}	responses.APIResponse										"Product option code already exists"
//	@Failure		500		{object}	responses.APIResponse										"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/product-options [post]
func (h *ProductOptionHandler) Create(c *gin.Context) {
	var req requests.CreateProductOptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request payload: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())

	option, err := h.productOptionService.Create(c.Request.Context(), uow, &req)
	if err != nil {
		uow.Rollback()
		statusCode := http.StatusInternalServerError
		if contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, responses.ErrorResponse(err.Error(), statusCode))
		return
	}

	uow.Commit()
	c.JSON(http.StatusCreated, responses.SuccessResponse("Product option created successfully", utils.PtrOrNil(http.StatusCreated), option))
}

// Update godoc
//
//	@Summary		Update product option
//	@Description	Update an existing product option. Admin only.
//	@Tags			ProductOptions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string														true	"Product option ID"	format(uuid)
//	@Param			data	body		requests.UpdateProductOptionRequest							true	"Product option update data"
//	@Success		200		{object}	responses.APIResponse{data=responses.ProductOptionResponse}	"Product option updated successfully"
//	@Failure		400		{object}	responses.APIResponse										"Invalid request or validation error"
//	@Failure		401		{object}	responses.APIResponse										"Unauthorized"
//	@Failure		403		{object}	responses.APIResponse										"Forbidden - Admin only"
//	@Failure		404		{object}	responses.APIResponse										"Product option not found"
//	@Failure		409		{object}	responses.APIResponse										"Product option code already exists"
//	@Failure		500		{object}	responses.APIResponse										"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/product-options/{id} [patch]
func (h *ProductOptionHandler) Update(c *gin.Context) {
	id, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid product option ID: "+err.Error(), http.StatusBadRequest))
		return
	}

	var req requests.UpdateProductOptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request payload: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())

	option, err := h.productOptionService.Update(c.Request.Context(), uow, id, &req)
	if err != nil {
		uow.Rollback()
		statusCode := http.StatusInternalServerError
		if err.Error() == "product option not found" {
			statusCode = http.StatusNotFound
		} else if contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, responses.ErrorResponse(err.Error(), statusCode))
		return
	}

	uow.Commit()
	c.JSON(http.StatusOK, responses.SuccessResponse("Product option updated successfully", utils.PtrOrNil(http.StatusOK), option))
}

// Delete godoc
//
//	@Summary		Delete product option
//	@Description	Soft delete a product option. Admin only.
//	@Tags			ProductOptions
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"Product option ID"	format(uuid)
//	@Success		200	{object}	responses.APIResponse	"Product option deleted successfully"
//	@Failure		400	{object}	responses.APIResponse	"Invalid product option ID"
//	@Failure		401	{object}	responses.APIResponse	"Unauthorized"
//	@Failure		403	{object}	responses.APIResponse	"Forbidden - Admin only"
//	@Failure		404	{object}	responses.APIResponse	"Product option not found"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/product-options/{id} [delete]
func (h *ProductOptionHandler) Delete(c *gin.Context) {
	id, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid product option ID: "+err.Error(), http.StatusBadRequest))
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())

	err = h.productOptionService.Delete(c.Request.Context(), uow, id)
	if err != nil {
		uow.Rollback()
		statusCode := http.StatusInternalServerError
		if err.Error() == "product option not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, responses.ErrorResponse(err.Error(), statusCode))
		return
	}

	uow.Commit()
	c.JSON(http.StatusOK, responses.SuccessResponse("Product option deleted successfully", utils.PtrOrNil(http.StatusOK), nil))
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
