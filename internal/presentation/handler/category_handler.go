package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"
	"strconv"

	"github.com/aws/smithy-go/ptr"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ProductCategoryHandler struct {
	categoryService iservice.ProductCategoryService
	unitOfWork      irepository.UnitOfWork
	validator       *validator.Validate
}

func NewCategoryHandler(
	categoryService iservice.ProductCategoryService,
	unitOfWork irepository.UnitOfWork,
) *ProductCategoryHandler {
	return &ProductCategoryHandler{
		categoryService: categoryService,
		validator:       validator.New(),
		unitOfWork:      unitOfWork,
	}
}

// GetAllCategories godoc
//
//	@Summary		Get product categories
//	@Description	Returns a paginated list of product categories. Optional search by name.
//	@Tags			Categories
//	@Accept			json
//	@Produce		json
//	@Param			page	query		int		false	"Page number"		default(1)
//	@Param			limit	query		int		false	"Items per page"	default(10)
//	@Param			search	query		string	false	"Search term for category name"
//	@Param			deleted	query		bool	false	"Include deleted categories (null - get all, true - only deleted, false - only active)"	default(false)
//	@Success		200		{array}		responses.ProductCategoryResponse
//	@Failure		500		{object}	map[string]string
//	@Router			/api/v1/categories [get]
func (h *ProductCategoryHandler) GetAllCategories(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	search := c.DefaultQuery("search", "")
	deleted := c.DefaultQuery("deleted", "")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// --- Call service ---
	categories, total, err := h.categoryService.GetAllCategories(page, limit, search, deleted)
	if err != nil {
		zap.L().Error("Failed to get categories", zap.Error(err))
		resp := responses.ErrorResponse("Failed to get categories: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	// --- Pagination info ---
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	pagination := responses.Pagination{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	// --- Success response ---
	resp := responses.NewPaginationResponse(
		"Categories retrieved successfully",
		http.StatusOK,
		categories,
		pagination,
	)
	c.JSON(http.StatusOK, resp)
}

// CreateCategory godoc
//
//	@Summary		Create a product category
//	@Description	Create a new product category
//	@Tags			Categories
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.CreateProductCategoryRequest	true	"Create category payload"
//	@Success		201		{object}	responses.ProductCategoryResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/api/v1/categories [post]
func (h *ProductCategoryHandler) CreateCategory(c *gin.Context) {
	var req requests.CreateProductCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errResp := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, errResp)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		response := processValidationError(err)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	category, err := h.categoryService.CreateCategory(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to create category: "+err.Error(), http.StatusBadRequest))
		return
	}

	resp := responses.SuccessResponse("Category created successfully", ptr.Int(http.StatusOK), category)
	c.JSON(http.StatusCreated, resp)
}

// AssignParentCategory godoc
//
//	@Summary		Assign parent category
//	@Description	Set a parent category for an existing category. If parent_id is not provided, the parent category will be removed.
//	@Tags			Categories
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string	true	"Category ID"
//	@Param			parent_id	query		string	false	"Parent category ID"
//	@Success		200			{object}	responses.ProductCategoryResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/api/v1/categories/{id}/parent [patch]
func (h *ProductCategoryHandler) AssignParentCategory(c *gin.Context) {
	currentID := c.Param("id")
	parentID := c.Query("parent_id")
	curUUID, err := uuid.Parse(currentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid categoryID: "+err.Error(), http.StatusBadRequest))
		return
	}

	var parentUUID uuid.UUID
	if parentID != "" {
		var err error
		parentUUID, err = uuid.Parse(parentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid category's parentID: "+err.Error(), http.StatusBadRequest))
			return
		}
	}

	category, err := h.categoryService.AddParentCategory(curUUID, parentUUID)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to add parent category: "+err.Error(), http.StatusBadRequest))
		return
	}

	resp := responses.SuccessResponse("Add parent to category successfully", ptr.Int(http.StatusOK), category)
	c.JSON(http.StatusOK, resp)
}

// DeleteCategory godoc
//
//	@Summary		Delete a product category
//	@Description	Delete category by id
//	@Tags			Categories
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Category ID"
//	@Success		204	{string}	string	""
//	@Failure		400	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/api/v1/categories/{id} [patch]
func (h *ProductCategoryHandler) DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	cateUUID, err := uuid.Parse(id)
	if err != nil {
		response := responses.ErrorResponse("Invalid product category id: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	ctx := c.Request.Context()
	uow := h.unitOfWork.Begin()
	err = h.categoryService.DeleteCategory(ctx, cateUUID, uow)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to delete category: "+err.Error(), http.StatusBadRequest))
		return
	}

	_ = uow.Commit()
	response := responses.SuccessResponse("Category deleted successfully", ptr.Int(http.StatusOK), http.StatusNoContent)
	c.JSON(http.StatusOK, response)
}
