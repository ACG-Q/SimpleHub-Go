package model

import "time"

type EmailConfig struct {
	ID              string    `json:"id" gorm:"primaryKey;size:25"`
	ResendAPIKeyEnc string    `json:"-" gorm:"not null"`
	NotifyEmails    string    `json:"notifyEmails" gorm:"not null;type:text"`
	Enabled         bool      `json:"enabled" gorm:"default:true"`
	CreatedAt       time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}
