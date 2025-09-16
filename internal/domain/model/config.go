// Package model defines the data structures for application configuration settings.
package model

// - Attributes: ConfigID (SERIAL PK), Key (VARCHAR UNIQUE, e.g., 'FB_API_KEY'), Value (TEXT/JSONB), Description (TEXT), UpdatedAt (TIMESTAMP).

type Config struct {
	ID          int    `json:"id" gorm:"primaryKey"`
	Key         string `json:"key" gorm:"not null;unique"`
	Value       string `json:"value" gorm:"type:text"`
	Description string `json:"description"`
	UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime"`
}
