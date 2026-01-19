package handler

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	customvalidator "core-backend/pkg/custom_validator"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ViolationHandler handles HTTP requests for contract violation operations
type ViolationHandler struct {
	violationService     iservice.ViolationService
	stateTransferService iservice.StateTransferService
	unitOfWork           irepository.UnitOfWork
	validatorBuilder     *customvalidator.ValidatorBuilder
	validator            *validator.Validate
}

// NewViolationHandler creates a new ViolationHandler
func NewViolationHandler(
	violationService iservice.ViolationService,
	stateTransferService iservice.StateTransferService,
	unitOfWork irepository.UnitOfWork,
) *ViolationHandler {
	validatorBuilder := customvalidator.NewValidatorBuilder()

	return &ViolationHandler{
		violationService:     violationService,
		stateTransferService: stateTransferService,
		unitOfWork:           unitOfWork,
		validatorBuilder:     validatorBuilder,
		validator:            validatorBuilder.Validate,
	}
}

// InitiateBrandViolation godoc
//
//	@Summary		Initiate brand violation
//	@Description	Creates a brand violation record and transitions contract to BRAND_VIOLATED status
//	@Tags			Contract Violations
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string													true	"Contract ID"
//	@Param			data	body		requests.InitiateViolationRequest						true	"Violation data"
//	@Success		201		{object}	responses.APIResponse{data=responses.ViolationResponse}	"Violation created successfully"
//	@Failure		400		{object}	responses.APIResponse									"Invalid request"
//	@Failure		401		{object}	responses.APIResponse									"Unauthorized"
//	@Failure		404		{object}	responses.APIResponse									"Contract not found"
//	@Failure		409		{object}	responses.APIResponse									"Active violation already exists"
//	@Failure		500		{object}	responses.APIResponse									"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id}/report-brand-violation [post]
func (h *ViolationHandler) InitiateBrandViolation(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}
	contractID, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest))
		return
	}

	var req requests.InitiateViolationRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}
	if errors := h.validatorBuilder.Check(&req); len(errors) > 0 {
		c.JSON(http.StatusBadRequest, responses.ValidationErrorResponse(http.StatusBadRequest, "Invalid request body", errors...))
		return
	}

	ctx := c.Request.Context()

	// 1. Create violation record (Service handles transaction and state transitions)
	v, err := h.violationService.InitiateBrandViolation(ctx, contractID, userID, req.Reason)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errMsg := err.Error()
		switch errMsg {
		case "contract not found":
			statusCode = http.StatusNotFound
		case "active violation already exists for this contract":
			statusCode = http.StatusConflict
		case "contract must be ACTIVE to initiate violation":
			statusCode = http.StatusBadRequest
		}
		zap.L().Error("Failed to initiate brand violation", zap.Error(err))
		c.JSON(statusCode, responses.ErrorResponse("Failed to initiate violation: "+errMsg, statusCode))
		return
	}

	violation := responses.ViolationResponse{}.ToViolationResponse(v)
	c.JSON(http.StatusCreated, responses.SuccessResponse("Brand violation initiated", nil, violation))
}

// InitiateKOLViolation godoc
//
//	@Summary		Initiate KOL violation
//	@Description	Creates a KOL violation record and transitions contract to KOL_VIOLATED status
//	@Tags			Contract Violations
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string													true	"Contract ID"
//	@Param			data	body		requests.InitiateViolationRequest						true	"Violation data"
//	@Success		201		{object}	responses.APIResponse{data=responses.ViolationResponse}	"Violation created successfully"
//	@Failure		400		{object}	responses.APIResponse									"Invalid request"
//	@Failure		401		{object}	responses.APIResponse									"Unauthorized"
//	@Failure		404		{object}	responses.APIResponse									"Contract not found"
//	@Failure		409		{object}	responses.APIResponse									"Active violation already exists"
//	@Failure		500		{object}	responses.APIResponse									"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id}/report-kol-violation [post]
func (h *ViolationHandler) InitiateKOLViolation(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	contractID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest))
		return
	}

	var req requests.InitiateViolationRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}
	if errors := h.validatorBuilder.Check(&req); len(errors) > 0 {
		c.JSON(http.StatusBadRequest, responses.ValidationErrorResponse(http.StatusBadRequest, "Invalid request body", errors...))
		return
	}

	ctx := c.Request.Context()

	// Service handles transaction and state transitions
	v, err := h.violationService.InitiateKOLViolation(ctx, contractID, userID, req.Reason)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errMsg := err.Error()
		switch errMsg {
		case "contract not found":
			statusCode = http.StatusNotFound
		case "active violation already exists for this contract":
			statusCode = http.StatusConflict
		case "contract must be ACTIVE to initiate violation":
			statusCode = http.StatusBadRequest
		}
		zap.L().Error("Failed to initiate KOL violation", zap.Error(err))
		c.JSON(statusCode, responses.ErrorResponse("Failed to initiate violation: "+errMsg, statusCode))
		return
	}

	violation := responses.ViolationResponse{}.ToViolationResponse(v)
	c.JSON(http.StatusCreated, responses.SuccessResponse("KOL violation initiated", nil, violation))
}

