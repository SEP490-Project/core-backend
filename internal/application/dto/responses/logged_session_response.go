package responses

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
)

// LoggedSessionResponse represents logged session information in responses
type LoggedSessionResponse struct {
	ID                string `json:"id" gorm:"type:uuid;column:id;primaryKey"`
	UserID            string `json:"user_id" gorm:"type:uuid;column:user_id;not null"`
	RefreshTokenHash  string `json:"-" gorm:"type:text;column:refresh_token_hash"`
	DeviceFingerprint string `json:"device_fingerprint" gorm:"type:text;column:device_fingerprint"`
	ExpiryAt          string `json:"expiry_at"`
	IsRevoked         bool   `json:"is_revoked" gorm:"default:false"`
	LastUsedAt        string `json:"last_used_at" gorm:"autoUpdateTime"`
	CreatedAt         string `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         string `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

// ToResponse converts LoggedSession model to LoggedSessionResponse
func (lsr LoggedSessionResponse) ToResponse(model *model.LoggedSession) *LoggedSessionResponse {
	return &LoggedSessionResponse{
		ID:                model.ID.String(),
		UserID:            model.UserID.String(),
		RefreshTokenHash:  model.RefreshTokenHash,
		DeviceFingerprint: model.DeviceFingerprint,
		ExpiryAt:          utils.FormatLocalTime(model.ExpiryAt, ""),
		IsRevoked:         model.IsRevoked,
		LastUsedAt:        utils.FormatLocalTime(model.LastUsedAt, ""),
		CreatedAt:         utils.FormatLocalTime(model.CreatedAt, ""),
		UpdatedAt:         utils.FormatLocalTime(model.UpdatedAt, ""),
	}
}

// LoggedDeviceListResponse represents a list of unique device fingerprints
type LoggedDeviceListResponse []*string

// ToResponseList converts a slice of LoggedSession models to a list of unique device fingerprints
func (ldr LoggedDeviceListResponse) ToResponseList(sessions []model.LoggedSession) (responses LoggedDeviceListResponse) {
	if len(sessions) == 0 {
		return LoggedDeviceListResponse{}
	}

	mapper := func(session model.LoggedSession) *string { return &session.DeviceFingerprint }
	uniqueSessions := utils.UniqueSliceMapper(sessions, mapper)

	return LoggedDeviceListResponse(uniqueSessions)
}
