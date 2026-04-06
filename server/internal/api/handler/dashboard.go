package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/msojocs/free2api/server/internal/model"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	db *gorm.DB
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

func (h *DashboardHandler) Stats(c *gin.Context) {
	var totalAccounts, activeAccounts int64
	h.db.Model(&model.Account{}).Count(&totalAccounts)
	h.db.Model(&model.Account{}).Where("status = ?", "active").Count(&activeAccounts)

	var totalTasks int64
	h.db.Model(&model.TaskBatch{}).Count(&totalTasks)

	var proxiesAvailable int64
	h.db.Model(&model.Proxy{}).Where("status = ?", "active").Count(&proxiesAvailable)

	var tempMailProviders int64
	h.db.Model(&model.TempMailProvider{}).Where("enabled = ?", true).Count(&tempMailProviders)

	c.JSON(http.StatusOK, OK(gin.H{
		"total_accounts":      totalAccounts,
		"active_accounts":     activeAccounts,
		"total_tasks":         totalTasks,
		"proxies_available":   proxiesAvailable,
		"temp_mail_providers": tempMailProviders,
	}))
}
