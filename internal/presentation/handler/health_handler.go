package handler

import (
	"net/http"
	"core-backend/internal/infrastructure"
	"core-backend/internal/presentation/dto/response"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	infrastructureRegistry *infrastructure.InfrastructureRegistry
}

func NewHealthHandler(infrastructureRegistry *infrastructure.InfrastructureRegistry) *HealthHandler {
	return &HealthHandler{
		infrastructureRegistry: infrastructureRegistry,
	}
}

// HealthCheck godoc
// @Summary      Health Check
// @Description  Returns the health status of the application and its dependencies
// @Tags         Health
// @Accept       json
// @Produce      json
// @Success      200 {object} response.APIResponse "Service is healthy"
// @Failure      503 {object} response.APIResponse "Service is unhealthy"
// @Router       /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	healthStatus := h.infrastructureRegistry.IsHealthy()
	
	// Check if any critical service is down
	allHealthy := true
	for _, healthy := range healthStatus {
		if !healthy {
			allHealthy = false
			break
		}
	}

	status := "healthy"
	statusCode := http.StatusOK

	if !allHealthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	healthData := map[string]interface{}{
		"status":   status,
		"services": healthStatus,
	}

	response := response.SuccessResponse("Health check completed", statusCode, healthData)
	c.JSON(statusCode, response)
}

// ReadinessCheck godoc
// @Summary      Readiness Check
// @Description  Returns whether the application is ready to serve requests
// @Tags         Health
// @Accept       json
// @Produce      json
// @Success      200 {object} response.APIResponse "Service is ready"
// @Failure      503 {object} response.APIResponse "Service is not ready"
// @Router       /health/ready [get]
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	healthStatus := h.infrastructureRegistry.IsHealthy()
	
	// For readiness, we only care about critical services like database
	ready := healthStatus["database"]

	status := "ready"
	statusCode := http.StatusOK

	if !ready {
		status = "not ready"
		statusCode = http.StatusServiceUnavailable
	}

	readinessData := map[string]interface{}{
		"status":   status,
		"database": healthStatus["database"],
	}

	response := response.SuccessResponse("Readiness check completed", statusCode, readinessData)
	c.JSON(statusCode, response)
}

// LivenessCheck godoc
// @Summary      Liveness Check
// @Description  Returns whether the application is alive and running
// @Tags         Health
// @Accept       json
// @Produce      json
// @Success      200 {object} response.APIResponse "Service is alive"
// @Router       /health/live [get]
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	// Simple liveness check - if we can respond, we're alive
	livenessData := map[string]interface{}{
		"status": "alive",
		"uptime": "running", // You could calculate actual uptime here
	}

	response := response.SuccessResponse("Liveness check completed", http.StatusOK, livenessData)
	c.JSON(http.StatusOK, response)
}