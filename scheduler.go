package main

import (
	"sync"

	"github.com/google/uuid"
)

type Scheduler struct {
	lock    sync.RWMutex
	runners map[string]Runner
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		lock:    sync.RWMutex{},
		runners: make(map[string]Runner, 0),
	}
}

func (s *Scheduler) NewRunner(runnerGroupConfig *RunnerGroupConfig) *Runner {
	s.lock.Lock()
	defer s.lock.Unlock()

	id := s.generateRunnerID()

	runner := NewRunner(id, runnerGroupConfig)
	s.runners[id] = runner

	return &runner
}

// Assume lock is already held
func (s *Scheduler) generateRunnerID() string {
	for {
		id := uuid.NewString()

		_, exist := s.runners[id]
		if !exist {
			return id
		}
	}
}

func (s *Scheduler) Runners() func(func(*Runner) bool) {
	return func(yield func(*Runner) bool) {
		s.lock.RLock()
		defer s.lock.RUnlock()

		for _, runner := range s.runners {
			if !yield(&runner) {
				break
			}
		}
	}
}

func (s *Scheduler) RemoveRunner(runnerID string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.runners, runnerID)
}
