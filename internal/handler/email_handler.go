package handler

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"simplehub-go/internal/crypto"
	"simplehub-go/internal/model"
	"simplehub-go/internal/repository"
	"simplehub-go/internal/service"
)

type EmailHandler struct {
	emailRepo     *repository.EmailConfigRepository
	encryptionKey string
	notifService  *service.NotificationService
}

func NewEmailHandler(emailRepo *repository.EmailConfigRepository, encryptionKey string, notifService *service.NotificationService) *EmailHandler {
	return &EmailHandler{
		emailRepo:     emailRepo,
		encryptionKey: encryptionKey,
		notifService:  notifService,
	}
}

// TEST: 发送测试邮件
func (h *EmailHandler) Test(c *gin.Context) {
	if h.notifService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "通知服务不可用"})
		return
	}
	if err := h.notifService.SendTestEmail(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "测试邮件已发送"})
}

func (h *EmailHandler) Get(c *gin.Context) {
	cfg, err := h.emailRepo.Get()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled":      false,
			"notifyEmails": "",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled":      cfg.Enabled,
		"notifyEmails": cfg.NotifyEmails,
	})
}

func (h *EmailHandler) Upsert(c *gin.Context) {
	var req struct {
		ResendAPIKey string `json:"resendApiKey" binding:"required"`
		NotifyEmails string `json:"notifyEmails" binding:"required"`
		Enabled      *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供必填字段"})
		return
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	parts := strings.Split(req.NotifyEmails, ",")
	for _, part := range parts {
		e := strings.TrimSpace(part)
		if e == "" {
			continue
		}
		if !emailRegex.MatchString(e) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式不正确: " + e})
			return
		}
	}

	encryptedKey, err := crypto.Encrypt(req.ResendAPIKey, h.encryptionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加密失败"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	cfg := &model.EmailConfig{
		ID:              newID(),
		ResendAPIKeyEnc: encryptedKey,
		NotifyEmails:    req.NotifyEmails,
		Enabled:         enabled,
	}

	if err := h.emailRepo.Upsert(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":      cfg.Enabled,
		"notifyEmails": cfg.NotifyEmails,
	})
}
