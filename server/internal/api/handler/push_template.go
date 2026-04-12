package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msojocs/ai-auto-register/server/internal/service"
)

type PushTemplateHandler struct {
	svc *service.PushTemplateService
}

func NewPushTemplateHandler(svc *service.PushTemplateService) *PushTemplateHandler {
	return &PushTemplateHandler{svc: svc}
}

func parsePushTemplateID(c *gin.Context) (uint, bool) {
	raw, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || raw > math.MaxUint32 {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid id"))
		return 0, false
	}
	return uint(raw), true
}

func (h *PushTemplateHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	templates, total, err := h.svc.List(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"push_templates": templates, "total": total}))
}

func (h *PushTemplateHandler) Create(c *gin.Context) {
	var req service.CreatePushTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	t, err := h.svc.Create(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusCreated, OK(gin.H{"push_template": t}))
}

func (h *PushTemplateHandler) Update(c *gin.Context) {
	id, ok := parsePushTemplateID(c)
	if !ok {
		return
	}
	var req service.UpdatePushTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	t, err := h.svc.Update(id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"push_template": t}))
}

func (h *PushTemplateHandler) Delete(c *gin.Context) {
	id, ok := parsePushTemplateID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"message": "deleted"}))
}

func (h *PushTemplateHandler) Copy(c *gin.Context) {
	id, ok := parsePushTemplateID(c)
	if !ok {
		return
	}
	t, err := h.svc.CopyTemplate(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusCreated, OK(gin.H{"push_template": t}))
}

func (h *PushTemplateHandler) ListForUpload(c *gin.Context) {
	accountType := c.Query("type")
	templates, err := h.svc.ListEnabledForType(accountType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"push_templates": templates}))
}

func (h *PushTemplateHandler) PushAccountByID(c *gin.Context) {
	id, ok := parsePushTemplateID(c)
	if !ok {
		return
	}
	var req struct {
		AccountID uint `json:"account_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	statusCode, response, err := h.svc.PushAccountByID(req.AccountID, id)
	if err != nil {
		c.JSON(http.StatusOK, OK(gin.H{"ok": false, "status_code": 0, "response": err.Error()}))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{
		"ok":          statusCode < 400,
		"status_code": statusCode,
		"response":    response,
	}))
}

func (h *PushTemplateHandler) TestPush(c *gin.Context) {
	id, ok := parsePushTemplateID(c)
	if !ok {
		return
	}

	tmpl, err := h.svc.GetTemplate(id)
	if err != nil {
		c.JSON(http.StatusNotFound, Fail(404, err.Error()))
		return
	}

	fakeData := map[string]interface{}{
		"email":      "test@example.com",
		"password":   "TestPass123!",
		"type":       "cursor",
		"status":     "active",
		"extra":      "",
		"task_id":    0,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}

	rendered, err := renderBodyTemplate(tmpl.BodyTemplate, fakeData)
	if err != nil {
		c.JSON(http.StatusOK, OK(gin.H{"ok": false, "status_code": 0, "response": "template render error: " + err.Error()}))
		return
	}

	method := strings.ToUpper(tmpl.Method)
	var reqBody io.Reader
	if method != "GET" {
		reqBody = strings.NewReader(rendered)
	}

	httpReq, err := http.NewRequest(method, tmpl.URL, reqBody)
	if err != nil {
		c.JSON(http.StatusOK, OK(gin.H{"ok": false, "status_code": 0, "response": "request build error: " + err.Error()}))
		return
	}

	if tmpl.Headers != "" {
		var headers map[string]string
		if json.Unmarshal([]byte(tmpl.Headers), &headers) == nil {
			for k, v := range headers {
				httpReq.Header.Set(k, v)
			}
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusOK, OK(gin.H{"ok": false, "status_code": 0, "response": err.Error()}))
		return
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)

	success := resp.StatusCode < 400
	c.JSON(http.StatusOK, OK(gin.H{
		"ok":          success,
		"status_code": resp.StatusCode,
		"response":    string(respBytes),
	}))
}

func renderBodyTemplate(bodyTmpl string, data map[string]interface{}) (string, error) {
	t, err := template.New("body").Parse(bodyTmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
