package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/msojocs/ai-auto-register/server/internal/service"
)

type ProxyHandler struct {
	svc *service.ProxyService
}

func NewProxyHandler(svc *service.ProxyService) *ProxyHandler {
	return &ProxyHandler{svc: svc}
}

func (h *ProxyHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	proxies, total, err := h.svc.List(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"proxies": proxies, "total": total}))
}

func (h *ProxyHandler) Create(c *gin.Context) {
	var req struct {
		Host         string `json:"host" binding:"required"`
		Port         string `json:"port" binding:"required"`
		ProxyGroupID *uint  `json:"proxy_group_id"`
		Username     string `json:"username"`
		Password     string `json:"password"`
		Protocol     string `json:"protocol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	proxy, err := h.svc.Create(req.Host, req.Port, req.ProxyGroupID, req.Username, req.Password, req.Protocol)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusCreated, OK(gin.H{"proxy": proxy}))
}

func (h *ProxyHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}
	var req struct {
		Host         string `json:"host" binding:"required"`
		Port         string `json:"port" binding:"required"`
		ProxyGroupID *uint  `json:"proxy_group_id"`
		Username     string `json:"username"`
		Password     string `json:"password"`
		Protocol     string `json:"protocol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	proxy, err := h.svc.Update(uint(id), req.Host, req.Port, req.ProxyGroupID, req.Username, req.Password, req.Protocol)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"proxy": proxy}))
}

func (h *ProxyHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}
	if err := h.svc.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"message": "deleted"}))
}

func (h *ProxyHandler) Test(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}
	ok, err := h.svc.Test(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Fail(404, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"ok": ok}))
}
