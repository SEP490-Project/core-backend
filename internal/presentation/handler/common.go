package handler

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
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
