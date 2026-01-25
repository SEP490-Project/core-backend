package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type ConceptHandler struct {
	conceptService iservice.ConceptService
	validator      *validator.Validate
}

func NewConceptHandler(conceptService iservice.ConceptService) *ConceptHandler {
	return &ConceptHandler{
		conceptService: conceptService,
		validator:      validator.New(),
	}
}

// GetConcepts godoc
//
//	@Summary		Get Concepts
//	@Description	Get paginated list of concepts (and their products flattened)
//	@Tags			Concepts
//	@Accept			json
//	@Produce		json
//	@Param			page	query		int		false	"Page number"		default(1)
//	@Param			limit	query		int		false	"Items per page"	default(10)
//	@Param			search	query		string	false	"Search by name"
//	@Param			status	query		string	false	"DANGLING (DEFAULT) | ATTACHED | ALL "
//	@Success		200		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Router			/api/v1/concepts [get]
func (h *ConceptHandler) GetConcepts(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	statusStr := c.DefaultQuery("status", "DANGLING")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	search := c.DefaultQuery("search", "")

	conceptProducts, total, err := h.conceptService.GetConceptPagination(limit, page, search, &statusStr)
	if err != nil {
		zap.L().Error("failed to get concepts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get concepts: "+err.Error(), http.StatusInternalServerError))
		return
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	pagination := responses.Pagination{
		Page:       page,
		Limit:      limit,
		Total:      int64(total),
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	resp := responses.NewPaginationResponse("Concepts retrieved successfully", http.StatusOK, conceptProducts, pagination)
	c.JSON(http.StatusOK, resp)
}

// CreateConcept godoc
//
//	@Summary		Create Concept
//	@Description	Create a new concept
//	@Tags			Concepts
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.ConceptRequest	true	"Concept payload"
//	@Success		201		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/concepts [post]
func (h *ConceptHandler) CreateConcept(c *gin.Context) {
	var req requests.ConceptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}
	created, err := h.conceptService.CreateConcept(req)
	if err != nil {
		zap.L().Error("failed to create concept", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to create concept: "+err.Error(), http.StatusInternalServerError))
		return
	}
	c.JSON(http.StatusCreated, responses.SuccessResponse("Concept created", nil, created))
}

// DeleteConcept godoc
//
//	@Summary		Delete Concept
//	@Description	Delete a concept by ID
//	@Tags			Concepts
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Concept ID"
//	@Success		204	{string}	string	""
//	@Failure		400	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/concepts/{id} [delete]
func (h *ConceptHandler) DeleteConcept(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("concept id is required", http.StatusBadRequest))
		return
	}
	if err := h.conceptService.DeleteConcept(idStr); err != nil {
		zap.L().Error("failed to delete concept", zap.Error(err))
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to delete concept: "+err.Error(), http.StatusBadRequest))
		return
	}
	c.JSON(http.StatusNoContent, gin.H{})
}

// UpdateConcept godoc
//
//	@Summary		Update Concept
//	@Description	Update existed concept
//	@Tags			Concepts
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Concept ID"
//	@Param			data	body		requests.UpdateConceptRequest	true	"Concept payload"
//	@Success		201		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/concepts/{id} [put]
func (h *ConceptHandler) UpdateConcept(c *gin.Context) {
	idStr := c.Param("id")
	conceptUUID, err := uuid.Parse(idStr)
	if err != nil {
		resp := responses.ErrorResponse("invalid concept id", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	var req requests.UpdateConceptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}
	model, err := h.conceptService.UpdateConcept(conceptUUID, req)
	if err != nil {
		zap.L().Error("failed to update concept", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to update concept: "+err.Error(), http.StatusInternalServerError))
		return
	}
	c.JSON(http.StatusCreated, responses.SuccessResponse("Concept updated", nil, model))
}
