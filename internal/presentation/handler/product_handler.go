package handler

import (
	"core-backend/internal/application/interfaces/iservice"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
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

// GetProductsByTask godoc
// @Summary      Get Products By Task
// @Description  Get paginated products (overview) belonging to a task. Authorization: staff roles or owning brand user.
// @Tags         Products
// @Accept       json
// @Produce      json
// @Param        taskId path string true "Task ID (UUID)"
// @Param        limit  query int false "Number of items per page" default(10)
// @Param        offset query int false "Number of items to skip" default(0)
// @Success      200 {object} object{data=[]responses.ProductOverviewResponse,total=int,limit=int,offset=int}
// @Failure      400 {object} object{error=string}
// @Failure      403 {object} object{error=string}
// @Failure      500 {object} object{error=string}
// @Security     BearerAuth
// @Router       /api/v1/tasks/{taskId}/products [get]
func (h *ProductHandler) GetProductsByTask(c *gin.Context) {
	// Parse path param
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	// Pagination params
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Auth context
	userIDVal, ok := c.Get("user_id")
	if !ok || userIDVal == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "missing user id in context"})
		return
	}
	userIDStr, _ := userIDVal.(string)
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid user id in context"})
		return
	}
	roleVal, _ := c.Get("roles")
	roleStr, _ := roleVal.(string)

	overview, total, svcErr := h.productService.GetProductsByTask(taskID, userUUID, roleStr, limit, offset)
	if svcErr != nil {
		if strings.Contains(strings.ToLower(svcErr.Error()), "forbidden") {
			c.JSON(http.StatusForbidden, gin.H{"error": svcErr.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": svcErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   overview,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
