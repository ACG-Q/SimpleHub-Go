package repository

import (
	"gorm.io/gorm"

	"simplehub-go/internal/model"
)

type EmailConfigRepository struct {
	db *gorm.DB
}

func NewEmailConfigRepository(db *gorm.DB) *EmailConfigRepository {
	return &EmailConfigRepository{db: db}
}

func (r *EmailConfigRepository) Get() (*model.EmailConfig, error) {
	var cfg model.EmailConfig
	err := r.db.First(&cfg).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *EmailConfigRepository) Upsert(cfg *model.EmailConfig) error {
	var existing model.EmailConfig
	if err := r.db.First(&existing).Error; err != nil {
		return r.db.Create(cfg).Error
	}
	return r.db.Model(&existing).Updates(map[string]interface{}{
		"resend_api_key_enc": cfg.ResendAPIKeyEnc,
		"notify_emails":      cfg.NotifyEmails,
		"from_email":         cfg.FromEmail,
		"enabled":            cfg.Enabled,
	}).Error
}