// CreatePenaltyPayment godoc
//
//	@Summary		Create penalty payment link
//	@Description	Creates a PayOS payment link for brand penalty and transitions to BRAND_PENALTY_PENDING
//	@Tags			Contract Violations
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string													true	"Contract ID"
//	@Success		200	{object}	responses.APIResponse{data=responses.PayOSLinkResponse}	"Payment link created"
//	@Failure		400	{object}	responses.APIResponse									"Invalid request"
//	@Failure		401	{object}	responses.APIResponse									"Unauthorized"
//	@Failure		404	{object}	responses.APIResponse									"Violation not found"
//	@Failure		500	{object}	responses.APIResponse									"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id}/violation/create-penalty-payment [post]
func (h *ViolationHandler) CreatePenaltyPayment(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}
	contractID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest))
		return
	}
	ctx := c.Request.Context()
	var request requests.CreatePenaltyPaymentRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}

	if request.ViolationID == nil || *request.ViolationID == uuid.Nil {
		var violation *model.ContractViolation
		violation, err = h.violationService.GetByContractID(ctx, contractID)
		if err != nil {
			statusCode := http.StatusInternalServerError
			if err.Error() == "no active violation found for contract" {
				statusCode = http.StatusNotFound
			}
			c.JSON(statusCode, responses.ErrorResponse("Failed to get violation: "+err.Error(), statusCode))
			return
		}
		request.ViolationID = &violation.ID
	}

	paymentLink, err := h.violationService.CreatePenaltyPayment(ctx, userID, &request)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errMsg := err.Error()
		if errMsg == "violation not found" {
			statusCode = http.StatusNotFound
		}
		zap.L().Error("Failed to create penalty payment", zap.Error(err))
		c.JSON(statusCode, responses.ErrorResponse("Failed to create payment link: "+errMsg, statusCode))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Penalty payment link created", nil, paymentLink))
}

// SubmitRefundProof godoc
//
//	@Summary		Submit refund proof
//	@Description	Marketing Staff submits proof of refund for Brand approval
//	@Tags			Contract Violations
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string													true	"Contract ID"
//	@Param			data	body		requests.SubmitRefundProofRequest						true	"Proof data"
//	@Success		200		{object}	responses.APIResponse{data=responses.ViolationResponse}	"Proof submitted"
//	@Failure		400		{object}	responses.APIResponse									"Invalid request"
//	@Failure		401		{object}	responses.APIResponse									"Unauthorized"
//	@Failure		404		{object}	responses.APIResponse									"Violation not found"
//	@Failure		500		{object}	responses.APIResponse									"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id}/violation/submit-proof [post]
func (h *ViolationHandler) SubmitRefundProof(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	contractID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest))
		return
	}

	var req requests.SubmitRefundProofRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}
	if errors := h.validatorBuilder.Check(&req); len(errors) > 0 {
		c.JSON(http.StatusBadRequest, responses.ValidationErrorResponse(http.StatusBadRequest, "Invalid request body", errors...))
		return
	}

	ctx := c.Request.Context()

	// Find violation by contract ID
	vModel, err := h.violationService.GetByContractID(ctx, contractID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "no active violation found for contract" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, responses.ErrorResponse("Failed to get violation: "+err.Error(), statusCode))
		return
	}

	var violation *responses.ViolationResponse
	err = helper.WithTransaction(ctx, h.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		var v *model.ContractViolation
		v, err = h.violationService.SubmitRefundProof(ctx, vModel.ID, req.ProofURL, req.Message, userID)
		if err != nil {
			return err
		}

		// Transition contract to KOL_PROOF_SUBMITTED
		if err = h.stateTransferService.MoveContractToState(ctx, uow, v.ContractID, enum.ContractStatusKOLProofSubmitted, userID); err != nil {
			return err
		}

		violation = responses.ViolationResponse{}.ToViolationResponse(v)
		return nil
	})

	if err != nil {
		statusCode := http.StatusInternalServerError
		errMsg := err.Error()
		if errMsg == "violation not found" {
			statusCode = http.StatusNotFound
		}
		zap.L().Error("Failed to submit refund proof", zap.Error(err))
		c.JSON(statusCode, responses.ErrorResponse("Failed to submit proof: "+errMsg, statusCode))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Refund proof submitted", nil, violation))
}

