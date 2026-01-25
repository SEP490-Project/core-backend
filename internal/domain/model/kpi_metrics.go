package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KPIMetrics struct {
	ID            uuid.UUID             `json:"id" gorm:"type:uuid;column:id;primaryKey;default:gen_random_uuid()"`
	ReferenceID   uuid.UUID             `json:"reference_id" gorm:"type:uuid;column:reference_id;not null"`
	ReferenceType enum.KPIReferenceType `json:"reference_type" gorm:"type:varchar(50);column:reference_type;not null;check:reference_type IN ('CONTENT', 'CAMPAIGN', 'AFFILIATE_LINK')"`
	Type          enum.KPIValueType     `json:"type" gorm:"type:varchar(50);column:type;not null;check:type IN ('REACH', 'IMPRESSIONS', 'LIKES', 'COMMENTS', 'SHARES', 'CTR', 'ENGAGEMENT')"`
	Value         float64               `json:"value" gorm:"type:decimal(15,2);column:value;not null"`
	Unit          *string               `json:"unit" gorm:"type:varchar(10);column:unit"`
	RecordedDate  time.Time             `json:"recorded_date" gorm:"type:timestamptz;column:recorded_date;current_timestamp;not null"`
}

func (KPIMetrics) TableName() string { return "kpi_metrics" }

func (k *KPIMetrics) BeforeCreate(tx *gorm.DB) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	return nil
}
