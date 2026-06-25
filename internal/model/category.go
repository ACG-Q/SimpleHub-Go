package model

import "time"

type Category struct {
	ID           string    `json:"id" gorm:"primaryKey;size:25"`
	Name         string    `json:"name" gorm:"uniqueIndex;not null;size:255"`
	ScheduleCron *string   `json:"scheduleCron" gorm:"size:100"`
	Timezone     string    `json:"timezone" gorm:"default:Asia/Shanghai;size:50"`
	CreatedAt    time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updatedAt" gorm:"autoUpdateTime"`

	Sites []Site `json:"sites,omitempty" gorm:"foreignKey:CategoryID"`
}
