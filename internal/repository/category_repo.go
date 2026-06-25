package repository

import (
	"gorm.io/gorm"

	"simplehub-go/internal/model"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) List() ([]model.Category, error) {
	var categories []model.Category
	err := r.db.Preload("Sites", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Find(&categories).Error
	return categories, err
}

func (r *CategoryRepository) GetByID(id string) (*model.Category, error) {
	var cat model.Category
	err := r.db.Preload("Sites").First(&cat, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *CategoryRepository) GetByName(name string) (*model.Category, error) {
	var cat model.Category
	err := r.db.Where("name = ?", name).First(&cat).Error
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *CategoryRepository) Create(cat *model.Category) error {
	return r.db.Create(cat).Error
}

func (r *CategoryRepository) Update(id string, updates map[string]interface{}) error {
	return r.db.Model(&model.Category{}).Where("id = ?", id).Updates(updates).Error
}

func (r *CategoryRepository) Delete(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Site{}).Where("category_id = ?", id).
			Update("category_id", nil).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Category{}, "id = ?", id).Error
	})
}
