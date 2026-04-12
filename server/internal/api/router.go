package api

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/msojocs/ai-auto-register/server/internal/api/handler"
	"github.com/msojocs/ai-auto-register/server/internal/service"
)

func authMiddleware(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			c.Abort()
			return
		}
		token := authHeader[7:]
		claims, err := authSvc.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func SetupRouter(
	authH *handler.AuthHandler,
	taskH *handler.TaskHandler,
	accountH *handler.AccountHandler,
	proxyH *handler.ProxyHandler,
	proxyGroupH *handler.ProxyGroupHandler,
	captchaH *handler.CaptchaHandler,
	dashboardH *handler.DashboardHandler,
	pushTemplateH *handler.PushTemplateHandler,
	tempMailProviderH *handler.TempMailProviderHandler,
	settingH *handler.SettingHandler,
	authSvc *service.AuthService,
) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: false,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
	}

	api := r.Group("/api", authMiddleware(authSvc))
	{
		authProtected := api.Group("/auth")
		{
			authProtected.POST("/change-password", authH.ChangePassword)
		}

		tasks := api.Group("/tasks")
		{
			tasks.GET("", taskH.List)
			tasks.POST("", taskH.Create)
			tasks.GET("/:id", taskH.Get)
			tasks.DELETE("/:id", taskH.Delete)
			tasks.POST("/:id/start", taskH.Start)
			tasks.POST("/:id/pause", taskH.Pause)
			tasks.POST("/:id/retry", taskH.Retry)
			tasks.GET("/:id/logs", taskH.Logs)
			tasks.GET("/:id/progress", taskH.Progress)
		}

		accounts := api.Group("/accounts")
		{
			accounts.GET("", accountH.List)
			accounts.DELETE("/:id", accountH.Delete)
			accounts.GET("/export", accountH.Export)
			accounts.POST("/import", accountH.Import)
			accounts.POST("/:id/check", accountH.Check)
			accounts.POST("/:id/chatgpt/refresh-token", accountH.RefreshChatGPTToken)
			accounts.GET("/:id/chatgpt/detail", accountH.ChatGPTDetail)
		}

		proxies := api.Group("/proxies")
		{
			proxies.GET("", proxyH.List)
			proxies.POST("", proxyH.Create)
			proxies.DELETE("/:id", proxyH.Delete)
			proxies.POST("/:id/test", proxyH.Test)
			proxies.PUT("/:id", proxyH.Update)
		}

		proxyGroups := api.Group("/proxy-groups")
		{
			proxyGroups.GET("", proxyGroupH.List)
			proxyGroups.POST("", proxyGroupH.Create)
			proxyGroups.PUT("/:id", proxyGroupH.Update)
			proxyGroups.DELETE("/:id", proxyGroupH.Delete)
		}

		captcha := api.Group("/captcha")
		{
			captcha.GET("/stats", captchaH.Stats)
		}

		dashboard := api.Group("/dashboard")
		{
			dashboard.GET("/stats", dashboardH.Stats)
		}

		push_templates := api.Group("/push-templates")
		{
			push_templates.GET("", pushTemplateH.List)
			push_templates.GET("/for-upload", pushTemplateH.ListForUpload)
			push_templates.POST("", pushTemplateH.Create)
			push_templates.PUT("/:id", pushTemplateH.Update)
			push_templates.DELETE("/:id", pushTemplateH.Delete)
			push_templates.POST("/:id/copy", pushTemplateH.Copy)
			push_templates.POST("/:id/test", pushTemplateH.TestPush)
			push_templates.POST("/:id/push-account", pushTemplateH.PushAccountByID)
		}

		tempMailProviders := api.Group("/temp-mail-providers")
		{
			tempMailProviders.GET("", tempMailProviderH.List)
			tempMailProviders.POST("", tempMailProviderH.Create)
			tempMailProviders.PUT("/:id", tempMailProviderH.Update)
			tempMailProviders.DELETE("/:id", tempMailProviderH.Delete)
			tempMailProviders.POST("/:id/test", tempMailProviderH.Test)
		}

		settings := api.Group("/settings")
		{
			settings.GET("", settingH.Get)
			settings.PUT("", settingH.Update)
		}
	}

	return r
}
