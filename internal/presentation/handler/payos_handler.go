package handler

import (
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

type PayOsHandler struct {
	config                    *config.AppConfig
	paymentTransactionService iservice.PaymentTransactionService
	stateTransferService      iservice.StateTransferService
	payosProxy                iproxies.PayOSProxy
	unitOfWork                irepository.UnitOfWork
	webhookDataService        iservice.WebhookDataService
	validator                 *validator.Validate
}

func NewPayOsHandler(
	config *config.AppConfig,
	paymentTransactionService iservice.PaymentTransactionService,
	stateTransferService iservice.StateTransferService,
	webhookDataService iservice.WebhookDataService,
	payosProxy iproxies.PayOSProxy,
	unitOfWork irepository.UnitOfWork,
) *PayOsHandler {
	validator := validator.New()

	return &PayOsHandler{
		config:                    config,
		paymentTransactionService: paymentTransactionService,
		stateTransferService:      stateTransferService,
		payosProxy:                payosProxy,
		unitOfWork:                unitOfWork,
		webhookDataService:        webhookDataService,
		validator:                 validator,
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
//	@Param			order_code	path		string															true	"Order Code"
//	@Success		200			{object}	responses.APIResponse{data=responses.PayOSOrderInfoResponse}	"Payment info retrieved successfully"
//	@Failure		400			{object}	responses.APIResponse											"Bad request"
//	@Failure		404			{object}	responses.APIResponse											"Payment not found"
//	@Security		BearerAuth
//	@Router			/api/v1/payos/payment/{orderCode} [get]
func (h *PayOsHandler) GetPaymentInfo(c *gin.Context) {
	orderCode := c.Param("order_code")
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

	// Store raw webhook data for audit (non-blocking, best effort)
	webhookData := &model.WebhookData{
		Source:     model.WebhookSourcePayOS,
		RawPayload: datatypes.JSON(bodyBytes),
	}

	// Parse webhook payload
	var webhookPayload dtos.PayOSWebhookPayload
	if err = json.Unmarshal(bodyBytes, &webhookPayload); err != nil {
		zap.L().Error("Failed to parse webhook payload", zap.Error(err))
		// Store webhook data with error
		eventType := "payment.unknown"
		webhookData.EventType = &eventType
		webhookData.MarkFailed("Failed to parse webhook payload: " + err.Error())
		h.webhookDataService.SaveWebhookData(c.Request.Context(), true, webhookData)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook payload"})
		return
	}

	// Update webhook data with parsed info
	eventType := "payment." + webhookPayload.Code
	externalID := utils.ToString(webhookPayload.Data.OrderCode)
	webhookData.EventType = &eventType
	webhookData.ExternalID = &externalID

	// Verify signature
	// Note: PayOS sends the signature in the payload itself, not as a header
	// We need to reconstruct the data string and verify
	dataBytes, err := json.Marshal(webhookPayload.Data)
	if err != nil {
		zap.L().Error("Failed to marshal webhook data", zap.Error(err))
		webhookData.MarkFailed("Failed to marshal webhook data: " + err.Error())
		h.webhookDataService.SaveWebhookData(c.Request.Context(), true, webhookData)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook data"})
		return
	}

	if !h.payosProxy.VerifyWebhookSignature(dataBytes, webhookPayload.Signature) {
		zap.L().Warn("Invalid webhook signature",
			zap.Int64("order_code", webhookPayload.Data.OrderCode),
			zap.String("signature", webhookPayload.Signature))
		webhookData.MarkFailed("Invalid webhook signature")
		h.webhookDataService.SaveWebhookData(c.Request.Context(), true, webhookData)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// Begin transaction for webhook processing
	uow := h.unitOfWork.Begin(c.Request.Context())
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
	}()

	// Process webhook
	if err = h.paymentTransactionService.ProcessWebhook(c.Request.Context(), uow, &webhookPayload); err != nil {
		uow.Rollback()
		zap.L().Error("Failed to process webhook", zap.Error(err))
		webhookData.MarkFailed("Failed to process webhook: " + err.Error())
		h.webhookDataService.SaveWebhookData(c.Request.Context(), true, webhookData)
		// Still return 200 to acknowledge receipt, but log the error
		c.JSON(http.StatusInternalServerError, gin.H{"status": "received", "error": err.Error()})
		return
	}

	// After processing payment transaction, find the updated transaction and trigger state transition
	// transaction, err := h.paymentTransactionService.GetPaymentTransactionByOrderCode(c.Request.Context(), strconv.Itoa(webhookPayload.Data.OrderCode))
	transaction, err := h.paymentTransactionService.GetPaymentTransactionByOrderCode(c.Request.Context(), utils.ToString(webhookPayload.Data.OrderCode))
	if err == nil && transaction != nil {
		// Use StateTransferService to handle state transition and side effects

		// 2. Map PayOS status to internal status
		var payosStatus string
		if webhookPayload.Code == "00" {
			payosStatus = enum.PayOSStatusPaid.String()
		} else {
			payosStatus = webhookPayload.Data.Code
		}

		newStatus := dtos.MapPayOSStatusString(payosStatus)

		if stateErr := h.stateTransferService.MovePaymentTransactionToState(
			c.Request.Context(),
			uow,
			transaction.ID,
			newStatus,
			uuid.Nil,
		); stateErr != nil {
			uow.Rollback()
			zap.L().Error("Failed to handle payment transaction state change",
				zap.String("transaction_id", transaction.ID.String()),
				zap.String("status", string(transaction.Status)),
				zap.Error(stateErr))
			webhookData.MarkFailed("Failed to update related entities: " + stateErr.Error())
			h.webhookDataService.SaveWebhookData(c.Request.Context(), true, webhookData)
			c.JSON(http.StatusInternalServerError, gin.H{"status": "received", "error": "Failed to update related entities"})
			return
		}
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		zap.L().Error("Failed to commit webhook transaction", zap.Error(err))
		webhookData.MarkFailed("Failed to commit transaction: " + err.Error())
		h.webhookDataService.SaveWebhookData(c.Request.Context(), true, webhookData)
		c.JSON(http.StatusOK, gin.H{"status": "received", "error": "Failed to commit transaction"})
		return
	}

	// Mark webhook as processed and store
	webhookData.MarkProcessed()
	h.webhookDataService.SaveWebhookData(c.Request.Context(), true, webhookData)

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
	uow := h.unitOfWork.Begin(c.Request.Context())
	defer func() {
		if r := recover(); r != nil {
			err := uow.Rollback()
			if err != nil {
				return
			}
		}
	}()
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

// HandleCancelCallback godoc
//
//	@Summary		Handle PayOS payment cancellation callback
//	@Description	Handles user redirection after cancelling a PayOS payment. Cancels the payment link and redirects the user.
//	@Tags			PayOS
//	@Produce		json
//	@Param			returnUrl	query	string				false	"URL to redirect to after cancellation"
//	@Param			code		query	string				true	"Payment link code"
//	@Param			id			query	string				true	"Payment transaction ID"
//	@Param			cancel		query	bool				true	"Indicates if the payment was cancelled"
//	@Param			status		query	enum.PayOSStatus	true	"Status of the payment link"
//	@Param			orderCode	query	string				true	"Order code associated with the payment"
//	@Success		302			"Redirects to the specified return URL or default cancel URL"
//	@Failure		400			{object}	responses.APIResponse	"Bad request"
//	@Failure		500			{object}	responses.APIResponse	"Internal server error"
//	@Router			/api/v1/payos/cancel-callback [get]
func (h *PayOsHandler) HandleCancelCallback(c *gin.Context) {
	var req requests.CancelPaymentRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}
	zap.L().Debug("Cancel payment callback received", zap.Any("req", req))

	paymentResponse, err := h.paymentTransactionService.GetPaymentTransactionByOrderCode(c.Request.Context(), req.OrderCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get payment transaction", http.StatusInternalServerError))
		return
	}

	switch paymentResponse.Status {
	case enum.PaymentTransactionStatusCancelled.String(), enum.PaymentTransactionStatusExpired.String():
		c.JSON(http.StatusOK, responses.SuccessResponse("Payment transaction is already cancelled or expired", utils.PtrOrNil(http.StatusOK), nil))
		return
	case enum.PaymentTransactionStatusCompleted.String():
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Payment transaction is already completed", http.StatusBadRequest))
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			c.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse("Internal server error", http.StatusInternalServerError))
		}
	}()

	if err = h.stateTransferService.MovePaymentTransactionToState(c.Request.Context(), uow, paymentResponse.ID, enum.PaymentTransactionStatusCancelled, uuid.Nil); err != nil {
		uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to cancel payment link", http.StatusInternalServerError))
		return
	}

	if err = uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to cancel payment link", http.StatusInternalServerError))
		return
	}

	var redirectURL string
	queryMap := map[string]any{
		"code":      req.Code,
		"id":        req.ID,
		"cancel":    req.Cancel,
		"status":    req.Status,
		"orderCode": req.OrderCode,
	}
	if req.ReturnURL != "" {
		redirectURL, err = utils.AddQueryParams(req.ReturnURL, queryMap)
	} else {
		redirectURL, err = utils.AddQueryParams(h.config.PayOS.FrontendCancelURL, queryMap)
	}
	if err != nil {
		zap.L().Error("Failed to add return URL query param", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to add return URL query param", http.StatusInternalServerError))
		return
	}

	c.Redirect(http.StatusFound, redirectURL)
}
