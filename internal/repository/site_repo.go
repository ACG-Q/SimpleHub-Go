package repository

import (
	"gorm.io/gorm"

	"simplehub-go/internal/model"
)

type SiteRepository struct {
	db *gorm.DB
}

func NewSiteRepository(db *gorm.DB) *SiteRepository {
	return &SiteRepository{db: db}
}

func (r *SiteRepository) List(search string) ([]model.Site, error) {
	var sites []model.Site
	q := r.db.Preload("Category").Order("pinned DESC, sort_order ASC, created_at DESC")
	if search != "" {
		q = q.Where("name LIKE ? OR base_url LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	err := q.Find(&sites).Error
	return sites, err
}

func (r *SiteRepository) GetByID(id string) (*model.Site, error) {
	var site model.Site
	err := r.db.Preload("Category").First(&site, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &site, nil
}

func (r *SiteRepository) Create(site *model.Site) error {
	return r.db.Create(site).Error
}

func (r *SiteRepository) Update(id string, updates map[string]interface{}) error {
	return r.db.Model(&model.Site{}).Where("id = ?", id).Updates(updates).Error
}

func (r *SiteRepository) Delete(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("site_id = ?", id).Delete(&model.ModelSnapshot{}).Error; err != nil {
			return err
		}
		if err := tx.Where("site_id = ?", id).Delete(&model.ModelDiff{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Site{}, "id = ?", id).Error
	})
}

func (r *SiteRepository) BatchUpdateOrder(ids []string) error {
	tx := r.db.Begin()
	for i, id := range ids {
		if err := tx.Model(&model.Site{}).Where("id = ?", id).Update("sort_order", i).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}
