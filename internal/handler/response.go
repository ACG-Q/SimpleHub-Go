package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Data(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"ok": true, "data": data})
}

func OK(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func Fail(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"ok": false, "error": msg})
}
