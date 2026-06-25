package repository

import (
	"gorm.io/gorm"

	"simplehub-go/internal/model"
)

type DiffRepository struct {
	db *gorm.DB
}

func NewDiffRepository(db *gorm.DB) *DiffRepository {
	return &DiffRepository{db: db}
}

func (r *DiffRepository) List(siteID string, limit int) ([]model.ModelDiff, error) {
	if limit <= 0 {
		limit = 50
	}
	var diffs []model.ModelDiff
	err := r.db.Where("site_id = ?", siteID).
		Order("diff_at DESC").
		Limit(limit).
		Find(&diffs).Error
	return diffs, err
}

func (r *DiffRepository) Create(diff *model.ModelDiff) error {
	return r.db.Create(diff).Error
}
