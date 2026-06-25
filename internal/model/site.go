package model

import "time"

type Site struct {
	ID               string     `json:"id" gorm:"primaryKey;size:25"`
	Name             string     `json:"name" gorm:"not null;size:255"`
	BaseURL          string     `json:"baseUrl" gorm:"not null;size:512"`
	APIKeyEnc        string     `json:"-" gorm:"not null"`
	APIType          string     `json:"apiType" gorm:"default:other;size:50"`
	UserID           *string    `json:"userId" gorm:"size:25"`
	BillingURL       *string    `json:"billingUrl" gorm:"size:512"`
	BillingAuthType  string     `json:"billingAuthType" gorm:"default:token;size:20"`
	BillingAuthValue *string    `json:"-" gorm:""`
	ProxyURLEnc      *string    `json:"-" gorm:""`
	BillingLimitField  *string  `json:"billingLimitField" gorm:"size:255"`
	BillingUsageField  *string  `json:"billingUsageField" gorm:"size:255"`
	UnlimitedQuota   bool       `json:"unlimitedQuota" gorm:"default:false"`
	EnableCheckIn    bool       `json:"enableCheckIn" gorm:"default:false"`
	CheckInMode      string     `json:"checkInMode" gorm:"default:both;size:20"`
	ScheduleCron     *string    `json:"scheduleCron" gorm:"size:100"`
	Timezone         string     `json:"timezone" gorm:"default:UTC;size:50"`
	Pinned           bool       `json:"pinned" gorm:"default:false"`
	ExcludeFromBatch bool       `json:"excludeFromBatch" gorm:"default:false"`
	CategoryID       *string    `json:"categoryId" gorm:"size:25"`
	Extralink        *string    `json:"extralink" gorm:"size:512"`
	Remark           *string    `json:"remark" gorm:"size:1000"`
	SortOrder        int        `json:"sortOrder" gorm:"default:0"`
	LastCheckedAt    *time.Time `json:"lastCheckedAt" gorm:""`
	CreatedAt        time.Time  `json:"createdAt" gorm:"autoCreateTime"`

	Category   *Category       `json:"category" gorm:"foreignKey:CategoryID"`
	Snapshots  []ModelSnapshot `json:"-" gorm:"foreignKey:SiteID"`
	Diffs      []ModelDiff     `json:"-" gorm:"foreignKey:SiteID"`
}

type SiteResponse struct {
	Site
	Token      string  `json:"token"`
	ProxyURL   *string `json:"proxyUrl"`
	BillingAV  *string `json:"billingAuthValue"`
	Type       string  `json:"type"`
}