// ReviewRefundProof godoc
//
//	@Summary		Review refund proof
//	@Description	Brand reviews KOL's refund proof (approve/reject)
//	@Tags			Contract Violations
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string													true	"Contract ID"
//	@Param			data	body		requests.ReviewRefundProofRequest						true	"Review data"
//	@Success		200		{object}	responses.APIResponse{data=responses.ViolationResponse}	"Proof reviewed"
//	@Failure		400		{object}	responses.APIResponse									"Invalid request"
//	@Failure		401		{object}	responses.APIResponse									"Unauthorized"
//	@Failure		404		{object}	responses.APIResponse									"Violation not found"
//	@Failure		500		{object}	responses.APIResponse									"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id}/violation/review-proof [post]
func (h *ViolationHandler) ReviewRefundProof(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	contractID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest))
		return
	}

	var req requests.ReviewRefundProofRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}
	if errors := h.validatorBuilder.Check(&req); len(errors) > 0 {
		c.JSON(http.StatusBadRequest, responses.ValidationErrorResponse(http.StatusBadRequest, "Invalid request body", errors...))
		return
	}

	ctx := c.Request.Context()

	// Find violation by contract ID
	vModel, err := h.violationService.GetByContractID(ctx, contractID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "no active violation found for contract" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, responses.ErrorResponse("Failed to get violation: "+err.Error(), statusCode))
		return
	}

	var violation *responses.ViolationResponse
	err = helper.WithTransaction(ctx, h.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		var v *model.ContractViolation
		v, err = h.violationService.ReviewRefundProof(ctx, vModel.ID, &req, userID)
		if err != nil {
			return err
		}

		// Transition contract based on review result
		var targetState enum.ContractStatus
		if req.IsApprove() {
			targetState = enum.ContractStatusKOLRefundApproved
		} else {
			targetState = enum.ContractStatusKOLProofRejected
		}

		if err = h.stateTransferService.MoveContractToState(ctx, uow, v.ContractID, targetState, userID); err != nil {
			return err
		}

		violation = responses.ViolationResponse{}.ToViolationResponse(v)
		return nil
	})

	if err != nil {
		statusCode := http.StatusInternalServerError
		errMsg := err.Error()
		if errMsg == "violation not found" {
			statusCode = http.StatusNotFound
		}
		zap.L().Error("Failed to review refund proof", zap.Error(err))
		c.JSON(statusCode, responses.ErrorResponse("Failed to review proof: "+errMsg, statusCode))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Refund proof reviewed", nil, violation))
}

// GetViolation godoc
//
//	@Summary		Get violation details
//	@Description	Retrieves detailed information about a contract violation
//	@Tags			Contract Violations
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string													true	"Violation ID"
//	@Success		200	{object}	responses.APIResponse{data=responses.ViolationResponse}	"Violation details"
//	@Failure		400	{object}	responses.APIResponse									"Invalid violation ID"
//	@Failure		401	{object}	responses.APIResponse									"Unauthorized"
//	@Failure		404	{object}	responses.APIResponse									"Violation not found"
//	@Security		BearerAuth
//	@Router			/api/v1/violations/{id} [get]
func (h *ViolationHandler) GetViolation(c *gin.Context) {
	violationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid violation ID", http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	violation, err := h.violationService.GetByID(ctx, violationID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "violation not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, responses.ErrorResponse("Failed to get violation: "+err.Error(), statusCode))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Violation retrieved", nil, responses.ViolationResponse{}.ToViolationResponse(violation)))
}

// GetViolationByContract godoc
//
//	@Summary		Get violation by contract
//	@Description	Retrieves active violation for a contract
//	@Tags			Contract Violations
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string													true	"Contract ID"
//	@Success		200	{object}	responses.APIResponse{data=responses.ViolationResponse}	"Violation details"
//	@Failure		400	{object}	responses.APIResponse									"Invalid contract ID"
//	@Failure		401	{object}	responses.APIResponse									"Unauthorized"
//	@Failure		404	{object}	responses.APIResponse									"No active violation found"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id}/violation [get]
func (h *ViolationHandler) GetViolationByContract(c *gin.Context) {
	contractID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	violation, err := h.violationService.GetByContractID(ctx, contractID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "no active violation found for contract" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, responses.ErrorResponse("Failed to get violation: "+err.Error(), statusCode))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Violation retrieved", nil, responses.ViolationResponse{}.ToViolationResponse(violation)))
}

