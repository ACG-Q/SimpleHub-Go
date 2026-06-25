package model

import "time"

type ModelDiff struct {
	ID             string     `json:"id" gorm:"primaryKey;size:25"`
	SiteID         string     `json:"siteId" gorm:"index;not null;size:25"`
	DiffAt         time.Time  `json:"diffAt" gorm:"autoCreateTime"`
	AddedJSON      string     `json:"addedJson" gorm:"type:text;not null"`
	RemovedJSON    string     `json:"removedJson" gorm:"type:text;not null"`
	ChangedJSON    string     `json:"changedJson" gorm:"type:text;not null"`
	SnapshotFromID *string    `json:"snapshotFromId" gorm:"size:25"`
	SnapshotToID   *string    `json:"snapshotToId" gorm:"size:25"`
}
