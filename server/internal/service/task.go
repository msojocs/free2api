package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/msojocs/free2api/server/internal/core"
	"github.com/msojocs/free2api/server/internal/executor"
	"github.com/msojocs/free2api/server/internal/model"
	"github.com/msojocs/free2api/server/internal/repository"
	"github.com/msojocs/free2api/server/internal/resource"
	"gorm.io/gorm"
)

var jobCounter uint64

type TaskService struct {
	repo     repository.TaskRepository
	pool     *core.WorkerPool
	db       *gorm.DB
	proxyRes *resource.ProxyResource
}

func NewTaskService(repo repository.TaskRepository, pool *core.WorkerPool, db *gorm.DB, proxyRes *resource.ProxyResource) *TaskService {
	return &TaskService{repo: repo, pool: pool, db: db, proxyRes: proxyRes}
}

func (s *TaskService) List(page, limit int) ([]model.TaskBatch, int64, error) {
	offset := (page - 1) * limit
	return s.repo.List(offset, limit)
}

func (s *TaskService) Create(name, taskType string, total int, config map[string]interface{}) (*model.TaskBatch, error) {
	validTypes := map[string]bool{
		"chatgpt": true,
		"cursor":  true,
		"trae":    true,
		"grok":    true,
		"tavily":  true,
		"kiro":    true,
	}
	if !validTypes[taskType] {
		return nil, fmt.Errorf("invalid task type: must be one of: chatgpt, cursor, trae, grok, tavily, kiro")
	}
	task := &model.TaskBatch{
		Name:   name,
		Type:   taskType,
		Status: model.TaskStatusPending,
		Total:  total,
		Config: model.JSONMap(config),
	}
	if err := s.repo.Create(task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) Get(id uint) (*model.TaskBatch, error) {
	task, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("task not found")
	}
	return task, nil
}

func (s *TaskService) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *TaskService) Start(id uint) error {
	task, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return errors.New("task not found")
	}
	if task.Status == model.TaskStatusRunning {
		return errors.New("task is already running")
	}
	if err := s.repo.UpdateFields(id, map[string]interface{}{"status": model.TaskStatusRunning, "logs": ""}); err != nil {
		return err
	}
	task.Status = model.TaskStatusRunning
	go s.dispatchJobs(*task)
	return nil
}

func (s *TaskService) dispatchJobs(task model.TaskBatch) {
	log.Printf("Dispatching jobs for task %d: total=%d\n", task.ID, task.Total)
	total := task.Total
	if total <= 0 {
		total = 1
	}

	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		current, err := s.repo.FindByID(task.ID)
		if err != nil || current == nil {
			break
		}
		if current.Status == model.TaskStatusPaused {
			log.Printf("Task %d is paused, stopping dispatch\n", task.ID)
			return
		}

		jobID := atomic.AddUint64(&jobCounter, 1)
		taskID := task.ID
		taskType := task.Type
		cfg := map[string]interface{}(task.Config)
		cfg = s.resolveProxyConfig(cfg)
		// If the task config references a temp mail provider by ID, resolve it
		// and merge the provider's settings into the job config so executors
		// receive mail_provider + mail_* keys transparently.
		cfg = s.resolveMailProviderConfig(cfg)
		wg.Add(1)
		s.pool.Submit(core.Job{
			ID: uint(jobID),
			Execute: func(ctx context.Context, publish func(core.ProgressUpdate)) {
				defer wg.Done()
				publishWithLog := func(update core.ProgressUpdate) {
					s.appendProgressLog(taskID, update)
					publish(update)
				}
				var exec executor.Executor
				log.Printf("Job type: %s\n", taskType)
				switch taskType {
				case "chatgpt":
					exec = executor.NewChatGPTExecutor()
				case "cursor":
					exec = executor.NewCursorExecutor()
				default:
					log.Printf("Unknown job type: %s\n", taskType)
					return
				}
				log.Printf("Starting job %d for task %d\n", jobID, taskID)
				result, err := exec.Execute(ctx, taskID, cfg, publishWithLog)
				if err == nil {
					if result == nil || result.Account == nil {
						err = errors.New("executor returned no account to persist")
					} else if dbErr := s.db.Create(result.Account).Error; dbErr != nil {
						publishWithLog(core.ProgressUpdate{
							TaskID:   taskID,
							Progress: 100,
							Message:  fmt.Sprintf("Save account failed: %v", dbErr),
							Status:   "failed",
						})
						err = dbErr
					} else if result.SuccessMessage != "" {
						publishWithLog(core.ProgressUpdate{
							TaskID:   taskID,
							Progress: 100,
							Message:  result.SuccessMessage,
							Status:   "completed",
						})
					}
				}

				if err != nil {
					s.repo.UpdateFields(taskID, map[string]interface{}{
						"failed": gorm.Expr("failed + ?", 1),
					})
				} else {
					s.repo.UpdateFields(taskID, map[string]interface{}{
						"completed": gorm.Expr("completed + ?", 1),
					})
				}
			},
		})
	}

	wg.Wait()
	finalTask, err := s.repo.FindByID(task.ID)
	if err == nil && finalTask != nil && finalTask.Status == model.TaskStatusRunning {
		s.repo.UpdateFields(task.ID, map[string]interface{}{"status": model.TaskStatusCompleted})
	}
}

