package enum

type CampaignStatus string

// 	Status          enum.CampaignStatus `json:"status" gorm:"type:enum('RUNNING','COMPLETED','CANCELED');not null"`
const (
	CampaignRunning   CampaignStatus = "RUNNING"
	CampaignCompleted CampaignStatus = "COMPLETED"
	CampaignCanceled  CampaignStatus = "CANCELED"
)
