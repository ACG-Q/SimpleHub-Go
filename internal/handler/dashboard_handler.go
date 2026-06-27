package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simplehub-go/internal/repository"
)

type DashboardHandler struct {
	dashRepo *repository.DashboardRepository
}

func NewDashboardHandler(dashRepo *repository.DashboardRepository) *DashboardHandler {
	return &DashboardHandler{dashRepo: dashRepo}
}

func (h *DashboardHandler) Stats(c *gin.Context) {
	stats, err := h.dashRepo.GetStats()
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	diffs, err := h.dashRepo.GetRecentDiffs(10)
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	Data(c, gin.H{
		"stats":       stats,
		"recentDiffs": diffs,
	})
}
