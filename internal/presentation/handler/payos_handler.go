package handler

import (
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"core-backend/internal/application/interfaces/iservice_third_party"
)

type PayOsHandler struct {
	payOsService iservice_third_party.PayOSService
	config       *config.AppConfig
}

func NewPayOsHandler(payOsService iservice_third_party.PayOSService) *PayOsHandler {
	return &PayOsHandler{
		payOsService: payOsService,
		config:       config.GetAppConfig(),
	}
}

// GeneratePaymentLink godoc
//
//	@Summary		Create a PayOS payment
//	@Description	Initiate a payment with PayOS. Backend sẽ tự set `cancelUrl` và `returnUrl` từ config, KHÔNG lấy từ client.
//	@Tags			payos
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.PaymentRequest		true	"Payment Request (client should NOT send cancelUrl/returnUrl)"
//	@Success		200		{object}	responses.PaymentResponse	"PayOS wrapper response"
//	@Failure		400		{object}	map[string]string			"error"
//	@Router			/api/v1/payos/payment [post]
func (h *PayOsHandler) GeneratePaymentLink(c *gin.Context) {
	var req requests.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.payOsService.GeneratePayOSLink(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// InspectPayOSOrder godoc
//
//	@Summary		Get PayOS order info
//	@Description	Inspect payment detail
//	@Tags			payos
//	@Produce		json
//	@Param			orderCode	path		string	true	"Order Code"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		400			{object}	map[string]string
//	@Router			/api/v1/payos/payment/{orderCode} [get]
func (h *PayOsHandler) InspectPayOSOrder(c *gin.Context) {
	orderCode := c.Param("orderCode")
	result, err := h.payOsService.GetPayOSOrderInfo(orderCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

//func (h *PayOsHandler) HandlePayOsWebhook(c *gin.Context) {
//	var payload map[string]interface{}
//	if err := c.ShouldBindJSON(&payload); err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//		return
//	}
//	if err := h.payOsService.ProcessPayOSWebhook(payload); err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//		return
//	}
//	c.JSON(http.StatusOK, gin.H{"status": "success"})
//}

func (h *PayOsHandler) CancelCallback(c *gin.Context) {
	codeStr := c.Query("code")     // ví dụ "00"
	payosID := c.Query("id")       // ví dụ "ade25886f6f54a7d9a8840b1b48864a1"
	cancelStr := c.Query("cancel") // ví dụ "true"
	status := c.Query("status")    // ví dụ "CANCELLED"
	orderCodeStr := c.Query("orderCode")

	cancelFlag := strings.EqualFold(cancelStr, "true") || cancelStr == "1"
	orderCode, err := strconv.ParseInt(orderCodeStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       "invalid orderCode",
			"orderCode":   orderCodeStr,
			"description": "orderCode must be an integer",
		})
		return
	}

	_ = codeStr

	isCancelled := strings.EqualFold(status, "CANCELLED") && cancelFlag

	//update db
	if isCancelled {
		_ = orderCode
		_ = payosID
		_ = h.payOsService.UpdatePaymentStatus(orderCode, payosID, "User cancelled on return URL")
	}

	//return to FE
	target := h.config.PayOS.FrontendCancelURL
	c.Redirect(http.StatusFound, target)
}
