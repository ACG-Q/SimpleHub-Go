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
		Fail(c, http.StatusInternalServerError, "通知服务不可用")
		return
	}
	var opts service.SendTestEmailOptions
	c.ShouldBindJSON(&opts)
	if err := h.notifService.SendTestEmail(&opts); err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	Data(c, gin.H{"message": "测试邮件已发送"})
}

func (h *EmailHandler) Get(c *gin.Context) {
	cfg, err := h.emailRepo.Get()
	if err != nil {
		Data(c, gin.H{
			"enabled":      false,
			"notifyEmails": "",
			"fromEmail":    "onboarding@resend.dev",
		})
		return
	}
	fromEmail := cfg.FromEmail
	if fromEmail == "" {
		fromEmail = "onboarding@resend.dev"
	}
	Data(c, gin.H{
		"enabled":      cfg.Enabled,
		"notifyEmails": cfg.NotifyEmails,
		"fromEmail":    fromEmail,
	})
}

func (h *EmailHandler) Upsert(c *gin.Context) {
	var req struct {
		ResendAPIKey string `json:"resendApiKey"`
		NotifyEmails string `json:"notifyEmails" binding:"required"`
		FromEmail    string `json:"fromEmail"`
		Enabled      *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请提供必填字段")
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
			Fail(c, http.StatusBadRequest, "邮箱格式不正确: "+e)
			return
		}
	}

	var encryptedKey string
	if req.ResendAPIKey != "" {
		var err error
		encryptedKey, err = crypto.Encrypt(req.ResendAPIKey, h.encryptionKey)
		if err != nil {
			Fail(c, http.StatusInternalServerError, "加密失败")
			return
		}
	} else {
		existing, err := h.emailRepo.Get()
		if err == nil {
			encryptedKey = existing.ResendAPIKeyEnc
		}
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	cfg := &model.EmailConfig{
		ID:              newID(),
		ResendAPIKeyEnc: encryptedKey,
		NotifyEmails:    req.NotifyEmails,
		FromEmail:       req.FromEmail,
		Enabled:         enabled,
	}
	if cfg.FromEmail == "" {
		cfg.FromEmail = "onboarding@resend.dev"
	}

	if err := h.emailRepo.Upsert(cfg); err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	Data(c, gin.H{
		"enabled":      cfg.Enabled,
		"notifyEmails": cfg.NotifyEmails,
		"fromEmail":    cfg.FromEmail,
	})
}
