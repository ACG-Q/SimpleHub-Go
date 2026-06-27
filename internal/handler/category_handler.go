package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"simplehub-go/internal/model"
	"simplehub-go/internal/repository"
	"simplehub-go/internal/service"
)

type CategoryHandler struct {
	catRepo          *repository.CategoryRepository
	checkService     *service.CheckService
	schedulerService *service.SchedulerService
}

func NewCategoryHandler(catRepo *repository.CategoryRepository, checkService *service.CheckService, schedulerService *service.SchedulerService) *CategoryHandler {
	return &CategoryHandler{
		catRepo:          catRepo,
		checkService:     checkService,
		schedulerService: schedulerService,
	}
}

func (h *CategoryHandler) List(c *gin.Context) {
	categories, err := h.catRepo.List()
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	Data(c, categories)
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var req struct {
		Name         string  `json:"name" binding:"required"`
		ScheduleCron *string `json:"scheduleCron"`
		Timezone     string  `json:"timezone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请提供分类名称")
		return
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}

	cat := &model.Category{
		ID:           newID(),
		Name:         req.Name,
		ScheduleCron: req.ScheduleCron,
		Timezone:     timezone,
	}
	if err := h.catRepo.Create(cat); err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if h.schedulerService != nil {
		h.schedulerService.ScheduleAll()
	}
	Created(c, cat)
}

func (h *CategoryHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		Fail(c, http.StatusBadRequest, "无效的请求数据")
		return
	}
	if err := h.catRepo.Update(id, updates); err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	cat, err := h.catRepo.GetByID(id)
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if h.schedulerService != nil {
		h.schedulerService.ScheduleAll()
	}
	Data(c, cat)
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.catRepo.Delete(id); err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if h.schedulerService != nil {
		h.schedulerService.ScheduleAll()
	}
	OK(c)
}

func (h *CategoryHandler) Check(c *gin.Context) {
	id := c.Param("id")
	cat, err := h.catRepo.GetByID(id)
	if err != nil {
		Fail(c, http.StatusNotFound, "分类不存在")
		return
	}

	skipNotif := c.Query("skipNotification") == "true"

	type changeInfo struct {
		SiteName string `json:"siteName"`
		Diff     string `json:"diff"`
	}
	type failInfo struct {
		SiteName string `json:"siteName"`
		Error    string `json:"error"`
	}

	var changes []changeInfo
	var failures []failInfo

	for _, site := range cat.Sites {
		if site.Pinned || site.ExcludeFromBatch {
			continue
		}
		sr, err := h.checkService.CheckSite(site.ID, skipNotif)
		if err != nil {
			failures = append(failures, failInfo{SiteName: site.Name, Error: err.Error()})
		} else if sr.Result.ErrorMessage != "" {
			failures = append(failures, failInfo{SiteName: site.Name, Error: sr.Result.ErrorMessage})
		} else if sr.Result.Hash != "" {
			changes = append(changes, changeInfo{SiteName: site.Name, Diff: sr.Result.Hash})
		}
		time.Sleep(5 * time.Second)
	}

	Data(c, gin.H{
		"changes":    changes,
		"failures":   failures,
		"totalSites": len(cat.Sites),
	})
}
