package handler

import (
	"core-backend/internal/application/dto/requests"
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
	contractPaymentService    iservice.ContractPaymentService
	paymentTransactionService iservice.PaymentTransactionService
	unitOfWork                irepository.UnitOfWork
	validator                 *validator.Validate
}

func NewContractPaymentHandler(
	contractPaymentService iservice.ContractPaymentService,
	paymentTransactionService iservice.PaymentTransactionService,
	unitOfWork irepository.UnitOfWork,
) *ContractPaymentHandler {
	validator := validator.New()
	return &ContractPaymentHandler{
		contractPaymentService:    contractPaymentService,
		paymentTransactionService: paymentTransactionService,
		unitOfWork:                unitOfWork,
		validator:                 validator,
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

	uow := h.unitOfWork.Begin(c.Request.Context())

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

// GetContractPaymentsByFilter godoc
//
//	@Summary	Get contract payments based on filtering criteria
//	@Tags		Contract Payments
//	@Accept		json
//	@Produce	json
//	@Param		contract_id		query		string										false	"Contract ID"		example("a1b2c3d4-e5f6-7a8b-9c0d-e1f2a3b4c5d6")
//	@Param		status			query		string										false	"Payment Status"	enums(PENDING,PAID,OVERDUE)		example("PAID")
//	@Param		due_date_from	query		string										false	"Due Date From"		format(date)					example("2023-01-01")
//	@Param		due_date_to		query		string										false	"Due Date To"		format(date)					example("2023-12-31")
//	@Param		payment_method	query		string										false	"Payment Method"	enums(BANK_TRANSFER,CASH,CHECK)	example("BANK_TRANSFER")
//	@Param		page			query		int											false	"Page number"		default(1)						example(1)
//	@Param		limit			query		int											false	"Items per page"	default(10)						example(10)
//	@Success	200				{object}	responses.ContractPaymentPaginationResponse	"Contract payments retrieved successfully"
//	@Failure	400				{object}	responses.APIResponse						"Invalid request or validation error"
//	@Failure	401				{object}	responses.APIResponse						"Unauthorized"
//	@Failure	500				{object}	responses.APIResponse						"Internal server error"
//	@Security	BearerAuth
//	@Router		/api/v1/contract_payments [get]
func (h *ContractPaymentHandler) GetContractPaymentsByFilter(c *gin.Context) {
	var req requests.ContractPaymentFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response := responses.ErrorResponse(err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}
	if err := h.validator.Struct(&req); err != nil {
		response := responses.ErrorResponse(err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	contractPaymentResponse, total, err := h.contractPaymentService.GetContractPaymentsByFilter(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			responses.ErrorResponse("Failed to retrieve contract payments", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.NewPaginationResponse(
		"Contract payments retrieved successfully",
		http.StatusOK,
		*contractPaymentResponse,
		responses.Pagination{Total: total, Page: req.Page, Limit: req.Limit},
	))
}

// GetContractPaymentByID godoc
//
//	@Summary	Get a contract payment by its ID
//	@Tags		Contract Payments
//	@Accept		json
//	@Produce	json
//	@Param		contract_payment_id	path		string															true	"Contract Payment ID"	example("b1c2d3e4-f5a6-7b8c-9d0e-f1a2b3c4d5e6")
//	@Success	200					{object}	responses.APIResponse{data=responses.ContractPaymenntResponse}	"Contract payment retrieved successfully"
//	@Failure	400					{object}	responses.APIResponse											"Invalid request or validation error"
//	@Failure	401					{object}	responses.APIResponse											"Unauthorized"
//	@Failure	404					{object}	responses.APIResponse											"Contract payment not found"
//	@Failure	500					{object}	responses.APIResponse											"Internal server error"
//	@Security	BearerAuth
//	@Router		/api/v1/contract_payments/{contract_payment_id} [get]
func (h *ContractPaymentHandler) GetContractPaymentByID(c *gin.Context) {
	contractPaymentID, err := extractParamID(c, "contract_payment_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid contract_payment_id: "+err.Error(), http.StatusBadRequest))
		return
	}

	contractPayment, err := h.contractPaymentService.GetContractPaymentByID(c.Request.Context(), contractPaymentID)
	if err != nil {
		var response *responses.APIResponse
		var statusCode int
		switch err.Error() {
		case fmt.Sprintf("contract payment with ID %d not found", contractPaymentID):
			response = responses.ErrorResponse(err.Error(), http.StatusNotFound)
			statusCode = http.StatusNotFound
		default:
			response = responses.ErrorResponse("Failed to retrieve contract payment", http.StatusInternalServerError)
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, response)
		return
	}

	c.JSON(http.StatusOK,
		responses.SuccessResponse("Contract payment retrieved successfully", utils.PtrOrNil(http.StatusOK), contractPayment))
}

// GeneratePaymentLink godoc
//
//	@Summary		Generate PayOS payment link for contract payment
//	@Description	Generate a PayOS payment link for a specific contract payment. Only PENDING payments can generate links.
//	@Tags			Contract Payments
//	@Accept			json
//	@Produce		json
//	@Param			contract_payment_id	path		string													true	"Contract Payment ID"	example("b1c2d3e4-f5a6-7b8c-9d0e-f1a2b3c4d5e6")
//	@Success		200					{object}	responses.APIResponse{data=responses.PayOSLinkResponse}	"Payment link generated successfully"
//	@Failure		400					{object}	responses.APIResponse									"Invalid request or payment not in PENDING status"
//	@Failure		404					{object}	responses.APIResponse									"Contract payment not found"
//	@Failure		500					{object}	responses.APIResponse									"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contract_payments/{contract_payment_id}/payment-link [post]
func (h *ContractPaymentHandler) GeneratePaymentLink(c *gin.Context) {
	contractPaymentID, err := extractParamID(c, "contract_payment_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid contract_payment_id: "+err.Error(), http.StatusBadRequest))
		return
	}

	// Begin transaction
	uow := h.unitOfWork.Begin(c.Request.Context())
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	// Generate payment link
	payosResp, err := h.contractPaymentService.CreatePaymentLinkFromContractPayment(
		c.Request.Context(),
		uow,
		contractPaymentID,
		h.paymentTransactionService,
	)

	if err != nil {
		uow.Rollback()
		var statusCode int
		var message string

		switch {
		case err.Error() == "contract payment not found":
			statusCode = http.StatusNotFound
			message = "Contract payment not found"
		case err.Error()[:45] == "payment link can only be generated for pending":
			statusCode = http.StatusBadRequest
			message = err.Error()
		default:
			statusCode = http.StatusInternalServerError
			message = "Failed to generate payment link"
		}

		c.JSON(statusCode, responses.ErrorResponse(message, statusCode))
		return
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to commit transaction", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK,
		responses.SuccessResponse("Payment link generated successfully", utils.PtrOrNil(http.StatusOK), payosResp))
}
