package dtos

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// TaskDetailDTO represents the detailed information of a task joined with related entities.
type TaskDetailDTO struct {
	ID             uuid.UUID       `json:"id" gorm:"column:id"`
	Name           string          `json:"name" gorm:"column:name"`
	Description    datatypes.JSON  `json:"description" gorm:"column:description"`
	Deadline       time.Time       `json:"deadline" gorm:"column:deadline"`
	Type           enum.TaskType   `json:"type" gorm:"column:type"`
	Status         enum.TaskStatus `json:"status" gorm:"column:status"`
	AssignedToID   *uuid.UUID      `json:"assigned_to_id,omitempty" gorm:"column:assigned_to_id"`
	AssignedToName *string         `json:"assigned_to_name,omitempty" gorm:"column:assigned_to_name"`
	AssignedToRole *enum.UserRole  `json:"assigned_to_role,omitempty" gorm:"column:assigned_to_role"`
	CreatedAt      time.Time       `json:"created_at" gorm:"column:created_at"`
	CreatedByID    uuid.UUID       `json:"created_by_id" gorm:"column:created_by_id"`
	CreatedByName  string          `json:"created_by_name" gorm:"column:created_by_name"`
	CreatedByRole  enum.UserRole   `json:"created_by_role" gorm:"column:created_by_role"`
	UpdatedAt      time.Time       `json:"updated_at" gorm:"column:updated_at"`
	UpdatedByID    uuid.UUID       `json:"updated_by_id" gorm:"column:updated_by_id"`
	UpdatedByName  string          `json:"updated_by_name" gorm:"column:updated_by_name"`
	UpdatedByRole  enum.UserRole   `json:"updated_by_role" gorm:"column:updated_by_role"`
	MilestoneID    *uuid.UUID      `json:"milestone_id" gorm:"column:milestone_id"`
	MilestoneInfo  datatypes.JSON  `json:"milestone_info,omitempty" gorm:"column:milestone_info"`
	CampaignID     *uuid.UUID      `json:"campaign_id" gorm:"column:campaign_id"`
	CampaignInfo   datatypes.JSON  `json:"campaign_info,omitempty" gorm:"column:campaign_info"`
	ContractID     *uuid.UUID      `json:"contract_id" gorm:"column:contract_id"`

	// Aggregated fields
	ContentInfos []ContentInfo `json:"content_info,omitempty" gorm:"-"`
	ProductInfos []ProductInfo `json:"product_info,omitempty" gorm:"-"`
	BrandInfo    *BrandInfo    `json:"brand_info,omitempty" gorm:"-"`
}

// TaskListDTO represents a summarized view of a task for listing purposes.
type TaskListDTO struct {
	ID             uuid.UUID           `json:"id" gorm:"column:id"`
	Name           string              `json:"name" gorm:"column:name"`
	Deadline       time.Time           `json:"deadline" gorm:"column:deadline"`
	Type           enum.TaskType       `json:"type" gorm:"column:type"`
	Status         enum.TaskStatus     `json:"status" gorm:"column:status"`
	AssignedToID   *uuid.UUID          `json:"assigned_to_id,omitempty" gorm:"column:assigned_to_id"`
	AssignedToName *string             `json:"assigned_to_name,omitempty" gorm:"column:assigned_to_name"`
	AssignedToRole *enum.UserRole      `json:"assigned_to_role,omitempty" gorm:"column:assigned_to_role"`
	CreatedAt      time.Time           `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      time.Time           `json:"updated_at" gorm:"column:updated_at"`
	MilestoneID    *uuid.UUID          `json:"milestone_id" gorm:"column:milestone_id"`
	CampaignID     *uuid.UUID          `json:"campaign_id" gorm:"column:campaign_id"`
	CampaignName   *string             `json:"campaign_name,omitempty" gorm:"column:campaign_name"`
	ContractID     *uuid.UUID          `json:"contract_id" gorm:"column:contract_id"`
	ProductID      *uuid.UUID          `json:"product_id" gorm:"column:product_id"`
	ChildStatus    *enum.ProductStatus `json:"child_status" gorm:"column:child_status"`
}

type ProductInfo struct {
	ID   uuid.UUID        `json:"id"`
	Name string           `json:"name"`
	Type enum.ProductType `json:"type"`
}

type ContentInfo struct {
	ID          uuid.UUID        `json:"id"`
	Title       string           `json:"title"`
	Description *string          `json:"description,omitempty"`
	Type        enum.ContentType `json:"type"`
}

type BrandInfo struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	LogoURL *string `json:"logo_url"`
	Status  string  `json:"status"`
}
