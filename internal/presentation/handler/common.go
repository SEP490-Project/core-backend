package handler

import (
	"core-backend/internal/application/dto/responses"
	"core-backend/pkg/utils"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// extractUserID utility extracts the user ID from the Gin context.
func extractUserID(c *gin.Context) (userID uuid.UUID, err error) {
	userIDData, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("user ID not found in context")

	}
	userIDStr, ok := userIDData.(string)
	if !ok {
		return uuid.Nil, errors.New("invalid user ID format")
	}
	userID, err = uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, err
	}

	return
}

// extractParamID utility extracts a UUID parameter from the path param based on the provided parameter name.
// For example, if the path is /api/v1/campaigns/{id}, and the paramName is "id", it will extract the UUID from the path.
// If paramName is empty, it defaults to "id".
func extractParamID(c *gin.Context, paramName string) (paramID uuid.UUID, err error) {
	if paramName == "" {
		paramName = "id"
	}
	extractedID := c.Param(paramName)
	if extractedID == "" {
		return uuid.Nil, fmt.Errorf("%s is required", paramName)
	}
	paramID, err = uuid.Parse(extractedID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s format: %v", paramName, err)
	}
	return
}

// processValidationError processes validation errors from the validator package and returns a structured APIValidationErrorResponse.
func processValidationError(err error) *responses.APIValidationErrorResponse {
	if err == nil {
		return nil
	}

	ve, ok := err.(validator.ValidationErrors)
	if ok {
		details := make([]responses.ValidationErrorDetail, 0, len(ve))
		for _, fe := range ve {
			// The param field of the ValidationError are not usually used, so it is used as a workaround for custom error messages
			msg := fe.Param()
			if msg == "" {
				msg = fmt.Sprintf("%s failed on '%s' validation", fe.Field(), fe.Tag())
			}

			details = append(details, responses.ValidationErrorDetail{
				JSONField:   fe.Field(),
				StructField: fe.StructField(),
				Value:       utils.ToString(fe.Value()),
				Message:     msg,
			})
		}

		return responses.ValidationErrorResponse(http.StatusBadRequest, "Validation error", details...)
	}
	return responses.ValidationErrorResponse(400, "Validation Error, Unable to process the validation errors")
}
