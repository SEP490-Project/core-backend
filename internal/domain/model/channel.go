package model

import (
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;column:id;primaryKey;default:gen_random_uuid()"`
	Name        string     `json:"name" gorm:"type:varchar(100);column:name;not null;unique;index"`
	Description *string    `json:"description" gorm:"type:text;column:description"`
	HomePageURL *string    `json:"home_page_url" gorm:"type:text;column:home_page_url"`
	IsActive    bool       `json:"is_active" gorm:"type:boolean;column:is_active;default:true;not null"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   *time.Time `json:"deleted_at" gorm:"column:deleted_at;index"`
}

func (Channel) TableName() string { return "channels" }

func (c *Channel) BeforeCreate() error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
