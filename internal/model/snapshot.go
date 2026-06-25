package model

import "time"

type ModelSnapshot struct {
	ID             string     `json:"id" gorm:"primaryKey;size:25"`
	SiteID         string     `json:"siteId" gorm:"index;not null;size:25"`
	FetchedAt      time.Time  `json:"fetchedAt" gorm:"autoCreateTime"`
	ModelsJSON     string     `json:"modelsJson" gorm:"type:text;not null"`
	Hash           string     `json:"hash" gorm:"size:64"`
	RawResponse    *string    `json:"rawResponse" gorm:"type:text"`
	ErrorMessage   *string    `json:"errorMessage" gorm:"size:1000"`
	StatusCode     *int       `json:"statusCode" gorm:""`
	ResponseTime   *int       `json:"responseTime" gorm:""`
	BillingLimit   *float64   `json:"billingLimit" gorm:"type:decimal(12,4)"`
	BillingUsage   *float64   `json:"billingUsage" gorm:"type:decimal(12,4)"`
	BillingError   *string    `json:"billingError" gorm:"size:1000"`
	CheckInSuccess *bool      `json:"checkInSuccess" gorm:""`
	CheckInMessage *string    `json:"checkInMessage" gorm:"size:500"`
	CheckInQuota   *float64   `json:"checkInQuota" gorm:"type:decimal(12,4)"`
	CheckInError   *string    `json:"checkInError" gorm:"size:1000"`
}
