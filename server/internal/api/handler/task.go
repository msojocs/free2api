package handler

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msojocs/ai-auto-register/server/internal/service"
)

type TaskHandler struct {
	svc *service.TaskService
}

func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

func (h *TaskHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	tasks, total, err := h.svc.List(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"tasks": tasks, "total": total, "page": page, "limit": limit}))
}

func (h *TaskHandler) Create(c *gin.Context) {
	var req struct {
		Type   string                 `json:"type" binding:"required"`
		Total  int                    `json:"total" binding:"required,min=1"`
		Config map[string]interface{} `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	task, err := h.svc.Create(req.Type, req.Total, req.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusCreated, OK(gin.H{"task": task}))
}

func (h *TaskHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	task, err := h.svc.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Fail(404, "task not found"))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"task": task}))
}

func (h *TaskHandler) Delete(c *gin.Context) {
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

func (h *TaskHandler) Start(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}
	if err := h.svc.Start(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"message": "started"}))
}

func (h *TaskHandler) Pause(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}
	if err := h.svc.Pause(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"message": "paused"}))
}

func (h *TaskHandler) Retry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}
	if err := h.svc.Retry(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"message": "retrying"}))
}

func (h *TaskHandler) Logs(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}
	logs, err := h.svc.GetLogs(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Fail(404, "task not found"))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"logs": logs}))
}

func (h *TaskHandler) Progress(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return
	}
	taskID := uint(id)
	ch := h.svc.Subscribe(taskID)
	defer h.svc.Unsubscribe(taskID, ch)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case update, ok := <-ch:
			if !ok {
				return false
			}
			c.SSEvent("progress", update)
			return true
		case <-ticker.C:
			c.SSEvent("ping", "")
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
