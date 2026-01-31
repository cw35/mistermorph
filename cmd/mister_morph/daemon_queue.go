package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

type queuedTask struct {
	info   *TaskInfo
	ctx    context.Context
	cancel context.CancelFunc
}

type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*queuedTask
	queue chan *queuedTask
}

func NewTaskStore(maxQueue int) *TaskStore {
	if maxQueue <= 0 {
		maxQueue = 100
	}
	return &TaskStore{
		tasks: make(map[string]*queuedTask),
		queue: make(chan *queuedTask, maxQueue),
	}
}

func (s *TaskStore) Enqueue(parent context.Context, task string, model string, timeout time.Duration) (*TaskInfo, error) {
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	if model == "" {
		model = "gpt-4o-mini"
	}

	id := fmt.Sprintf("%x", rand.Uint64())
	now := time.Now()
	ctx, cancel := context.WithTimeout(parent, timeout)

	info := &TaskInfo{
		ID:        id,
		Status:    TaskQueued,
		Task:      task,
		Model:     model,
		Timeout:   timeout.String(),
		CreatedAt: now,
	}
	qt := &queuedTask{info: info, ctx: ctx, cancel: cancel}

	s.mu.Lock()
	s.tasks[id] = qt
	s.mu.Unlock()

	select {
	case s.queue <- qt:
		return info, nil
	default:
		qt.cancel()
		s.mu.Lock()
		delete(s.tasks, id)
		s.mu.Unlock()
		return nil, fmt.Errorf("queue is full")
	}
}

func (s *TaskStore) Get(id string) (*TaskInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	qt, ok := s.tasks[id]
	if !ok || qt == nil || qt.info == nil {
		return nil, false
	}
	// Return a shallow copy for safe reads.
	cp := *qt.info
	return &cp, true
}

func (s *TaskStore) Next() *queuedTask {
	return <-s.queue
}

func (s *TaskStore) Update(id string, fn func(info *TaskInfo)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	qt := s.tasks[id]
	if qt == nil || qt.info == nil {
		return
	}
	fn(qt.info)
}