func (s *TaskService) Pause(id uint) error {
	task, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return errors.New("task not found")
	}
	if task.Status != model.TaskStatusRunning {
		return errors.New("task is not running")
	}
	return s.repo.UpdateFields(id, map[string]interface{}{"status": model.TaskStatusPaused})
}

func (s *TaskService) Retry(id uint) error {
	task, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return errors.New("task not found")
	}
	if task.Status != model.TaskStatusFailed && task.Status != model.TaskStatusPaused {
		return fmt.Errorf("task status is %s, can only retry failed or paused tasks", task.Status)
	}
	if err := s.repo.UpdateFields(id, map[string]interface{}{
		"status": model.TaskStatusRunning,
		"failed": 0,
		"logs":   "",
	}); err != nil {
		return err
	}
	task.Status = model.TaskStatusRunning
	go s.dispatchJobs(*task)
	return nil
}

func (s *TaskService) Subscribe(taskID uint) chan core.ProgressUpdate {
	return s.pool.Subscribe(taskID)
}

func (s *TaskService) Unsubscribe(taskID uint, ch chan core.ProgressUpdate) {
	s.pool.Unsubscribe(taskID, ch)
}

func (s *TaskService) GetLogs(id uint) ([]core.ProgressUpdate, error) {
	task, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("task not found")
	}
	if task.Logs == "" {
		return []core.ProgressUpdate{}, nil
	}

	entries := make([]core.ProgressUpdate, 0)
	for _, line := range strings.Split(task.Logs, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry core.ProgressUpdate
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			entries = append(entries, entry)
			continue
		}
		entries = append(entries, core.ProgressUpdate{TaskID: id, Message: line})
	}
	return entries, nil
}

func (s *TaskService) appendProgressLog(taskID uint, update core.ProgressUpdate) {
	line, err := json.Marshal(update)
	if err != nil {
		return
	}
	value := string(line) + "\n"
	fields := map[string]interface{}{}
	switch s.db.Dialector.Name() {
	case "mysql":
		fields["logs"] = gorm.Expr("CONCAT(COALESCE(logs, ''), ?)", value)
	default:
		fields["logs"] = gorm.Expr("COALESCE(logs, '') || ?", value)
	}
	_ = s.repo.UpdateFields(taskID, fields)
}

func (s *TaskService) resolveProxyConfig(cfg map[string]interface{}) map[string]interface{} {
	if strings.TrimSpace(taskConfigString(cfg, "proxy")) != "" {
		return cfg
	}
	if s.proxyRes == nil {
		return cfg
	}
	var proxy *model.Proxy
	if groupID := taskConfigUint(cfg, "proxy_group_id"); groupID > 0 {
		proxy = s.proxyRes.NextByGroupID(groupID)
	} else {
		group := strings.TrimSpace(taskConfigString(cfg, "proxy_group"))
		if group != "" {
			proxy = s.proxyRes.NextByGroupName(group)
		}
	}
	if proxy == nil {
		return cfg
	}

	merged := make(map[string]interface{}, len(cfg)+1)
	for k, v := range cfg {
		merged[k] = v
	}
	merged["proxy"] = buildProxyURL(proxy)
	return merged
}

func taskConfigString(cfg map[string]interface{}, key string) string {
	if value, ok := cfg[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func taskConfigUint(cfg map[string]interface{}, key string) uint {
	if value, ok := cfg[key]; ok {
		switch typed := value.(type) {
		case float64:
			return uint(typed)
		case int:
			return uint(typed)
		case uint:
			return typed
		}
	}
	return 0
}

func buildProxyURL(proxy *model.Proxy) string {
	protocol := strings.TrimSpace(proxy.Protocol)
	if protocol == "" {
		protocol = "http"
	}
	u := &url.URL{
		Scheme: protocol,
		Host:   net.JoinHostPort(proxy.Host, proxy.Port),
	}
	if proxy.Username != "" || proxy.Password != "" {
		u.User = url.UserPassword(proxy.Username, proxy.Password)
	}
	return u.String()
}

// resolveMailProviderConfig looks up temp_mail_provider_id in cfg, loads the
// corresponding TempMailProvider record from the database, and merges its
// settings as mail_provider / mail_* keys so executors can consume them
// without being aware of the TempMailProvider model.
func (s *TaskService) resolveMailProviderConfig(cfg map[string]interface{}) map[string]interface{} {
	raw, ok := cfg["temp_mail_provider_id"]
	if !ok {
		return cfg
	}
	var id uint
	switch v := raw.(type) {
	case float64:
		id = uint(v)
	case int:
		id = uint(v)
	case uint:
		id = v
	default:
		return cfg
	}
	if id == 0 {
		return cfg
	}

	var p model.TempMailProvider
	if err := s.db.First(&p, id).Error; err != nil {
		// Provider not found — fall through and let the executor handle the missing config.
		return cfg
	}

	// Build a merged copy so the original task config is not mutated.
	merged := make(map[string]interface{}, len(cfg)+len(p.Config)+1)
	for k, v := range cfg {
		merged[k] = v
	}
	// Provider type and per-provider config keys.
	merged["mail_provider"] = p.ProviderType
	for k, v := range p.Config {
		if _, already := merged["mail_"+k]; !already {
			merged["mail_"+k] = v
		}
	}
	return merged
}
