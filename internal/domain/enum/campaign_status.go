package enum

import (
	"database/sql/driver"
	"fmt"
)

type CampaignStatus string

// Status          enum.CampaignStatus `json:"status" gorm:"type:enum('RUNNING','COMPLETED','CANCELED');not null"`
const (
	CampaignRunning   CampaignStatus = "ON_GOING"
	CampaignCompleted CampaignStatus = "COMPLETED"
	CampaignCanceled  CampaignStatus = "CANCELED"
)

func (cs CampaignStatus) IsValid() bool {
	switch cs {
	case CampaignRunning, CampaignCompleted, CampaignCanceled:
		return true
	}
	return false
}

func (cs *CampaignStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan CampaignStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*cs = CampaignStatus(s)
	return nil
}

func (cs CampaignStatus) Value() (driver.Value, error) {
	return string(cs), nil
}

func (cs CampaignStatus) String() string {
	return string(cs)
}