// ListViolations godoc
//
//	@Summary		List violations
//	@Description	Retrieves a paginated list of contract violations with filtering
//	@Tags			Contract Violations
//	@Accept			json
//	@Produce		json
//	@Param			type			query		string															false	"Filter by violation type (BRAND/KOL)"
//	@Param			is_resolved		query		bool															false	"Filter by resolution status"
//	@Param			proof_status	query		string															false	"Filter by proof status"
//	@Param			contract_id		query		string															false	"Filter by contract ID"
//	@Param			campaign_id		query		string															false	"Filter by campaign ID"
//	@Param			page			query		int																false	"Page number (default: 1)"
//	@Param			page_size		query		int																false	"Page size (default: 10, max: 100)"
//	@Success		200				{object}	responses.APIResponse{data=[]responses.ViolationListResponse}	"Violations list"
//	@Failure		401				{object}	responses.APIResponse											"Unauthorized"
//	@Failure		500				{object}	responses.APIResponse											"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/violations [get]
func (h *ViolationHandler) ListViolations(c *gin.Context) {
	var filter requests.ViolationFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	// Set defaults
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 10
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	ctx := c.Request.Context()
	violations, total, err := h.violationService.List(ctx, &filter)
	if err != nil {
		zap.L().Error("Failed to list violations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to list violations: "+err.Error(), http.StatusInternalServerError))
		return
	}

	totalPages := (int(total) + filter.PageSize - 1) / filter.PageSize
	pagination := responses.Pagination{
		Page:       filter.Page,
		Limit:      filter.PageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    filter.Page < totalPages,
		HasPrev:    filter.Page > 1,
	}

	c.JSON(http.StatusOK, responses.NewPaginationResponse("Violations retrieved", http.StatusOK, violations, pagination))
}

// CalculateBrandPenalty godoc
//
//	@Summary		Calculate brand penalty
//	@Description	Calculates the penalty amount for a brand violation
//	@Tags			Contract Violations
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string																true	"Contract ID"
//	@Success		200	{object}	responses.APIResponse{data=responses.ViolationCalculationResponse}	"Calculation result"
//	@Failure		400	{object}	responses.APIResponse												"Invalid contract ID"
//	@Failure		401	{object}	responses.APIResponse												"Unauthorized"
//	@Failure		404	{object}	responses.APIResponse												"Contract not found"
//	@Failure		500	{object}	responses.APIResponse												"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id}/violation/calculate/brand [get]
func (h *ViolationHandler) CalculateBrandPenalty(c *gin.Context) {
	contractID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	calculation, err := h.violationService.CalculateBrandPenalty(ctx, contractID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "contract not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, responses.ErrorResponse("Failed to calculate penalty: "+err.Error(), statusCode))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Brand penalty calculated", nil, calculation))
}

// CalculateKOLRefund godoc
//
//	@Summary		Calculate KOL refund
//	@Description	Calculates the refund amount for a KOL violation
//	@Tags			Contract Violations
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string																true	"Contract ID"
//	@Success		200	{object}	responses.APIResponse{data=responses.ViolationCalculationResponse}	"Calculation result"
//	@Failure		400	{object}	responses.APIResponse												"Invalid contract ID"
//	@Failure		401	{object}	responses.APIResponse												"Unauthorized"
//	@Failure		404	{object}	responses.APIResponse												"Contract not found"
//	@Failure		500	{object}	responses.APIResponse												"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id}/violation/calculate/kol [get]
func (h *ViolationHandler) CalculateKOLRefund(c *gin.Context) {
	contractID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	calculation, err := h.violationService.CalculateKOLRefund(ctx, contractID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "contract not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, responses.ErrorResponse("Failed to calculate refund: "+err.Error(), statusCode))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("KOL refund calculated", nil, calculation))
}

// ResolveViolation godoc
//
//	@Summary		Resolve violation
//	@Description	Marks a violation as resolved and terminates the contract
//	@Tags			Contract Violations
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"Contract ID"
//	@Success		200	{object}	responses.APIResponse	"Violation resolved"
//	@Failure		400	{object}	responses.APIResponse	"Invalid contract ID"
//	@Failure		401	{object}	responses.APIResponse	"Unauthorized"
//	@Failure		404	{object}	responses.APIResponse	"Violation not found"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id}/violation/resolve [post]
func (h *ViolationHandler) ResolveViolation(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	contractID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()

	// Find violation by contract ID
	violation, err := h.violationService.GetByContractID(ctx, contractID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "no active violation found for contract" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, responses.ErrorResponse("Failed to get violation: "+err.Error(), statusCode))
		return
	}

	err = helper.WithTransaction(ctx, h.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// Resolve the violation
		if err = h.violationService.ResolveViolation(ctx, violation.ID, userID); err != nil {
			return err
		}

		// Terminate the contract
		if err = h.stateTransferService.MoveContractToState(ctx, uow, violation.ContractID, enum.ContractStatusTerminated, userID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		zap.L().Error("Failed to resolve violation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to resolve violation: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Violation resolved and contract terminated", nil, nil))
}
