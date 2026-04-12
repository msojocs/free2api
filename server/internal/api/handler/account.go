package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/msojocs/ai-auto-register/server/internal/service"
)

type AccountHandler struct {
	svc *service.AccountService
}

func NewAccountHandler(svc *service.AccountService) *AccountHandler {
	return &AccountHandler{svc: svc}
}

func (h *AccountHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	accountType := c.Query("type")
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	accounts, total, err := h.svc.List(page, limit, accountType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"accounts": accounts, "total": total}))
}

func (h *AccountHandler) Delete(c *gin.Context) {
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

func (h *AccountHandler) Export(c *gin.Context) {
	accountType := c.Query("type")
	data, err := h.svc.Export(accountType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}
	c.Header("Content-Disposition", "attachment; filename=accounts.json")
	c.Data(http.StatusOK, "application/json", data)
}

func (h *AccountHandler) Import(c *gin.Context) {
	var records []service.ImportAccountRecord
	if err := c.ShouldBindJSON(&records); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid JSON: "+err.Error()))
		return
	}
	result, err := h.svc.Import(records)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(result))
}

func (h *AccountHandler) Check(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}

	result, err := h.svc.Check(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(result))
}

func (h *AccountHandler) RefreshChatGPTToken(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}

	result, err := h.svc.RefreshChatGPTToken(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(result))
}

func (h *AccountHandler) ChatGPTDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}

	result, err := h.svc.GetChatGPTDetail(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(result))
}
