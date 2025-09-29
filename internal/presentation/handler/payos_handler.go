package handler

import (
	"core-backend/internal/application/dto/requests"
	"net/http"

	"github.com/gin-gonic/gin"

	"core-backend/internal/application/interfaces/iservice_third_party"
)

type PayOsHandler struct {
	payOsService iservice_third_party.PayOSService
}

func NewPayOsHandler(payOsService iservice_third_party.PayOSService) *PayOsHandler {
	return &PayOsHandler{payOsService: payOsService}
}

// GeneratePaymentLink godoc
// @Summary Create a PayOS payment
// @Description Initiate a payment with PayOS
// @Tags payos
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "Payment Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /api/v1/payos/payment [post]
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
// @Summary Get PayOS order info
// @Description Inspect payment detail
// @Tags payos
// @Produce json
// @Param orderCode path string true "Order Code"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /api/v1/payos/payment/{orderCode} [get]
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
