package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/msojocs/ai-auto-register/server/internal/service"
)

type SettingHandler struct {
	svc *service.SettingService
}

func NewSettingHandler(svc *service.SettingService) *SettingHandler {
	return &SettingHandler{svc: svc}
}

// Get returns the current system settings.
func (h *SettingHandler) Get(c *gin.Context) {
	setting, err := h.svc.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(setting))
}

// Update saves new system settings.
func (h *SettingHandler) Update(c *gin.Context) {
	var req struct {
		SentinelBaseURL             string `json:"sentinel_base_url"`
		AccountActionProxyGroupID   *uint  `json:"account_action_proxy_group_id"`
		AccountCheckEnabled         bool   `json:"account_check_enabled"`
		AccountCheckIntervalMinutes int    `json:"account_check_interval_minutes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	setting, err := h.svc.Save(
		req.SentinelBaseURL,
		req.AccountActionProxyGroupID,
		req.AccountCheckEnabled,
		req.AccountCheckIntervalMinutes,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(setting))
}
