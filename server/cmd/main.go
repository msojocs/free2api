package main

import (
	"log"

	"github.com/msojocs/free2api/server/config"
	"github.com/msojocs/free2api/server/internal/api"
	"github.com/msojocs/free2api/server/internal/api/handler"
	"github.com/msojocs/free2api/server/internal/core"
	"github.com/msojocs/free2api/server/internal/model"
	"github.com/msojocs/free2api/server/internal/repository"
	"github.com/msojocs/free2api/server/internal/resource"
	"github.com/msojocs/free2api/server/internal/scheduler"
	"github.com/msojocs/free2api/server/internal/service"
)

func main() {
	// Load configuration (config.yaml in the working directory, overridden by env vars).
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if cfg.Auth.JWTSecret == "free2api_jwt_secret_change_in_production" {
		log.Println("WARNING: using default JWT_SECRET; set jwt_secret in config.yaml or JWT_SECRET env var in production.")
	}

	log.Printf("Database driver: %s", cfg.Database.Driver)
	if err := model.InitDB(cfg.Database.Driver, cfg.Database.DSN()); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database initialized")

	pool := core.NewWorkerPool(10)
	pool.Start()
	defer pool.Stop()

	proxyRes := resource.NewProxyResource(model.DB)
	captchaRes := resource.NewCaptchaResource(model.DB, cfg.Captcha.Provider, cfg.Captcha.APIKey)

	sched := scheduler.NewScheduler(proxyRes)
	sched.Start()
	defer sched.Stop()

	// Repository layer
	userRepo := repository.NewUserRepository(model.DB)
	taskRepo := repository.NewTaskRepository(model.DB)
	accountRepo := repository.NewAccountRepository(model.DB)
	proxyRepo := repository.NewProxyRepository(model.DB)
	proxyGroupRepo := repository.NewProxyGroupRepository(model.DB)
	pushTemplateRepo := repository.NewPushTemplateRepository(model.DB)
	settingRepo := repository.NewSettingRepository(model.DB)

	// Service layer (uses repositories)
	authSvc := service.NewAuthService(userRepo, cfg.Auth.JWTSecret)
	defaultAdmin, err := authSvc.EnsureDefaultAdmin(cfg.Auth.DefaultAdminUsername, cfg.Auth.DefaultAdminPassword)
	if err != nil {
		log.Fatalf("Failed to ensure default admin: %v", err)
	}
	if defaultAdmin != nil {
		log.Printf("Created default admin account: %s", defaultAdmin.Username)
		log.Println("WARNING: change the default admin password immediately after first login.")
	}
	settingSvc := service.NewSettingService(settingRepo, cfg.Executor.SentinelBaseURL)
	taskSvc := service.NewTaskService(taskRepo, pool, model.DB, proxyRes, settingSvc)
	accountSvc := service.NewAccountService(accountRepo)
	proxySvc := service.NewProxyService(proxyRepo, proxyGroupRepo, proxyRes)
	proxyGroupSvc := service.NewProxyGroupService(proxyGroupRepo, proxyRepo)
	pushTemplateSvc := service.NewPushTemplateService(pushTemplateRepo, accountRepo)
	tempMailProviderRepo := repository.NewTempMailProviderRepository(model.DB)
	tempMailProviderSvc := service.NewTempMailProviderService(tempMailProviderRepo)
	model.SeedPushTemplate(model.DB)
	model.SeedTempMailProviders(model.DB)
	pushTemplateSvc.RegisterDBHook(model.DB)

	authH := handler.NewAuthHandler(authSvc)
	taskH := handler.NewTaskHandler(taskSvc)
	accountH := handler.NewAccountHandler(accountSvc)
	proxyH := handler.NewProxyHandler(proxySvc)
	proxyGroupH := handler.NewProxyGroupHandler(proxyGroupSvc)
	captchaH := handler.NewCaptchaHandler(captchaRes)
	dashboardH := handler.NewDashboardHandler(model.DB)
	pushTemplateH := handler.NewPushTemplateHandler(pushTemplateSvc)
	tempMailProviderH := handler.NewTempMailProviderHandler(tempMailProviderSvc)
	settingH := handler.NewSettingHandler(settingSvc)

	r := api.SetupRouter(authH, taskH, accountH, proxyH, proxyGroupH, captchaH, dashboardH, pushTemplateH, tempMailProviderH, settingH, authSvc)

	log.Printf("Server starting on :%s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
