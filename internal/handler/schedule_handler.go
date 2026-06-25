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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "config": cfg})
}

// TEST: 手动触发检测
func (h *ScheduleHandler) Trigger(c *gin.Context) {
	if h.schedulerService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "调度服务不可用"})
		return
	}
	go h.schedulerService.RunGlobalCheck()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已触发全局检测（后台执行）"})
}

func (h *ScheduleHandler) Update(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// Convert camelCase JSON keys to snake_case for GORM column names
	dbUpdates := make(map[string]interface{}, len(updates))
	for k, v := range updates {
		dbUpdates[camelToSnake(k)] = v
	}

	if v, ok := dbUpdates["hour"]; ok {
		if h, ok2 := v.(float64); !ok2 || h < 0 || h > 23 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hour 必须在 0-23 之间"})
			return
		}
	}
	if v, ok := dbUpdates["minute"]; ok {
		if m, ok2 := v.(float64); !ok2 || m < 0 || m > 59 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "minute 必须在 0-59 之间"})
			return
		}
	}
	if v, ok := dbUpdates["interval"]; ok {
		if i, ok2 := v.(float64); !ok2 || i < 5 || i > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "interval 必须在 10-300 之间"})
			return
		}
	}

	cfg, err := h.schedRepo.GetOrCreate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.schedRepo.Update(cfg.ID, dbUpdates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.schedRepo.GetOrCreate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.schedulerService != nil {
		h.schedulerService.ScheduleAll()
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "config": updated})
}
