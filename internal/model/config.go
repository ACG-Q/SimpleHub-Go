package model

import "gorm.io/gorm"

type AppConfig struct {
	Key   string `gorm:"primaryKey;size:50"`
	Value string `gorm:"not null;size:500"`
}

func (AppConfig) TableName() string { return "app_config" }

func GetConfig(db *gorm.DB, key string) string {
	var c AppConfig
	if err := db.Where("key = ?", key).First(&c).Error; err != nil {
		return ""
	}
	return c.Value
}

func SetConfig(db *gorm.DB, key, value string) error {
	return db.Where("key = ?", key).Assign(AppConfig{Value: value}).FirstOrCreate(&AppConfig{Key: key}).Error
}
