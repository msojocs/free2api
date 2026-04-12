package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/msojocs/ai-auto-register/server/internal/resource"
)

type CaptchaHandler struct {
	res *resource.CaptchaResource
}

func NewCaptchaHandler(res *resource.CaptchaResource) *CaptchaHandler {
	return &CaptchaHandler{res: res}
}

func (h *CaptchaHandler) Stats(c *gin.Context) {
	stats := h.res.GetStats()
	c.JSON(http.StatusOK, OK(gin.H{"stats": stats}))
}
