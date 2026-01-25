package handler

import (
	"net/http"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type BlogHandler struct {
	blogService iservice.BlogService
	unitOfWork  irepository.UnitOfWork
	*validator.Validate
}

func NewBlogHandler(blogService iservice.BlogService, unitOfWork irepository.UnitOfWork) *BlogHandler {
	return &BlogHandler{
		blogService: blogService,
		unitOfWork:  unitOfWork,
		Validate:    validator.New(),
	}
}

// UpdateBlogDetails updates blog-specific attributes
//
//	@Summary		Update blog details
//	@Description	Updates blog-specific attributes (tags, excerpt, read_time) for POST type content
//	@Tags			Content/Blog
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Content ID (UUID)"
//	@Param			request	body		requests.UpdateBlogRequest	true	"Blog update data"
//	@Success		200		{object}	responses.APIResponse		"Blog details updated successfully"
//	@Failure		400		{object}	responses.APIResponse		"Invalid request or content is not POST type"
//	@Failure		401		{object}	responses.APIResponse		"Authentication required"
//	@Failure		403		{object}	responses.APIResponse		"Insufficient permissions"
//	@Failure		404		{object}	responses.APIResponse		"Content or blog not found"
//	@Failure		500		{object}	responses.APIResponse		"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contents/{id}/blog [put]
func (h *BlogHandler) UpdateBlogDetails(c *gin.Context) {
	// Parse content ID from URL parameter
	contentID, err := extractParamID(c, "id")
	if err != nil {
		response := responses.ErrorResponse("Invalid content ID format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Bind request
	var req requests.UpdateBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("Failed to bind request", zap.Error(err))
		response := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())

	// Call service to update blog details
	if err := h.blogService.UpdateBlogDetails(c.Request.Context(), uow, contentID, &req); err != nil {
		uow.Rollback()
		zap.L().Error("Failed to update blog details", zap.String("content_id", contentID.String()), zap.Error(err))

		errMsg := err.Error()
		switch errMsg {
		case "content not found":
			response := responses.ErrorResponse("Content not found", http.StatusNotFound)
			c.JSON(http.StatusNotFound, response)
			return
		case "blog not found for this content":
			response := responses.ErrorResponse("Blog not found for this content", http.StatusNotFound)
			c.JSON(http.StatusNotFound, response)
			return
		case "blog operations are only allowed for POST type content":
			response := responses.ErrorResponse("Blog operations are only allowed for POST type content", http.StatusBadRequest)
			c.JSON(http.StatusBadRequest, response)
			return
		default:
			response := responses.ErrorResponse("Failed to update blog details", http.StatusInternalServerError)
			c.JSON(http.StatusInternalServerError, response)
			return
		}
	}

	uow.Commit()

	response := responses.SuccessResponse("Blog details updated successfully", nil, nil)
	c.JSON(http.StatusOK, response)
}
