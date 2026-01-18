package asynqtask

import (
	"time"

	"github.com/google/uuid"
)

// LimitedProductAnnouncementType represents the type of announcement
type LimitedProductAnnouncementType string

const (
	// AnnouncementTypePremiereDate3Days is 3 days before premiere date
	AnnouncementTypePremiereDate3Days LimitedProductAnnouncementType = "PREMIERE_3_DAYS"
	// AnnouncementTypePremiereDate1Day is 1 day before premiere date
	AnnouncementTypePremiereDate1Day LimitedProductAnnouncementType = "PREMIERE_1_DAY"
	// AnnouncementTypeAvailability3Days is 3 days before availability start date
	AnnouncementTypeAvailability3Days LimitedProductAnnouncementType = "AVAILABILITY_3_DAYS"
	// AnnouncementTypeAvailability1Day is 1 day before availability start date
	AnnouncementTypeAvailability1Day LimitedProductAnnouncementType = "AVAILABILITY_1_DAY"
)

// LimitedProductAnnouncementPayload is the payload for limited product announcement tasks
type LimitedProductAnnouncementPayload struct {
	ProductID        uuid.UUID                      `json:"product_id"`
	ProductName      string                         `json:"product_name"`
	AnnouncementType LimitedProductAnnouncementType `json:"announcement_type"`
	TargetDate       time.Time                      `json:"target_date"` // The actual premiere or availability date
	ScheduledAt      time.Time                      `json:"scheduled_at"`
}
