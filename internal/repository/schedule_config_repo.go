package repository

import (
	"time"

	"gorm.io/gorm"

	"simplehub-go/internal/model"
)

type ScheduleConfigRepository struct {
	db *gorm.DB
}

func NewScheduleConfigRepository(db *gorm.DB) *ScheduleConfigRepository {
	return &ScheduleConfigRepository{db: db}
}

func (r *ScheduleConfigRepository) GetOrCreate() (*model.ScheduleConfig, error) {
	var cfg model.ScheduleConfig
	err := r.db.First(&cfg).Error
	if err != nil {
		cfg = model.ScheduleConfig{
			ID:       newID(),
			Enabled:  false,
			Hour:     9,
			Minute:   0,
			Interval: 30,
			Timezone: "Asia/Shanghai",
		}
		if e := r.db.Create(&cfg).Error; e != nil {
			return nil, e
		}
		return &cfg, nil
	}
	if cfg.ID == "" {
		cfg.ID = newID()
		if e := r.db.Save(&cfg).Error; e != nil {
			return nil, e
		}
	}
	return &cfg, nil
}

func newID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 25)
	t := time.Now().UnixNano()
	for i := 0; i < 25; i++ {
		b[i] = chars[(t>>(i*2))%36]
	}
	return string(b)
}

func (r *ScheduleConfigRepository) Update(id string, updates map[string]interface{}) error {
	return r.db.Model(&model.ScheduleConfig{}).Where("id = ?", id).Updates(updates).Error
}
