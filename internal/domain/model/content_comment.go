package model

import (
	"core-backend/internal/domain/enum"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContentComment struct {
	ID               uuid.UUID               `json:"id" gorm:"type:uuid;primaryKey;column:id"`
	ContentChannelID uuid.UUID               `json:"content_channel_id" gorm:"type:uuid;not null;column:content_channel_id;index"`
	Comment          string                  `json:"comment" gorm:"type:text;not null;column:comment"`
	Reactions        ContentCommentReactions `json:"reactions" gorm:"type:jsonb;column:reactions"`
	CreatedAt        *time.Time              `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	CreatedBy        *uuid.UUID              `json:"created_by" gorm:"type:uuid;not null;column:created_by"`
	UpdatedAt        *time.Time              `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
	UpdatedBy        *uuid.UUID              `json:"updated_by" gorm:"type:uuid;column:updated_by"`
	IsCensored       bool                    `json:"is_censored" gorm:"type:boolean;default:false;column:is_censored"`
	CensorReason     *string                 `json:"censor_reason,omitempty" gorm:"type:text;column:censor_reason"`

	// Relationships
	ContentChannel *ContentChannel `json:"content_channel,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE"`
	CreatedUser    *User           `json:"created_user,omitempty" gorm:"foreignKey:CreatedBy;constraint:OnDelete:SET NULL"`
}

func (ContentComment) TableName() string { return "content_comments" }

func (c *ContentComment) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.Reactions == nil {
		c.Reactions = []ContentReaction{}
	}
	return nil
}

// region: ======== ContentReaction ========

type ContentCommentReactions []ContentReaction

type ContentReaction struct {
	ID        uuid.UUID         `json:"id"`
	UserID    *uuid.UUID        `json:"user_id,omitempty"`
	ReactedAt time.Time         `json:"reacted_at"`
	Type      enum.ReactionType `json:"reaction_type"`
}

func (ccr *ContentCommentReactions) Scan(value any) error {
	if value == nil {
		*ccr = ContentCommentReactions{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan ContentReactions: %v", value)
	}

	return json.Unmarshal(bytes, ccr)
}

func (ccr ContentCommentReactions) Value() (driver.Value, error) {
	if ccr == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(ccr)
}

// endregion
