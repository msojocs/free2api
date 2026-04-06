package core

import (
	"context"
	"sync"
)

type ProgressUpdate struct {
	TaskID   uint   `json:"task_id"`
	Progress int    `json:"progress"`
	Message  string `json:"message"`
	Status   string `json:"status"`
}

type Job struct {
	ID      uint
	Execute func(ctx context.Context, publish func(ProgressUpdate))
}

type WorkerPool struct {
	workers     int
	jobs        chan Job
	subscribers map[uint][]chan ProgressUpdate
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewWorkerPool(workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workers:     workers,
		jobs:        make(chan Job, 200),
		subscribers: make(map[uint][]chan ProgressUpdate),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.wg.Wait()
}

func (wp *WorkerPool) Submit(job Job) {
	select {
	case wp.jobs <- job:
	case <-wp.ctx.Done():
	}
}

func (wp *WorkerPool) Subscribe(taskID uint) chan ProgressUpdate {
	ch := make(chan ProgressUpdate, 200)
	wp.mu.Lock()
	wp.subscribers[taskID] = append(wp.subscribers[taskID], ch)
	wp.mu.Unlock()
	return ch
}

func (wp *WorkerPool) Unsubscribe(taskID uint, ch chan ProgressUpdate) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	subs := wp.subscribers[taskID]
	for i, s := range subs {
		if s == ch {
			wp.subscribers[taskID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			return
		}
	}
}

func (wp *WorkerPool) publish(update ProgressUpdate) {
	wp.mu.RLock()
	subs := make([]chan ProgressUpdate, len(wp.subscribers[update.TaskID]))
	copy(subs, wp.subscribers[update.TaskID])
	wp.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- update:
		default:
		}
	}
}

func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for {
		select {
		case job, ok := <-wp.jobs:
			if !ok {
				return
			}
			job.Execute(wp.ctx, func(u ProgressUpdate) {
				wp.publish(u)
			})
		case <-wp.ctx.Done():
			return
		}
	}
}
