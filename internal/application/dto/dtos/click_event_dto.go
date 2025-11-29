package dtos

import (
	"time"

	"github.com/google/uuid"
)

// HourlyClickStats represents aggregated hourly click statistics
type HourlyClickStats struct {
	Hour           time.Time `json:"hour"`
	TotalClicks    int64     `json:"total_clicks"`
	UniqueUsers    int64     `json:"unique_users"`
	UniqueSessions int64     `json:"unique_sessions"`
}

// DailyClickStats represents aggregated daily click statistics
type DailyClickStats struct {
	Date           time.Time `json:"date"`
	TotalClicks    int64     `json:"total_clicks"`
	UniqueUsers    int64     `json:"unique_users"`
	UniqueSessions int64     `json:"unique_sessions"`
}

// AffiliateLinkPerformance represents performance metrics for an affiliate link
type AffiliateLinkPerformance struct {
	AffiliateLinkID uuid.UUID `json:"affiliate_link_id"`
	Hash            string    `json:"hash"`
	TotalClicks     int64     `json:"total_clicks"`
	UniqueUsers     int64     `json:"unique_users"`
	ContentTitle    string    `json:"content_title"`
	ChannelName     string    `json:"channel_name"`
}

// GlobalClickStats represents platform-wide click statistics
type GlobalClickStats struct {
	TotalClicks    int64 `json:"total_clicks"`
	UniqueUsers    int64 `json:"unique_users"`
	UniqueSessions int64 `json:"unique_sessions"`
}

// ContractPerformance represents click performance metrics for a contract
type ContractPerformance struct {
	ContractID   uuid.UUID `json:"contract_id"`
	ContractName string    `json:"contract_name"`
	TotalClicks  int64     `json:"total_clicks"`
	UniqueUsers  int64     `json:"unique_users"`
}

// ChannelPerformance represents click performance metrics for a channel
type ChannelPerformance struct {
	ChannelID   uuid.UUID `json:"channel_id"`
	ChannelName string    `json:"channel_name"`
	TotalClicks int64     `json:"total_clicks"`
	UniqueUsers int64     `json:"unique_users"`
}
