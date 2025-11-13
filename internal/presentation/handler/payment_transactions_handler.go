package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type PaymentTransactionsHandler struct {
	paymentTransactionService iservice.PaymentTransactionService
	validator                 *validator.Validate
}

func NewPaymentTransactionsHandler(paymentTransactionService iservice.PaymentTransactionService) *PaymentTransactionsHandler {
	validator := validator.New()
	validator.RegisterStructValidation(requests.ValidatePaymentTransactionFilterRequest, requests.PaymentTransactionFilterRequest{})
	return &PaymentTransactionsHandler{
		paymentTransactionService: paymentTransactionService,
		validator:                 validator,
	}
}

// GetByFilter godoc
//
//	@Summary		Get payment transactions by filter
//	@Description	Retrieve payment transactions based on various filter criteria with pagination
//	@Tags			PayOS
//	@Produce		json
//	@Param			reference_id			query		string											false	"Filter by Reference ID (UUID)"
//	@Param			reference_type			query		string											false	"Filter by Reference Type (ORDER, CONTRACT_PAYMENT)"
//	@Param			payer_id				query		string											false	"Filter by Payer User ID (UUID)"
//	@Param			status					query		string											false	"Filter by Payment Status (PENDING, COMPLETED, CANCELLED, EXPIRED)"
//	@Param			method					query		string											false	"Filter by Payment Method (CREDIT_CARD, BANK_TRANSFER, E_WALLET)"
//	@Param			transaction_from_date	query		string											false	"Filter by Transaction From Date (YYYY-MM-DD)"
//	@Param			transaction_to_date		query		string											false	"Filter by Transaction To Date (YYYY-MM-DD)"
//	@Param			page					query		int												false	"Page number for pagination (default is 1)"
//	@Param			limit					query		int												false	"Number of items per page for pagination (default is 10)"
//	@Param			sort_by					query		string											false	"Field to sort by (e.g., transaction_date, amount)"
//	@Param			sort_order				query		string											false	"Sort order (asc or desc)"
//	@Success		200						{object}	responses.PaymentTransactionPaginationResponse	"Payment transactions retrieved successfully"
//	@Failure		400						{object}	responses.APIResponse							"Bad request"
//	@Failure		500						{object}	responses.APIResponse							"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/payments [get]
func (h *PaymentTransactionsHandler) GetByFilter(c *gin.Context) {
	var req requests.PaymentTransactionFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}
	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}

	paymentResponse, total, err := h.paymentTransactionService.GetPaymentTransactionByFilter(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get payment transactions", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.NewPaginationResponse(
		"Payment transactions retrieved successfully",
		http.StatusOK,
		paymentResponse,
		responses.Pagination{
			Total: total,
			Page:  req.Page,
			Limit: req.Limit},
	))
}

// GetByProfileFilter godoc
//
//	@Summary		Get payment transactions by profile filter
//	@Description	Retrieve payment transactions based on various filter criteria with pagination
//	@Tags			PayOS
//	@Produce		json
//	@Param			reference_id			query		string											false	"Filter by Reference ID (UUID)"
//	@Param			reference_type			query		string											false	"Filter by Reference Type (ORDER, CONTRACT_PAYMENT)"
//	@Param			status					query		string											false	"Filter by Payment Status (PENDING, COMPLETED, CANCELLED, EXPIRED)"
//	@Param			method					query		string											false	"Filter by Payment Method (CREDIT_CARD, BANK_TRANSFER, E_WALLET)"
//	@Param			transaction_from_date	query		string											false	"Filter by Transaction From Date (YYYY-MM-DD)"
//	@Param			transaction_to_date		query		string											false	"Filter by Transaction To Date (YYYY-MM-DD)"
//	@Param			page					query		int												false	"Page number for pagination (default is 1)"
//	@Param			limit					query		int												false	"Number of items per page for pagination (default is 10)"
//	@Param			sort_by					query		string											false	"Field to sort by (e.g., transaction_date, amount)"
//	@Param			sort_order				query		string											false	"Sort order (asc or desc)"
//	@Success		200						{object}	responses.PaymentTransactionPaginationResponse	"Payment transactions retrieved successfully"
//	@Failure		400						{object}	responses.APIResponse							"Bad request"
//	@Failure		500						{object}	responses.APIResponse							"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/payments/profile [get]
func (h *PaymentTransactionsHandler) GetByProfileFilter(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized", http.StatusUnauthorized))
		return
	}
	var req requests.PaymentTransactionFilterRequest
	if err = c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}
	req.PayerID = utils.PtrOrNil(userID.String())
	if err = h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}

	paymentResponse, total, err := h.paymentTransactionService.GetPaymentTransactionByFilter(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get payment transactions", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.NewPaginationResponse(
		"Payment transactions retrieved successfully",
		http.StatusOK,
		paymentResponse,
		responses.Pagination{
			Total: total,
			Page:  req.Page,
			Limit: req.Limit},
	))

}

// GetByID godoc
//
//	@Summary		Get payment transaction by ID
//	@Description	Retrieve a payment transaction by its unique ID
//	@Tags			PayOS
//	@Produce		json
//	@Param			id	path		string																true	"Payment Transaction ID (UUID)"
//	@Success		200	{object}	responses.APIResponse{data=responses.PaymentTransactionResponse}	"Payment transaction retrieved successfully"
//	@Failure		400	{object}	responses.APIResponse												"Bad request"
//	@Failure		404	{object}	responses.APIResponse												"Payment transaction not found"
//	@Failure		500	{object}	responses.APIResponse												"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/payments/id/{id} [get]
func (h *PaymentTransactionsHandler) GetByID(c *gin.Context) {
	transactionID, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid transaction ID", http.StatusBadRequest))
		return
	}

	paymentResponse, err := h.paymentTransactionService.GetPaymentTransactionByID(c.Request.Context(), transactionID)
	if err != nil {
		switch err.Error() {
		case "payment transaction not found":
			c.JSON(http.StatusNotFound, responses.ErrorResponse("Payment transaction not found", http.StatusNotFound))
		default:
			c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get payment transaction", http.StatusInternalServerError))
		}
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Payment transaction retrieved successfully", nil, paymentResponse))
}

// GetByOrderCode godoc
//
//	@Summary		Get payment transaction by order code
//	@Description	Retrieve a payment transaction by its associated order code
//	@Tags			PayOS
//	@Produce		json
//	@Param			order_code	path		string																true	"Order Code"
//	@Success		200			{object}	responses.APIResponse{data=responses.PaymentTransactionResponse}	"Payment transaction retrieved successfully"
//	@Failure		400			{object}	responses.APIResponse												"Bad request"
//	@Failure		404			{object}	responses.APIResponse												"Payment transaction not found"
//	@Failure		500			{object}	responses.APIResponse												"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/payments/order-code/{order_code} [get]
func (h *PaymentTransactionsHandler) GetByOrderCode(c *gin.Context) {
	orderCode := c.Param("order_code")
	if orderCode == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Order code is required", http.StatusBadRequest))
		return
	}
	paymentResponse, err := h.paymentTransactionService.GetPaymentTransactionByOrderCode(c.Request.Context(), orderCode)
	if err != nil {
		switch err.Error() {
		case "payment transaction not found":
			c.JSON(http.StatusNotFound, responses.ErrorResponse("Payment transaction not found", http.StatusNotFound))
		default:
			c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get payment transaction", http.StatusInternalServerError))
		}
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Payment transaction retrieved successfully", nil, paymentResponse))
}
