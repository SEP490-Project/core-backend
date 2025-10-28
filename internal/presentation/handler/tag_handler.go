package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type TagHandler struct {
	tagService iservice.TagService
	unitOfWork irepository.UnitOfWork
	validator  *validator.Validate
}

func NewTagHandler(tagService iservice.TagService, unitOfWork irepository.UnitOfWork) *TagHandler {
	validator := validator.New()
	return &TagHandler{
		tagService: tagService,
		unitOfWork: unitOfWork,
		validator:  validator,
	}
}

// Create godoc
//
//	@Summary		Create new tag
//	@Description	Create a new tag
//	@Tags			Tags
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.CreateTagRequest							true	"Tag creation data"
//	@Success		201		{object}	responses.APIResponse{data=responses.TagResponse}	"Tag created successfully"
//	@Failure		400		{object}	responses.APIResponse								"Invalid request or validation error"
//	@Failure		401		{object}	responses.APIResponse								"Unauthorized"
//	@Failure		409		{object}	responses.APIResponse								"Tag name already exists"
//	@Failure		500		{object}	responses.APIResponse								"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tags [post]
func (h *TagHandler) Create(c *gin.Context) {
	creatingUserID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}
	var request requests.CreateTagRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request payload: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err = h.validator.Struct(request); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}
	request.CreatedByID = utils.PtrOrNil(creatingUserID.String())

	uow := h.unitOfWork.Begin(c.Request.Context())

	response, err := h.tagService.Create(c, uow, &request)
	if err != nil {
		uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to create tag: "+err.Error(), http.StatusInternalServerError))
		return
	}

	uow.Commit()
	c.JSON(http.StatusOK, responses.SuccessResponse("Tag created successfully", utils.PtrOrNil(http.StatusOK), response))
}

// GetByID godoc
//
//	@Summary		Get tag by ID
//	@Description	Retrieve detailed information about a specific tag
//	@Tags			Tags
//	@Accept			json
//	@Produce		json
//	@Param			tag_id	path		string												true	"Tag ID"	format(uuid)
//	@Success		200		{object}	responses.APIResponse{data=responses.TagResponse}	"Tag retrieved successfully"
//	@Failure		400		{object}	responses.APIResponse								"Invalid tag ID"
//	@Failure		401		{object}	responses.APIResponse								"Unauthorized"
//	@Failure		404		{object}	responses.APIResponse								"Tag not found"
//	@Failure		500		{object}	responses.APIResponse								"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tags/{tag_id} [get]
func (h *TagHandler) GetByID(c *gin.Context) {
	tagID, err := extractParamID(c, "tag_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid tag ID: "+err.Error(), http.StatusBadRequest))
		return
	}

	response, err := h.tagService.GetByID(c.Request.Context(), tagID)
	if err != nil {
		var response *responses.APIResponse
		var statusCode int
		switch err.Error() {
		case "tag not found":
			response = responses.ErrorResponse("Tag not found", http.StatusNotFound)
			statusCode = http.StatusNotFound
		default:
			response = responses.ErrorResponse("Failed to get tag: "+err.Error(), http.StatusInternalServerError)
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, response)
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Tag retrieved successfully", utils.PtrOrNil(http.StatusOK), response))
}

// GetByName godoc
//
//	@Summary		Get tag by name
//	@Description	Retrieve a tag by its name
//	@Tags			Tags
//	@Accept			json
//	@Produce		json
//	@Param			name	path		string												true	"Tag name"	format(string)
//	@Success		200		{object}	responses.APIResponse{data=responses.TagResponse}	"Tag retrieved successfully"
//	@Failure		400		{object}	responses.APIResponse								"Invalid query parameters"
//	@Failure		401		{object}	responses.APIResponse								"Unauthorized"
//	@Failure		404		{object}	responses.APIResponse								"Tag not found"
//	@Failure		500		{object}	responses.APIResponse								"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tags/name/{name} [get]
func (h *TagHandler) GetByName(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Tag name is required", http.StatusBadRequest))
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())
	response, err := h.tagService.GetByName(c.Request.Context(), uow, name, userID)
	if err != nil {
		uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get tag: "+err.Error(), http.StatusInternalServerError))
		return
	}

	uow.Commit()
	c.JSON(http.StatusOK, responses.SuccessResponse("Tag retrieved successfully", utils.PtrOrNil(http.StatusOK), response))
}

