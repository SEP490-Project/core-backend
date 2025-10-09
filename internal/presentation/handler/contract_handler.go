package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ContractHandler struct {
	contractService iservice.ContractService
	fileService     iservice.FileService
	unitOfWork      irepository.UnitOfWork
	validator       *validator.Validate
}

func NewContractHandler(
	contractService iservice.ContractService,
	fileService iservice.FileService,
	unitOfWork irepository.UnitOfWork,
) *ContractHandler {
	return &ContractHandler{
		contractService: contractService,
		fileService:     fileService,
		unitOfWork:      unitOfWork,
		validator:       validator.New(),
	}
}

// CreateContract godoc
//
//	@Summary		Create new contract
//	@Description	Create a new contract and optionally update brand information
//	@Tags			Contracts
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			data	body		requests.CreateContractRequest							false	"Contract creation data (to be JSON-stringified and placed in 'data' form field)"
//	@Param			data	formData	string													true	"Contract creation data in JSON format of struct type requests.CreateContractRequest"
//	@Param			file	formData	file													true	"Contract file"
//	@Success		201		{object}	responses.APIResponse{data=responses.ContractResponse}	"Contract created successfully"
//	@Failure		400		{object}	responses.APIResponse									"Invalid request or validation error"
//	@Failure		401		{object}	responses.APIResponse									"Unauthorized"
//	@Failure		404		{object}	responses.APIResponse									"Brand not found"
//	@Failure		409		{object}	responses.APIResponse									"Contract number already exists"
//	@Failure		500		{object}	responses.APIResponse									"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts [post]
func (h *ContractHandler) CreateContract(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		responses := responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, responses)
		return
	}

	dataForm := c.PostForm("data")
	fileForm, err := c.FormFile("file")
	if dataForm == "" {
		responses := responses.ErrorResponse("Missing data in form", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	} else if err != nil {
		responses := responses.ErrorResponse("Failed to get file from form: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	var req requests.CreateContractRequest
	if err = json.Unmarshal([]byte(dataForm), &req); err != nil {
		responses := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}
	if err = h.validator.Struct(&req); err != nil {
		zap.L().Debug("Validation failed", zap.Error(err))
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	uow := h.unitOfWork.Begin()

	var contractResponse *responses.ContractResponse
	if contractResponse, err = h.contractService.CreateContract(c.Request.Context(), userID, &req, uow); err != nil {
		uow.Rollback()

		statusCode := http.StatusInternalServerError
		errorStr := err.Error()
		if errorStr == "brand not found" {
			statusCode = http.StatusNotFound
		} else if errorStr == fmt.Sprintf("contract number %s already exists", req.ContractNumber) ||
			errorStr == "failed to validate contract number" {
			statusCode = http.StatusConflict
		}

		zap.L().Error("Failed to create contract", zap.Error(err))
		response := responses.ErrorResponse("Failed to create contract: "+err.Error(), statusCode)
		c.JSON(statusCode, response)
		return
	}

	var fileURL string
	tempFilePath := fmt.Sprintf("/tmp/%s", fileForm.Filename)
	if err = c.SaveUploadedFile(fileForm, tempFilePath); err != nil {
		responses := responses.ErrorResponse("Failed to save uploaded file: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, responses)
		return
	}
	defer func() { _ = os.Remove(tempFilePath) }()
	if fileURL, err = h.fileService.UploadFile(userID.String(), tempFilePath, fileForm.Filename); err != nil {
		uow.Rollback()
		responses := responses.ErrorResponse("Failed to upload file: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, responses)
		return
	}

	contractID, _ := uuid.Parse(contractResponse.ID)
	if err = h.contractService.UpdateContractFileURL(c.Request.Context(), contractID, fileURL, uow); err != nil {
		uow.Rollback()
		responses := responses.ErrorResponse("Failed to update contract with file URL: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, responses)
		return
	}

	uow.Commit()

	statusCode := http.StatusCreated
	response := responses.SuccessResponse("Contract created successfully", &statusCode, contractResponse)
	c.JSON(http.StatusCreated, response)
}

// UpdateContract godoc
//
//	@Summary		Update contract
//	@Description	Update an existing contract and optionally update brand information
//	@Tags			Contracts
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string													true	"Contract ID"	format(uuid)
//	@Param			request	body		requests.UpdateContractRequest							true	"Contract update data"
//	@Success		200		{object}	responses.APIResponse{data=responses.ContractResponse}	"Contract updated successfully"
//	@Failure		400		{object}	responses.APIResponse									"Invalid request or validation error"
//	@Failure		401		{object}	responses.APIResponse									"Unauthorized"
//	@Failure		404		{object}	responses.APIResponse									"Contract not found"
//	@Failure		409		{object}	responses.APIResponse									"Contract number already exists"
//	@Failure		500		{object}	responses.APIResponse									"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id} [put]
func (h *ContractHandler) UpdateContract(c *gin.Context) {
	contractIDStr := c.Param("id")
	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		response := responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var req requests.UpdateContractRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("Failed to bind JSON", zap.Error(err))
		response := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate request
	if err = h.validator.Struct(&req); err != nil {
		zap.L().Debug("Validation failed", zap.Error(err))
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Update contract
	var contract *responses.ContractResponse
	contract, err = h.contractService.UpdateContract(c.Request.Context(), contractID, &req, h.unitOfWork)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "contract not found" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "contract number already exists" {
			statusCode = http.StatusConflict
		}

		zap.L().Error("Failed to update contract", zap.Error(err))
		response := responses.ErrorResponse("Failed to update contract: "+err.Error(), statusCode)
		c.JSON(statusCode, response)
		return
	}

	response := responses.SuccessResponse("Contract updated successfully", nil, contract)
	c.JSON(http.StatusOK, response)
}

// GetContractByID godoc
//
//	@Summary		Get contract by ID
//	@Description	Retrieve detailed information about a specific contract
//	@Tags			Contracts
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string													true	"Contract ID"	format(uuid)
//	@Success		200	{object}	responses.APIResponse{data=responses.ContractResponse}	"Contract retrieved successfully"
//	@Failure		400	{object}	responses.APIResponse									"Invalid contract ID"
//	@Failure		401	{object}	responses.APIResponse									"Unauthorized"
//	@Failure		404	{object}	responses.APIResponse									"Contract not found"
//	@Failure		500	{object}	responses.APIResponse									"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id} [get]
func (h *ContractHandler) GetContractByID(c *gin.Context) {
	contractIDStr := c.Param("id")
	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		response := responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	contract, err := h.contractService.GetContractByID(c.Request.Context(), contractID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "contract not found" {
			statusCode = http.StatusNotFound
		}

		zap.L().Error("Failed to get contract", zap.Error(err))
		response := responses.ErrorResponse("Failed to get contract: "+err.Error(), statusCode)
		c.JSON(statusCode, response)
		return
	}

	response := responses.SuccessResponse("Contract retrieved successfully", nil, contract)
	c.JSON(http.StatusOK, response)
}

// GetContracts godoc
//
//	@Summary		Get contracts with filters
//	@Description	Retrieve a paginated list of contracts with optional filters
//	@Tags			Contracts
//	@Accept			json
//	@Produce		json
//	@Param			brand_id	query		string									false	"Brand ID"			format(uuid)
//	@Param			type		query		string									false	"Contract type"		Enums(ADVERTISING, AFFILIATE, BRAND_AMBASSADOR, CO_PRODUCING)
//	@Param			status		query		string									false	"Contract status"	Enums(DRAFT, ACTIVE, COMPLETED, TERMINATED)
//	@Param			keyword		query		string									false	"Search keyword (title or contract number)"
//	@Param			start_date	query		string									false	"Start date filter"	format(date-time)
//	@Param			end_date	query		string									false	"End date filter"	format(date-time)
//	@Param			page		query		int										false	"Page number"		default(1)
//	@Param			limit		query		int										false	"Items per page"	default(10)
//	@Param			sort_by		query		string									false	"Sort by field"		default(created_at)
//	@Param			sort_order	query		string									false	"Sort order"		Enums(asc, desc)	default(desc)
//	@Success		200			{object}	responses.ContractPaginationResponse	"Contracts retrieved successfully"
//	@Failure		400			{object}	responses.APIResponse					"Invalid query parameters"
//	@Failure		401			{object}	responses.APIResponse					"Unauthorized"
//	@Failure		500			{object}	responses.APIResponse					"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts [get]
func (h *ContractHandler) GetContracts(c *gin.Context) {
	var filterReq requests.ContractFilterRequest

	// Bind query parameters
	if err := c.ShouldBindQuery(&filterReq); err != nil {
		zap.L().Error("Failed to bind query parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate request
	if err := h.validator.Struct(&filterReq); err != nil {
		zap.L().Debug("Validation failed", zap.Error(err))
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Get contracts
	contracts, total, err := h.contractService.GetByFilter(c.Request.Context(), &filterReq)
	if err != nil {
		zap.L().Error("Failed to get contracts", zap.Error(err))
		response := responses.ErrorResponse("Failed to get contracts: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// Calculate pagination
	page := max(filterReq.Page, 1)
	limit := filterReq.Limit
	if limit < 1 {
		limit = 10
	}
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

	response := responses.NewPaginationResponse(
		"Contracts retrieved successfully",
		http.StatusOK,
		contracts,
		pagination,
	)
	c.JSON(http.StatusOK, response)
}

// GetContractsByBrandID godoc
//
//	@Summary		Get contracts by brand ID
//	@Description	Retrieve all contracts for a specific brand
//	@Tags			Contracts
//	@Accept			json
//	@Produce		json
//	@Param			brand_id	path		string									true	"Brand ID"			format(uuid)
//	@Param			page		query		int										false	"Page number"		default(1)
//	@Param			limit		query		int										false	"Items per page"	default(10)
//	@Success		200			{object}	responses.ContractPaginationResponse	"Contracts retrieved successfully"
//	@Failure		400			{object}	responses.APIResponse					"Invalid brand ID"
//	@Failure		401			{object}	responses.APIResponse					"Unauthorized"
//	@Failure		500			{object}	responses.APIResponse					"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/brands/{brand_id} [get]
func (h *ContractHandler) GetContractsByBrandID(c *gin.Context) {
	brandIDStr := c.Param("brand_id")
	brandID, err := uuid.Parse(brandIDStr)
	if err != nil {
		response := responses.ErrorResponse("Invalid brand ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	contracts, total, err := h.contractService.GetContractsByBrandID(c.Request.Context(), brandID, page, limit)
	if err != nil {
		zap.L().Error("Failed to get contracts by brand ID", zap.Error(err))
		response := responses.ErrorResponse("Failed to get contracts: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// Calculate pagination
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

	response := responses.NewPaginationResponse(
		"Contracts retrieved successfully",
		http.StatusOK,
		contracts,
		pagination,
	)
	c.JSON(http.StatusOK, response)
}

// DeleteContract godoc
//
//	@Summary		Delete contract
//	@Description	Soft delete a contract by ID
//	@Tags			Contracts
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"Contract ID"	format(uuid)
//	@Success		200	{object}	responses.APIResponse	"Contract deleted successfully"
//	@Failure		400	{object}	responses.APIResponse	"Invalid contract ID"
//	@Failure		401	{object}	responses.APIResponse	"Unauthorized"
//	@Failure		404	{object}	responses.APIResponse	"Contract not found"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id} [delete]
func (h *ContractHandler) DeleteContract(c *gin.Context) {
	contractIDStr := c.Param("id")
	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		response := responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	err = h.contractService.DeleteContractByID(c.Request.Context(), contractID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "contract not found" {
			statusCode = http.StatusNotFound
		}

		zap.L().Error("Failed to delete contract", zap.Error(err))
		response := responses.ErrorResponse("Failed to delete contract: "+err.Error(), statusCode)
		c.JSON(statusCode, response)
		return
	}

	response := responses.SuccessResponse("Contract deleted successfully", nil, nil)
	c.JSON(http.StatusOK, response)
}

// ApproveContract godoc
//
//	@Summary		Approve contract
//	@Description	Approve a contract by ID
//	@Tags			Contracts
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"Contract ID"	format(uuid)
//	@Success		200	{object}	responses.APIResponse	"Contract approved successfully"
//	@Failure		400	{object}	responses.APIResponse	"Invalid contract ID"
//	@Failure		401	{object}	responses.APIResponse	"Unauthorized"
//	@Failure		404	{object}	responses.APIResponse	"Contract not found"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id}/approve [patch]
func (h *ContractHandler) ApproveContract(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		responses := responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	err = h.contractService.ApproveContract(c.Request.Context(), id)
	if err == fmt.Errorf("contract not found") {
		responses := responses.ErrorResponse("Contract not found", http.StatusNotFound)
		c.JSON(http.StatusNotFound, responses)
		return
	} else if err != nil {
		responses := responses.ErrorResponse("Failed to approve contract: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, responses)
		return
	}

	responses := responses.SuccessResponse("Contract approved successfully", nil, nil)
	c.JSON(http.StatusOK, responses)
}
