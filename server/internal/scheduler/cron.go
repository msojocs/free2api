package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/msojocs/ai-auto-register/server/internal/repository"
	"github.com/msojocs/ai-auto-register/server/internal/resource"
	"github.com/msojocs/ai-auto-register/server/internal/service"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron         *cron.Cron
	proxyRes     *resource.ProxyResource
	accountSvc   *service.AccountService
	settingRepo  repository.SettingRepository
	mu           sync.Mutex
	lastCheckRun time.Time
}

func NewScheduler(
	proxyRes *resource.ProxyResource,
	accountSvc *service.AccountService,
	settingRepo repository.SettingRepository,
) *Scheduler {
	return &Scheduler{
		cron:        cron.New(),
		proxyRes:    proxyRes,
		accountSvc:  accountSvc,
		settingRepo: settingRepo,
	}
}

func (s *Scheduler) Start() {
	s.cron.AddFunc("@every 5m", func() {
		log.Println("[scheduler] Reloading proxy resources")
		s.proxyRes.Reload()
	})

	// Tick every minute; actual account-check runs when settings allow.
	s.cron.AddFunc("@every 1m", func() {
		s.maybeRunAccountCheck()
	})

	s.cron.Start()
	log.Println("[scheduler] Started")
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) maybeRunAccountCheck() {
	if s.accountSvc == nil || s.settingRepo == nil {
		return
	}
	setting, err := s.settingRepo.Get()
	if err != nil || setting == nil || !setting.AccountCheckEnabled {
		return
	}
	intervalMins := setting.AccountCheckIntervalMinutes
	if intervalMins <= 0 {
		intervalMins = 60
	}

	s.mu.Lock()
	elapsed := time.Since(s.lastCheckRun)
	if elapsed < time.Duration(intervalMins)*time.Minute {
		s.mu.Unlock()
		return
	}
	s.lastCheckRun = time.Now()
	s.mu.Unlock()

	log.Printf("[scheduler] Starting account check (interval=%dm)", intervalMins)
	go s.accountSvc.CheckAndRefreshAll(context.Background())
}
