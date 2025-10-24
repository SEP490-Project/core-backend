package requests

type TaskFilterRequest struct {
	PaginationRequest
	CreatedByID      *string `form:"created_by_id" json:"created_by" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	AssignedToID     *string `form:"assigned_to_id" json:"assigned_to" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	MilestoneID      *string `form:"milestone_id" json:"milestone_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ContractID       *string `form:"contract_id" json:"contract_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	DeadlineFromDate *string `form:"deadline_start_date" json:"start_date" validate:"omitempty,datetime=2006-01-02" example:"2023-10-01"`
	DeadlineToDate   *string `form:"deadline_end_date" json:"end_date" validate:"omitempty,datetime=2006-01-02" example:"2023-10-31"`
	UpdatedFromDate  *string `form:"updated_start_date" json:"updated_start_date" validate:"omitempty,datetime=2006-01-02" example:"2023-10-01"`
	UpdatedToDate    *string `form:"updated_end_date" json:"updated_end_date" validate:"omitempty,datetime=2006-01-02" example:"2023-10-31"`
	Status           *string `form:"status" json:"status" validate:"omitempty,oneof=TODO IN_PROGRESS CANCELLED RECAP DONE" example:"TODO"`
	Type             *string `form:"type" json:"type" validate:"omitempty,oneof=PRODUCT CONTENT EVENT OTHER" example:"OTHER"`
}
