package handler

import (
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/pkg/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type ContractPaymentHandler struct {
	contractPaymentService iservice.ContractPaymentService
	unitOfWork             irepository.UnitOfWork
	validator              *validator.Validate
}

func NewContractPaymentHandler(
	contractPaymentService iservice.ContractPaymentService,
	unitOfWork irepository.UnitOfWork,
) *ContractPaymentHandler {
	validator := validator.New()
	return &ContractPaymentHandler{
		contractPaymentService: contractPaymentService,
		unitOfWork:             unitOfWork,
		validator:              validator,
	}
}

// CreateContractPaymentsFromContract godoc
//
//	@Summary	Create contract payments based on a contract ID
//	@Tags		Contract Payments
//	@Accept		json
//	@Produce	json
//	@Param		contract_id	path		string								true	"Contract ID"
//	@Success	200			{object}	responses.APIResponse{data=string}	"Contract retrieved successfully"
//	@Failure	400			{object}	responses.APIResponse				"Invalid request or validation error"
//	@Failure	401			{object}	responses.APIResponse				"Unauthorized"
//	@Failure	404			{object}	responses.APIResponse				"Brand not found"
//	@Failure	409			{object}	responses.APIResponse				"Contract number already exists"
//	@Failure	500			{object}	responses.APIResponse				"Internal server error"
//	@Security	BearerAuth
//	@Router		/api/v1/contract_payments/contract/{contract_id} [post]
func (h *ContractPaymentHandler) CreateContractPaymentsFromContract(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		response := responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	contractIDStr := c.Param("contract_id")
	if contractIDStr == "" {
		response := responses.ErrorResponse("contract_id parameter is required", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}
	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		response := responses.ErrorResponse("invalid contract_id format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	uow := h.unitOfWork.Begin()

	if err := h.contractPaymentService.CreateContractPaymentsFromContract(c.Request.Context(), userID, contractID, uow); err != nil {
		uow.Rollback()
		// contract with ID %s not found
		var message string
		var statusCode int
		switch err.Error() {
		case fmt.Sprintf("contract with ID %s not found", contractID):
			message = err.Error()
			statusCode = http.StatusNotFound
		default:
			message = "Failed to create contract payments"
			statusCode = http.StatusInternalServerError
		}
		response := responses.ErrorResponse(message, statusCode)
		c.JSON(statusCode, response)
		return
	}

	uow.Commit()

	response := responses.SuccessResponse("Contract payments created successfully", utils.IntPtr(http.StatusCreated), nil)
	c.JSON(http.StatusOK, response)
}