// UpdateByID godoc
//
//	@Summary		Update tag
//	@Description	Update a tag by its ID
//	@Tags			Tags
//	@Accept			json
//	@Produce		json
//	@Param			tag_id	path		string												true	"Tag ID"	format(uuid)
//	@Param			request	body		requests.UpdateTagRequest							true	"Tag update data"
//	@Success		200		{object}	responses.APIResponse{data=responses.TagResponse}	"Tag updated successfully"
//	@Failure		400		{object}	responses.APIResponse								"Invalid request or validation error"
//	@Failure		401		{object}	responses.APIResponse								"Unauthorized"
//	@Failure		404		{object}	responses.APIResponse								"Tag not found"
//	@Failure		409		{object}	responses.APIResponse								"Tag name already exists"
//	@Failure		500		{object}	responses.APIResponse								"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tags/{tag_id} [put]
func (h *TagHandler) UpdateByID(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}
	tagID, err := extractParamID(c, "tag_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid tag ID: "+err.Error(), http.StatusBadRequest))
		return
	}
	var request requests.UpdateTagRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request payload: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err = h.validator.Struct(request); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}
	request.ID = utils.PtrOrNil(tagID.String())
	request.UpdatedByID = utils.PtrOrNil(userID.String())

	uow := h.unitOfWork.Begin(c.Request.Context())
	response, err := h.tagService.UpdateByID(c.Request.Context(), uow, &request)
	if err != nil {
		uow.Rollback()
		var response *responses.APIResponse
		var statusCode int
		switch err.Error() {
		case "tag not found":
			response = responses.ErrorResponse("Tag not found", http.StatusBadRequest)
			statusCode = http.StatusBadRequest
		default:
			response = responses.ErrorResponse("Failed to update tag: "+err.Error(), http.StatusInternalServerError)
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, response)
		return
	}

	uow.Commit()
	c.JSON(http.StatusOK, responses.SuccessResponse("Tag updated successfully", utils.PtrOrNil(http.StatusOK), response))
}

// DeleteByID godoc
//
//	@Summary		Delete tag
//	@Description	Soft delete a tag by ID
//	@Tags			Tags
//	@Accept			json
//	@Produce		json
//	@Param			tag_id	path		string					true	"Tag ID"	format(uuid)
//	@Success		200		{object}	responses.APIResponse	"Tag deleted successfully"
//	@Failure		400		{object}	responses.APIResponse	"Invalid tag ID"
//	@Failure		401		{object}	responses.APIResponse	"Unauthorized"
//	@Failure		404		{object}	responses.APIResponse	"Tag not found"
//	@Failure		500		{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tags/{tag_id} [delete]
func (h *TagHandler) DeleteByID(c *gin.Context) {
	tagID, err := extractParamID(c, "tag_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid tag ID: "+err.Error(), http.StatusBadRequest))
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())
	err = h.tagService.DeleteByID(c.Request.Context(), uow, tagID)
	if err != nil {
		uow.Rollback()
		var response *responses.APIResponse
		var statusCode int
		switch err.Error() {
		case "tag not found":
			response = responses.ErrorResponse("Tag not found", http.StatusBadRequest)
			statusCode = http.StatusBadRequest
		default:
			response = responses.ErrorResponse("Failed to delete tag: "+err.Error(), http.StatusInternalServerError)
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, response)
		return
	}

	uow.Commit()
	c.JSON(http.StatusOK, responses.SuccessResponse("Tag deleted successfully", utils.PtrOrNil(http.StatusOK), nil))
}

// GetByFilter godoc
//
//	@Summary		Get tags by filter
//	@Description	Retrieve a paginated list of tags with optional filters
//	@Tags			Tags
//	@Accept			json
//	@Produce		json
//	@Param			keyword		query		string							false	"Tag name"			format(string)
//	@Param			page		query		int								false	"Page number"		default(1)
//	@Param			limit		query		int								false	"Items per page"	default(10)
//	@Param			sort_by		query		string							false	"Sort by field"		default(created_at)
//	@Param			sort_order	query		string							false	"Sort order"		Enums(asc, desc)	default(desc)
//	@Success		200			{object}	responses.TagPaginationResponse	"Tags retrieved successfully"
//	@Failure		400			{object}	responses.APIResponse			"Invalid query parameters"
//	@Failure		401			{object}	responses.APIResponse			"Unauthorized"
//	@Failure		500			{object}	responses.APIResponse			"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tags [get]
func (h *TagHandler) GetByFilter(c *gin.Context) {
	var filterRequest requests.TagFilterRequest
	if err := c.ShouldBindQuery(&filterRequest); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err := h.validator.Struct(&filterRequest); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}

	responsesList, totalCount, err := h.tagService.GetByFilter(c.Request.Context(), &filterRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get tags: "+err.Error(), http.StatusInternalServerError))
		return
	}

	response := responses.NewPaginationResponse(
		"Tags retrieved successfully",
		http.StatusOK,
		responsesList,
		responses.Pagination{
			Page:  filterRequest.Page,
			Limit: filterRequest.Limit,
			Total: totalCount,
		},
	)
	c.JSON(http.StatusOK, response)
}
