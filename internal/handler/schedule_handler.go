package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simplehub-go/internal/repository"
	"simplehub-go/internal/service"
)

type ScheduleHandler struct {
	schedRepo        *repository.ScheduleConfigRepository
	schedulerService *service.SchedulerService
}

func NewScheduleHandler(schedRepo *repository.ScheduleConfigRepository, schedulerService *service.SchedulerService) *ScheduleHandler {
	return &ScheduleHandler{schedRepo: schedRepo, schedulerService: schedulerService}
}

func (h *ScheduleHandler) Get(c *gin.Context) {
	cfg, err := h.schedRepo.GetOrCreate()
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	Data(c, gin.H{"config": cfg})
}

// TEST: 手动触发检测
func (h *ScheduleHandler) Trigger(c *gin.Context) {
	if h.schedulerService == nil {
		Fail(c, http.StatusInternalServerError, "调度服务不可用")
		return
	}
	go h.schedulerService.RunGlobalCheck()
	Data(c, gin.H{"message": "已触发全局检测（后台执行）"})
}

func (h *ScheduleHandler) Update(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		Fail(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	// Convert camelCase JSON keys to snake_case for GORM column names
	dbUpdates := make(map[string]interface{}, len(updates))
	for k, v := range updates {
		dbUpdates[camelToSnake(k)] = v
	}

	if v, ok := dbUpdates["hour"]; ok {
		if h, ok2 := v.(float64); !ok2 || h < 0 || h > 23 {
			Fail(c, http.StatusBadRequest, "hour 必须在 0-23 之间")
			return
		}
	}
	if v, ok := dbUpdates["minute"]; ok {
		if m, ok2 := v.(float64); !ok2 || m < 0 || m > 59 {
			Fail(c, http.StatusBadRequest, "minute 必须在 0-59 之间")
			return
		}
	}
	if v, ok := dbUpdates["interval"]; ok {
		if i, ok2 := v.(float64); !ok2 || i < 5 || i > 300 {
			Fail(c, http.StatusBadRequest, "interval 必须在 10-300 之间")
			return
		}
	}

	cfg, err := h.schedRepo.GetOrCreate()
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.schedRepo.Update(cfg.ID, dbUpdates); err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	updated, err := h.schedRepo.GetOrCreate()
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if h.schedulerService != nil {
		h.schedulerService.ScheduleAll()
	}
	Data(c, gin.H{"config": updated})
}
