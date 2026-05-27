package controller

import (
	"errors"
	"testing"
)

func TestRegression_LifecycleSupervisor_stoppingReturnsIdle(t *testing.T) {
	s := NewLifecycleSupervisor()
	err := s.Do(LifecycleStopping, func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if s.State() != LifecycleIdle {
		t.Fatalf("state=%s want idle after stop", s.State())
	}
}

func TestRegression_LifecycleSupervisor_runningAfterStart(t *testing.T) {
	s := NewLifecycleSupervisor()
	err := s.Do(LifecycleStarting, func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if s.State() != LifecycleRunning {
		t.Fatalf("state=%s want running after successful start", s.State())
	}
}

func TestRegression_LifecycleSupervisor_failedStartIdle(t *testing.T) {
	s := NewLifecycleSupervisor()
	_ = s.Do(LifecycleStarting, func() error { return errors.New("start failed") })
	if s.State() != LifecycleIdle {
		t.Fatalf("state=%s want idle after failed start", s.State())
	}
}
