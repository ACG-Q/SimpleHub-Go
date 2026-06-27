package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simplehub-go/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请提供邮箱和密码")
		return
	}

	token, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		Fail(c, http.StatusUnauthorized, "管理员账号或密码错误")
		return
	}

	Data(c, gin.H{"token": token})
}
