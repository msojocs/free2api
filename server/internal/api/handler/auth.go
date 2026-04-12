package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/msojocs/ai-auto-register/server/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=32"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	user, err := h.svc.Register(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusConflict, Fail(409, err.Error()))
		return
	}
	c.JSON(http.StatusCreated, OK(gin.H{"user": user}))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	token, user, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, Fail(401, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"token": token, "user": user}))
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Fail(401, "unauthorized"))
		return
	}
	userID, ok := userIDRaw.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, Fail(401, "invalid user id"))
		return
	}
	if err := h.svc.ChangePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusOK, OK(gin.H{"changed": true}))
}
