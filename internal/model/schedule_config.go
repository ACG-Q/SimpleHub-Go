package model

import "time"

type ScheduleConfig struct {
	ID                 string     `json:"id" gorm:"primaryKey;size:25"`
	Enabled            bool       `json:"enabled" gorm:"default:false"`
	Hour               int        `json:"hour" gorm:"default:9"`
	Minute             int        `json:"minute" gorm:"default:0"`
	Timezone           string     `json:"timezone" gorm:"default:Asia/Shanghai;size:50"`
	Interval           int        `json:"interval" gorm:"default:30"`
	OverrideIndividual bool       `json:"overrideIndividual" gorm:"default:false"`
	LastRun            *time.Time `json:"lastRun" gorm:""`
	CreatedAt          time.Time  `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt          time.Time  `json:"updatedAt" gorm:"autoUpdateTime"`
}
