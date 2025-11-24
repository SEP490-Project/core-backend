package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AIHandler struct {
	aiService iservice.AIService
}

func NewAIHandler(aiService iservice.AIService) *AIHandler {
	return &AIHandler{
		aiService: aiService,
	}
}

// region: ============== General Generation ==============

// Generate handles general AI generation
//
//	@Summary		Generate AI content
//	@Description	Generates content based on a prompt
//	@Tags			AI
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.GenerateRequest	true	"Generation Request"
//	@Success		200		{object}	responses.APIResponse{data=responses.ChatResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ai/generate [post]
func (h *AIHandler) Generate(c *gin.Context) {
	var req requests.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body", http.StatusBadRequest))
		return
	}

	if req.Stream {
		h.streamGenerate(c, &req)
		return
	}

	resp, err := h.aiService.Generate(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to generate content", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Generated successfully", nil, resp))
}

func (h *AIHandler) streamGenerate(c *gin.Context, req *requests.GenerateRequest) {
	stream, err := h.aiService.Stream(c.Request.Context(), req)
	if err != nil {
		zap.L().Error("Failed to start stream", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	h.handleStream(c, stream)
}

// endregion

// region: ============== Content Generation ==============

// GenerateContent handles structured content generation (e.g. social posts)
//
//	@Summary		Generate structured content
//	@Description	Generates structured content (TipTap JSON) based on context and requirements
//	@Tags			AI
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.GenerateContentRequest	true	"Content Generation Request"
//	@Success		200		{object}	responses.APIResponse{data=responses.ChatResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ai/generate-content [post]
func (h *AIHandler) GenerateContent(c *gin.Context) {
	var req requests.GenerateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body", http.StatusBadRequest))
		return
	}

	if req.Stream {
		h.streamContent(c, &req)
		return
	}

	resp, err := h.aiService.GenerateContent(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to generate content", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Generated successfully", nil, resp))
}

func (h *AIHandler) streamContent(c *gin.Context, req *requests.GenerateContentRequest) {
	stream, err := h.aiService.StreamContent(c.Request.Context(), req)
	if err != nil {
		zap.L().Error("Failed to start stream", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	h.handleStream(c, stream)
}

func (h *AIHandler) handleStream(c *gin.Context, stream <-chan *responses.ChatResponse) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	c.Writer.Flush()

	ctx := c.Request.Context()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-stream:
			if !ok {
				return
			}
			c.SSEvent("message", msg)
			c.Writer.Flush()
		case <-ticker.C:
			c.SSEvent("heartbeat", "ping")
			c.Writer.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// endregion

// region: ============== Model Management ==============

// GetSupportedModels retrieves supported AI models from all providers
//
//	@Summary		Get supported AI models
//	@Description	Retrieves a list of supported AI models from all configured providers
//	@Tags			AI
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=[]responses.ModelProviderResponse}
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ai/models [get]
func (h *AIHandler) GetSupportedModels(c *gin.Context) {
	models, err := h.aiService.GetSupportedModels(c.Request.Context())
	if err != nil {
		zap.L().Error("Failed to get supported models", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}
	c.JSON(http.StatusOK, responses.SuccessResponse("Fetched supported models successfully", nil, models))
}

// endregion
