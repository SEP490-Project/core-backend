package handler

import (
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PayOsHandler struct {
	paymentTransactionService iservice.PaymentTransactionService
	stateTransferService      iservice.StateTransferService
	payosProxy                iproxies.PayOSProxy
	unitOfWork                irepository.UnitOfWork
}

func NewPayOsHandler(
	paymentTransactionService iservice.PaymentTransactionService,
	stateTransferService iservice.StateTransferService,
	payosProxy iproxies.PayOSProxy,
	unitOfWork irepository.UnitOfWork,
) *PayOsHandler {
	return &PayOsHandler{
		paymentTransactionService: paymentTransactionService,
		stateTransferService:      stateTransferService,
		payosProxy:                payosProxy,
		unitOfWork:                unitOfWork,
	}
}

// GeneratePaymentLink godoc
//
//	@Summary		Create a PayOS payment link
//	@Description	Generate a new PayOS payment link. Backend automatically sets cancelUrl and returnUrl from config.
//	@Tags			PayOS
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.PaymentRequest									true	"Payment Request"
//	@Success		200		{object}	responses.APIResponse{data=responses.PayOSLinkResponse}	"Payment link created successfully"
//	@Failure		400		{object}	responses.APIResponse									"Bad request"
//	@Failure		500		{object}	responses.APIResponse									"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/payos/payment [post]
func (h *PayOsHandler) GeneratePaymentLink(c *gin.Context) {
	var req requests.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body", http.StatusBadRequest))
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
	result, err := h.paymentTransactionService.GeneratePaymentLink(c.Request.Context(), uow, &req)
	if err != nil {
		uow.Rollback()
		zap.L().Error("Failed to generate payment link", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to generate payment link", http.StatusInternalServerError))
		return
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		zap.L().Error("Failed to commit transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to save payment transaction", http.StatusInternalServerError))
		return
	}

	statusCode := http.StatusOK
	c.JSON(http.StatusOK, responses.SuccessResponse("Payment link generated successfully", &statusCode, result))
}

// GetPaymentInfo godoc
//
//	@Summary		Get PayOS payment information
//	@Description	Retrieve payment details by order code
//	@Tags			PayOS
//	@Produce		json
//	@Param			orderCode	path		string															true	"Order Code"
//	@Success		200			{object}	responses.APIResponse{data=responses.PayOSOrderInfoResponse}	"Payment info retrieved successfully"
//	@Failure		400			{object}	responses.APIResponse											"Bad request"
//	@Failure		404			{object}	responses.APIResponse											"Payment not found"
//	@Security		BearerAuth
//	@Router			/api/v1/payos/payment/{orderCode} [get]
func (h *PayOsHandler) GetPaymentInfo(c *gin.Context) {
	orderCode := c.Param("orderCode")
	if orderCode == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Order code is required", http.StatusBadRequest))
		return
	}

	result, err := h.paymentTransactionService.GetPaymentStatus(c.Request.Context(), orderCode)
	if err != nil {
		zap.L().Error("Failed to get payment info", zap.Error(err), zap.String("order_code", orderCode))
		c.JSON(http.StatusNotFound, responses.ErrorResponse("Payment not found", http.StatusNotFound))
		return
	}

	statusCode := http.StatusOK
	c.JSON(http.StatusOK, responses.SuccessResponse("Payment info retrieved successfully", &statusCode, result))
}

