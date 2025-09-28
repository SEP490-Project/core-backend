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

// CreatePayment godoc
// @Summary Create a PayOS payment
// @Description Initiate a payment with PayOS
// @Tags payos
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "Payment Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /api/v1/payos/payment [post]
func (h *PayOsHandler) CreatePayment(c *gin.Context) {
	var req requests.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.payOsService.GeneratePaymentLink(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
