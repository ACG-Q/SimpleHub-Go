package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"simplehub-go/internal/service"
)

func AuthMiddleware(authService *service.AuthService, skipAuth bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if skipAuth {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "未授权访问，请重新登录",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "未授权访问，请重新登录",
			})
			return
		}

		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "未授权访问，请重新登录",
			})
			return
		}

		c.Set("userID", claims.Sub)
		c.Set("email", claims.Email)
		c.Set("isAdmin", claims.IsAdmin)
		c.Next()
	}
}
