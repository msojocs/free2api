package scheduler

import (
	"log"

	"github.com/msojocs/free2api/server/internal/resource"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron     *cron.Cron
	proxyRes *resource.ProxyResource
}

func NewScheduler(proxyRes *resource.ProxyResource) *Scheduler {
	return &Scheduler{
		cron:     cron.New(),
		proxyRes: proxyRes,
	}
}

func (s *Scheduler) Start() {
	s.cron.AddFunc("@every 5m", func() {
		log.Println("[scheduler] Reloading proxy resources")
		s.proxyRes.Reload()
	})

	s.cron.Start()
	log.Println("[scheduler] Started")
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}