// HandleWebhook godoc
//
//	@Summary		PayOS webhook endpoint
//	@Description	Receives payment status updates from PayOS. This is a public endpoint with signature verification.
//	@Tags			PayOS
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		dtos.PayOSWebhookPayload	true	"Webhook payload from PayOS"
//	@Success		200		{object}	map[string]string			"Webhook processed successfully"
//	@Failure		400		{object}	map[string]string			"Bad request"
//	@Failure		401		{object}	map[string]string			"Invalid signature"
//	@Router			/api/v1/payos/webhook [post]
func (h *PayOsHandler) HandleWebhook(c *gin.Context) {
	// Read raw body for signature verification
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		zap.L().Error("Failed to read webhook body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Parse webhook payload
	var webhookPayload dtos.PayOSWebhookPayload
	if err = json.Unmarshal(bodyBytes, &webhookPayload); err != nil {
		zap.L().Error("Failed to parse webhook payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook payload"})
		return
	}

	// Verify signature
	// Note: PayOS sends the signature in the payload itself, not as a header
	// We need to reconstruct the data string and verify
	dataBytes, err := json.Marshal(webhookPayload.Data)
	if err != nil {
		zap.L().Error("Failed to marshal webhook data", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook data"})
		return
	}

	if !h.payosProxy.VerifyWebhookSignature(dataBytes, webhookPayload.Signature) {
		zap.L().Warn("Invalid webhook signature",
			zap.Int64("order_code", webhookPayload.Data.OrderCode),
			zap.String("signature", webhookPayload.Signature))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// Begin transaction for webhook processing
	uow := h.unitOfWork.Begin(c.Request.Context())
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	// Process webhook
	if err = h.paymentTransactionService.ProcessWebhook(c.Request.Context(), uow, &webhookPayload); err != nil {
		uow.Rollback()
		zap.L().Error("Failed to process webhook", zap.Error(err))
		// Still return 200 to acknowledge receipt, but log the error
		c.JSON(http.StatusOK, gin.H{"status": "received", "error": err.Error()})
		return
	}

	// After processing payment transaction, find the updated transaction and trigger state transition
	filterQuery := func(db *gorm.DB) *gorm.DB {
		return db.Where("payos_metadata->>'order_code' = ?", strconv.FormatInt(webhookPayload.Data.OrderCode, 10))
	}

	transactions, _, err := uow.PaymentTransaction().GetAll(c.Request.Context(), filterQuery, nil, 1, 1)
	if err == nil && len(transactions) > 0 {
		transaction := transactions[0]

		// Use StateTransferService to handle state transition and side effects
		if stateErr := h.stateTransferService.MovePaymentTransactionToState(
			c.Request.Context(),
			uow,
			transaction.ID,
			transaction.Status,
			uuid.Nil,
		); stateErr != nil {
			uow.Rollback()
			zap.L().Error("Failed to handle payment transaction state change",
				zap.String("transaction_id", transaction.ID.String()),
				zap.String("status", string(transaction.Status)),
				zap.Error(stateErr))
			c.JSON(http.StatusOK, gin.H{"status": "received", "error": "Failed to update related entities"})
			return
		}
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		zap.L().Error("Failed to commit webhook transaction", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"status": "received", "error": "Failed to commit transaction"})
		return
	}

	zap.L().Info("Webhook processed successfully", zap.Int64("order_code", webhookPayload.Data.OrderCode))
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// CancelExpiredLinks godoc
//
//	@Summary		Cancel expired payment links
//	@Description	Manually trigger cancellation of all expired PayOS payment links
//	@Tags			PayOS
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=map[string]int}	"Cancellation completed"
//	@Failure		500	{object}	responses.APIResponse						"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/payos/cancel [post]
func (h *PayOsHandler) CancelExpiredLinks(c *gin.Context) {
	cancelledCount, err := h.paymentTransactionService.CancelExpiredLinks(c.Request.Context())
	if err != nil {
		zap.L().Error("Failed to cancel expired links", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to cancel expired links", http.StatusInternalServerError))
		return
	}

	statusCode := http.StatusOK
	result := map[string]int{
		"cancelled_count": cancelledCount,
	}
	c.JSON(http.StatusOK, responses.SuccessResponse("Expired links cancellation completed", &statusCode, result))
}

// ConfirmWebhookURL godoc
//
//	@Summary		Confirm PayOS webhook URL
//	@Description	Confirm the webhook URL with PayOS to start receiving webhooks
//	@Tags			PayOS
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.ConfirmWebhookRequest									true	"Webhook URL payload"
//	@Success		200		{object}	responses.APIResponse{data=dtos.PayOSConfirmWebhookResponse}	"Webhook URL confirmed successfully"
//	@Failure		400		{object}	responses.APIResponse											"Bad request"
//	@Failure		500		{object}	responses.APIResponse											"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/payos/confirm-webhook [post]
func (h *PayOsHandler) ConfirmWebhookURL(c *gin.Context) {
	var req requests.ConfirmWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body", http.StatusBadRequest))
		return
	}
	result, err := h.paymentTransactionService.ConfirmWebhookURL(c.Request.Context(), req.WebhookURL)
	if err != nil {
		zap.L().Error("Failed to confirm webhook URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to confirm webhook URL", http.StatusInternalServerError))
		return
	}

	statusCode := http.StatusOK
	c.JSON(http.StatusOK, responses.SuccessResponse("Webhook URL confirmed successfully", &statusCode, result))
}

func (h *PayOsHandler) PayOSCancelInterceptor(c *gin.Context) {
	// Get all query parameters
	queryParams := c.Request.URL.Query()

	// Log all query parameters with zap
	for key, values := range queryParams {
		zap.L().Info("Query parameter",
			zap.String("key", key),
			zap.Strings("values", values),
		)
	}

	// Extract fwdUrl if present
	fwdUrl := c.Query("fwdUrl")
	if fwdUrl == "" {
		fwdUrl = "https://facebook.com"
	}

	// Redirect to fwdUrl
	c.Redirect(http.StatusFound, fwdUrl)
}
