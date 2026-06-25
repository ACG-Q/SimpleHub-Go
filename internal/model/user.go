package model

import "time"

type User struct {
	ID           string    `gorm:"primaryKey;size:25"`
	Email        string    `gorm:"uniqueIndex;not null;size:255"`
	PasswordHash string    `gorm:"not null"`
	IsAdmin      bool      `gorm:"default:false"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}
