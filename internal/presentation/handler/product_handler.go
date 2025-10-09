package handler

import (
	"core-backend/internal/application/interfaces/iservice"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type ProductHandler struct {
	productService iservice.ProductService
	validator      *validator.Validate
}

func NewProductHandler(productService iservice.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		validator:      nil,
	}
}

// GetAllProducts godoc
//
//	@Summary		Get All Products
//	@Description	Get paginated list of products with optional search
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int																		false	"Number of items per page"	default(10)
//	@Param			offset	query		int																		false	"Number of items to skip"	default(0)
//	@Param			search	query		string																	false	"Search term for product name"
//	@Success		200		{object}	object{data=[]responses.ProductResponse,total=int,limit=int,offset=int}	"Products retrieved successfully"
//	@Failure		500		{object}	object{error=string}													"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/products [get]
func (h *ProductHandler) GetAllProducts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	search := c.DefaultQuery("search", "")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	products, total, err := (h.productService).GetProductsPagination(limit, offset, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   products,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
