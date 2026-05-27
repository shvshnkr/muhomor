package controller

import "sync"

type LifecycleState string

const (
	LifecycleIdle      LifecycleState = "idle"
	LifecycleStarting  LifecycleState = "starting"
	LifecycleRunning   LifecycleState = "running"
	LifecycleReloading LifecycleState = "reloading"
	LifecycleStopping  LifecycleState = "stopping"
)

type LifecycleSupervisor struct {
	mu    sync.Mutex
	state LifecycleState
}

func NewLifecycleSupervisor() *LifecycleSupervisor {
	return &LifecycleSupervisor{state: LifecycleIdle}
}

func (s *LifecycleSupervisor) Do(state LifecycleState, fn func() error) error {
	s.mu.Lock()
	s.state = state
	var err error
	defer func() {
		if err != nil || state == LifecycleStopping {
			s.state = LifecycleIdle
		} else {
			s.state = LifecycleRunning
		}
		s.mu.Unlock()
	}()
	err = fn()
	return err
}

func (s *LifecycleSupervisor) State() LifecycleState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}
