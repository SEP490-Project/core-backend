package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type BrandHandler struct {
	BrandService iservice.BrandService
	UnitOfWork   irepository.UnitOfWork
	Validator    *validator.Validate
}

func NewBrandHandler(brandService iservice.BrandService, unitOfWork irepository.UnitOfWork) *BrandHandler {
	return &BrandHandler{
		BrandService: brandService,
		UnitOfWork:   unitOfWork,
		Validator:    validator.New(),
	}
}

// CreateBrand godoc
// @Summary      Create Brand
// @Description  Create a new brand
// @Tags         Brands
// @Accept       json
// @Produce      json
// @Param        request body requests.CreateBrandRequest true "Brand creation data"
// @Success      201 {object} responses.APIResponse{data=responses.BrandResponse} "Brand created successfully"
// @Failure      400 {object} responses.APIResponse "Invalid request"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/brands [post]
func (bh *BrandHandler) CreateBrand(c *gin.Context) {
	var req requests.CreateBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := bh.Validator.Struct(&req); err != nil {
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	brand, err := bh.BrandService.CreateBrand(c.Request.Context(), &req)
	if err != nil {
		response := responses.ErrorResponse("Failed to create brand: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	c.JSON(http.StatusCreated, responses.SuccessResponse("Brand created successfully", utils.IntPtr(http.StatusCreated), brand))
}

// CreateBrandWithInActiveUsers godoc
// @Summary      Create Brand with Inactive Users
// @Description  Create a new brand with the associated inactive users
// @Tags         Brands
// @Accept       json
// @Produce      json
// @Param        request body requests.CreateBrandRequest true "Brand creation data"
// @Success      201 {object} responses.APIResponse{data=responses.BrandResponse} "Brand and User created successfully"
// @Failure      400 {object} responses.APIResponse "Invalid request"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/brands/with-users [post]
func (bh *BrandHandler) CreateBrandWithInActiveUsers(c *gin.Context) {
	var req requests.CreateBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := bh.Validator.Struct(&req); err != nil {
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	transactionalUOW := bh.UnitOfWork.Begin()

	defer func() {
		if r := recover(); r != nil {
			transactionalUOW.Rollback()
			zap.L().Error("Transaction panicked, rolled back", zap.Any("recover", r))
			panic(r)
		}
	}()

	brand, err := bh.BrandService.CreateBrandWithInActiveUsers(c.Request.Context(), &transactionalUOW, &req)
	if err != nil {
		transactionalUOW.Rollback()
		response := responses.ErrorResponse("Failed to create brand: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	transactionalUOW.Commit()
	c.JSON(http.StatusCreated, responses.SuccessResponse("Brand and User created successfully", utils.IntPtr(http.StatusCreated), brand))
}

// GetBrandByID godoc
// @Summary      Get Brand by ID
// @Description  Get brand details by ID
// @Tags         Brands
// @Accept       json
// @Produce      json
// @Param        id path string true "Brand ID"
// @Success      200 {object} responses.APIResponse{data=responses.BrandResponse} "Brand fetched successfully"
// @Failure      400 {object} responses.APIResponse "Invalid brand ID"
// @Failure      404 {object} responses.APIResponse "Brand not found"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/brands/{id} [get]
func (bh *BrandHandler) GetBrandByID(c *gin.Context) {
	brandIDStr := c.Param("id")
	brandID, err := uuid.Parse(brandIDStr)
	if err != nil {
		response := responses.ErrorResponse("Invalid brand ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	brand, err := bh.BrandService.GetByID(c.Request.Context(), brandID)
	if err != nil {
		response := responses.ErrorResponse("Failed to get brand: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	c.JSON(http.StatusOK, responses.SuccessResponse("Brand fetched successfully", nil, brand))
}

// GetBrandsByFilter godoc
// @Summary      Get Brands List by filter
// @Description  Get paginated list of brands with optional filters
// @Tags         Brands
// @Accept       json
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(10)
// @Param        keywords query string false "Search keywords for brand name"
// @Param        status query string false "Filter by brand status" Enums(ACTIVE, INACTIVE)
// @Success      200 {object} responses.BrandPaginationResponse "Brands fetched successfully"
// @Failure      400 {object} responses.APIResponse "Invalid request"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/brands [get]
func (bh *BrandHandler) GetBrandsByFilter(c *gin.Context) {
	request := requests.ListBrandsRequest{}
	if err := c.ShouldBindQuery(&request); err != nil {
		response := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}
	if err := bh.Validator.Struct(&request); err != nil {
		responses := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	brands, totalCount, err := bh.BrandService.GetByFilter(c.Request.Context(), &request)
	if err != nil {
		response := responses.ErrorResponse("Failed to get brands: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	} else if totalCount == 0 || len(brands) == 0 {
		responses := responses.EmptyPaginationResponse[responses.BrandResponse]("No brands found matching the filter criteria", nil, request.Page, request.Limit)
		c.JSON(http.StatusOK, responses)
		return
	}

	totalPages := int((totalCount + int64(request.Limit) - 1) / int64(request.Limit))
	paginationResponse := responses.PaginatedResponse("Successfully fetched brands", http.StatusOK, brands, responses.Pagination{
		Page:       request.Page,
		Limit:      request.Limit,
		Total:      totalCount,
		TotalPages: totalPages,
		HasNext:    request.Page < totalPages,
		HasPrev:    request.Page > 1,
	})
	c.JSON(http.StatusOK, paginationResponse)
}

// UpdateBrand godoc
// @Summary      Update Brand
// @Description  Update brand details by ID
// @Tags         Brands
// @Accept       json
// @Produce      json
// @Param        id path string true "Brand ID"
// @Param        request body requests.UpdateBrandRequest true "Brand update data"
// @Success      200 {object} responses.APIResponse{data=responses.BrandResponse} "Brand updated successfully"
// @Failure      400 {object} responses.APIResponse "Invalid request"
// @Failure      404 {object} responses.APIResponse "Brand not found"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/brands/{id} [put]
func (bh *BrandHandler) UpdateBrand(c *gin.Context) {
	brandIDStr := c.Param("id")
	var req requests.UpdateBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}
	if err := bh.Validator.Struct(&req); err != nil {
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	brandID, err := uuid.Parse(brandIDStr)
	if err != nil {
		response := responses.ErrorResponse("Invalid brand ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	brand, err := bh.BrandService.UpdateBrand(c.Request.Context(), brandID, &req)
	if err != nil {
		response := responses.ErrorResponse("Failed to update brand: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	c.JSON(http.StatusOK, responses.SuccessResponse("Brand updated successfully", nil, brand))
}

// UpdateBrandStatus godoc
// @Summary      Update Brand Status
// @Description  Update brand status (ACTIVE/INACTIVE)
// @Tags         Brands
// @Accept       json
// @Produce      json
// @Param        id path string true "Brand ID"
// @Param        status path string true "Brand Status" Enums(ACTIVE, INACTIVE)
// @Success      200 {object} responses.APIResponse{data=responses.BrandResponse} "Brand status updated successfully"
// @Failure      400 {object} responses.APIResponse "Invalid request"
// @Failure      404 {object} responses.APIResponse "Brand not found"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/brands/{id}/status/{status} [put]
func (bh *BrandHandler) UpdateBrandStatus(c *gin.Context) {
	brandIDStr := c.Param("id")
	brandStatusStr := c.Param("status")

	brandID, err := uuid.Parse(brandIDStr)
	if err != nil {
		response := responses.ErrorResponse("Invalid brand ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	brandStatus := enum.BrandStatus(brandStatusStr)
	if !enum.BrandStatus(brandStatus).IsValid() {
		response := responses.ErrorResponse("Invalid brand status", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	brand, err := bh.BrandService.UpdateBrandStatus(c.Request.Context(), brandID, brandStatus)
	if err != nil {
		response := responses.ErrorResponse("Failed to update brand status: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Brand status updated successfully", nil, brand))
}
