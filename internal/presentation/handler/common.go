package handler

import (
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"errors"
	"fmt"
	"net/http"
	"reflect"

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

// extractUserReoles utility extracts and validate the user roles from the Gin context.
func extractUserRoles(c *gin.Context) (*string, error) {
	rolesData, exists := c.Get("roles")
	if !exists {
		return nil, errors.New("user roles not found in context")
	}
	var ok bool
	userRoles, ok := rolesData.(string)
	if !ok {
		return nil, errors.New("invalid user roles format")
	}
	if !enum.UserRole(userRoles).IsValid() {
		return nil, fmt.Errorf("invalid user role: %s", userRoles)
	}

	return &userRoles, nil
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

// extractQueryID utility extracts a UUID parameter from the query string based on the provided query name.
func extractQueryID(c *gin.Context, queryName string) (queryID uuid.UUID, err error) {
	extractedID := c.Query(queryName)
	if extractedID == "" {
		return uuid.Nil, fmt.Errorf("%s is required", queryName)
	}
	queryID, err = uuid.Parse(extractedID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s format: %v", queryName, err)
	}
	return
}

// processValidationError processes validation errors from the validator package and returns a structured APIValidationErrorResponse.
func processValidationError(err error) *responses.APIValidationErrorResponse {
	if err == nil {
		return nil
	}

	errorValue := reflect.ValueOf(err)
	switch errorValue.Type().String() {
	case "validator.ValidationErrors":
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
	case "*validator.InvalidValidationError":
		errorStr := err.Error()
		return responses.ValidationErrorResponse(400, "Invalid validation error:"+errorStr)
	default:
		errorStr := errorValue.Type().String()
		return responses.ValidationErrorResponse(400, "Unknown validation error"+errorStr)
	}

	return responses.ValidationErrorResponse(400, "Validation Error, Unable to process the validation errors")
}

// IsAllowRole use for optional role check. It's mean that in the same endpoint, depending on the user role, the response will be different.
func IsAllowRole(c *gin.Context, allowFullViewRoles []enum.UserRole) bool {
	rolePtr, _ := extractUserRoles(c)
	if rolePtr == nil {
		return false
	}
	for _, ar := range allowFullViewRoles {
		if ar == enum.UserRole(*rolePtr) {
			return true
		}
	}
	return false
}
