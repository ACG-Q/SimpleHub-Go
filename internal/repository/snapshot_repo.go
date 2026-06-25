package repository

import (
	"strings"

	"gorm.io/gorm"

	"simplehub-go/internal/model"
)

type SnapshotRepository struct {
	db *gorm.DB
}

func NewSnapshotRepository(db *gorm.DB) *SnapshotRepository {
	return &SnapshotRepository{db: db}
}

func (r *SnapshotRepository) GetLatest(siteID string) (*model.ModelSnapshot, error) {
	var snap model.ModelSnapshot
	err := r.db.Where("site_id = ? AND error_message IS NULL", siteID).
		Order("fetched_at DESC").
		First(&snap).Error
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

func (r *SnapshotRepository) GetLatestWithError(siteID string) (*model.ModelSnapshot, error) {
	var snap model.ModelSnapshot
	err := r.db.Where("site_id = ?", siteID).
		Order("fetched_at DESC").
		First(&snap).Error
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

func (r *SnapshotRepository) List(siteID string, limit int) ([]model.ModelSnapshot, error) {
	if limit <= 0 {
		limit = 1
	}
	var snaps []model.ModelSnapshot
	err := r.db.Where("site_id = ? AND error_message IS NULL", siteID).
		Order("fetched_at DESC").
		Limit(limit).
		Find(&snaps).Error
	return snaps, err
}

func (r *SnapshotRepository) GetLatestForSites(siteIDs []string) (map[string]model.ModelSnapshot, error) {
	if len(siteIDs) == 0 {
		return map[string]model.ModelSnapshot{}, nil
	}

	var snaps []model.ModelSnapshot
	err := r.db.Where("site_id IN ? AND error_message IS NULL", siteIDs).
		Order("fetched_at DESC").
		Find(&snaps).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]model.ModelSnapshot, len(siteIDs))
	for _, snap := range snaps {
		if _, exists := result[snap.SiteID]; !exists {
			result[snap.SiteID] = snap
		}
	}
	return result, nil
}

func (r *SnapshotRepository) FindSiteIDsByModelID(search string) ([]string, error) {
	var siteIDs []string
	search = strings.ReplaceAll(search, `"`, `""`)
	pattern := `%"id":"` + search + `"%`
	err := r.db.Model(&model.ModelSnapshot{}).
		Where("error_message IS NULL").
		Where("models_json LIKE ?", pattern).
		Group("site_id").
		Pluck("site_id", &siteIDs).Error
	return siteIDs, err
}

func (r *SnapshotRepository) Create(snap *model.ModelSnapshot) error {
	return r.db.Create(snap).Error
}
