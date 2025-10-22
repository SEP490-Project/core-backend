package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/iservice"
	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	orderService iservice.OrderService
}

func (o *OrderHandler) placeOrder(c *gin.Context) {
	var req requests.OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

}
