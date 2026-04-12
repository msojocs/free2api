package handler

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/msojocs/ai-auto-register/server/internal/service"
)

// parseID parses a URL parameter as a uint, rejecting values that would
// overflow a uint on the current platform.
func parseTempMailID(s string) (uint, bool) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil || v > math.MaxUint32 {
		return 0, false
	}
	return uint(v), true
}

// TempMailProviderHandler handles HTTP requests for temporary email provider management.
type TempMailProviderHandler struct {
	svc *service.TempMailProviderService
}

func NewTempMailProviderHandler(svc *service.TempMailProviderService) *TempMailProviderHandler {
	return &TempMailProviderHandler{svc: svc}
}

// List returns all configured temp mail providers.
func (h *TempMailProviderHandler) List(c *gin.Context) {
	providers, err := h.svc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"providers": providers, "total": len(providers)}))
}

// Create adds a new temp mail provider configuration.
func (h *TempMailProviderHandler) Create(c *gin.Context) {
	var req struct {
		Name         string                 `json:"name" binding:"required"`
		ProviderType string                 `json:"provider_type" binding:"required"`
		Config       map[string]interface{} `json:"config"`
		Enabled      bool                   `json:"enabled"`
		Description  string                 `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	if req.Config == nil {
		req.Config = map[string]interface{}{}
	}
	p, err := h.svc.Create(req.Name, req.ProviderType, req.Description, req.Config, req.Enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusCreated, OK(gin.H{"provider": p}))
}

// Update modifies an existing temp mail provider configuration.
func (h *TempMailProviderHandler) Update(c *gin.Context) {
	id, ok := parseTempMailID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}
	var req struct {
		Name         string                 `json:"name"`
		ProviderType string                 `json:"provider_type"`
		Config       map[string]interface{} `json:"config"`
		Enabled      bool                   `json:"enabled"`
		Description  string                 `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	p, err := h.svc.Update(id, req.Name, req.ProviderType, req.Description, req.Config, req.Enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"provider": p}))
}

// Delete removes a temp mail provider configuration.
func (h *TempMailProviderHandler) Delete(c *gin.Context) {
	id, ok := parseTempMailID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}
	if err := h.svc.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"message": "deleted"}))
}

// Test validates the provider configuration by attempting to obtain a temporary email address.
func (h *TempMailProviderHandler) Test(c *gin.Context) {
	id, ok := parseTempMailID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}
	email, err := h.svc.TestProvider(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, OK(gin.H{"ok": false, "error": err.Error()}))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"ok": true, "email": email}))
}
