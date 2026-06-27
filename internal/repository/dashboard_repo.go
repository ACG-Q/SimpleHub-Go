package repository

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"simplehub-go/internal/model"
)

type DashboardRepository struct {
	db *gorm.DB
}

func NewDashboardRepository(db *gorm.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

type DashboardStats struct {
	TotalSites      int64          `json:"totalSites"`
	SitesByType     map[string]int `json:"sitesByType"`
	TotalCategories int64          `json:"totalCategories"`
	NeverChecked    int64          `json:"neverChecked"`
	RecentAdded     int            `json:"recentAdded"`
	RecentRemoved   int            `json:"recentRemoved"`
}

type RecentDiffItem struct {
	SiteID   string `json:"siteId"`
	SiteName string `json:"siteName"`
	DiffAt   string `json:"diffAt"`
	Added    int    `json:"added"`
	Removed  int    `json:"removed"`
}

func (r *DashboardRepository) GetStats() (*DashboardStats, error) {
	stats := &DashboardStats{
		SitesByType: make(map[string]int),
	}

	r.db.Model(&model.Site{}).Count(&stats.TotalSites)

	var typeCounts []struct {
		Type  string
		Count int64
	}
	r.db.Model(&model.Site{}).
		Select("api_type as type, count(*) as count").
		Group("api_type").
		Scan(&typeCounts)
	for _, tc := range typeCounts {
		stats.SitesByType[tc.Type] = int(tc.Count)
	}

	r.db.Model(&model.Category{}).Count(&stats.TotalCategories)

	r.db.Model(&model.Site{}).Where("last_checked_at IS NULL").Count(&stats.NeverChecked)

	var recentDiffs []struct {
		AddedJSON   string
		RemovedJSON string
	}
	r.db.Model(&model.ModelDiff{}).Limit(50).Order("diff_at DESC").Find(&recentDiffs)
	for _, d := range recentDiffs {
		if d.AddedJSON != "" {
			var arr []any
			if json.Unmarshal([]byte(d.AddedJSON), &arr) == nil {
				stats.RecentAdded += len(arr)
			}
		}
		if d.RemovedJSON != "" {
			var arr []any
			if json.Unmarshal([]byte(d.RemovedJSON), &arr) == nil {
				stats.RecentRemoved += len(arr)
			}
		}
	}

	return stats, nil
}

func (r *DashboardRepository) GetRecentDiffs(limit int) ([]RecentDiffItem, error) {
	if limit <= 0 {
		limit = 10
	}

	type result struct {
		SiteID   string `gorm:"column:site_id"`
		SiteName string `gorm:"column:name"`
		DiffAt   time.Time
		AddedJSON   string `gorm:"column:added_json"`
		RemovedJSON string `gorm:"column:removed_json"`
	}

	var rows []result
	err := r.db.Table("model_diffs").
		Select("model_diffs.site_id, sites.name, model_diffs.diff_at, model_diffs.added_json, model_diffs.removed_json").
		Joins("LEFT JOIN sites ON sites.id = model_diffs.site_id").
		Order("model_diffs.diff_at DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	items := make([]RecentDiffItem, len(rows))
	for i, row := range rows {
		added, removed := 0, 0
		if row.AddedJSON != "" {
			var arr []any
			if json.Unmarshal([]byte(row.AddedJSON), &arr) == nil {
				added = len(arr)
			}
		}
		if row.RemovedJSON != "" {
			var arr []any
			if json.Unmarshal([]byte(row.RemovedJSON), &arr) == nil {
				removed = len(arr)
			}
		}
		items[i] = RecentDiffItem{
			SiteID:   row.SiteID,
			SiteName: row.SiteName,
			DiffAt:   row.DiffAt.Format("2006-01-02 15:04:05"),
			Added:    added,
			Removed:  removed,
		}
	}
	return items, nil
}
