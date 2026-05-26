package process

import (
	"testing"
	"time"
)

// Edge cases for Defaults() & helpers that are easy to drop coverage on.

func TestSpecDefaultsZeroDurations(t *testing.T) {
	s := &Spec{Name: "x", Script: "/bin/true"}
	s.Defaults()
	if s.KillTimeout <= 0 {
		t.Errorf("KillTimeout default should be > 0, got %v", s.KillTimeout)
	}
	if s.MinUptime <= 0 {
		t.Errorf("MinUptime default should be > 0, got %v", s.MinUptime)
	}
}

func TestRuntimeSnapshotLaunchingHasNoUptime(t *testing.T) {
	rt := &Runtime{State: StateLaunching, StartedAt: time.Now().Add(-3 * time.Second)}
	v := rt.Snapshot()
	if v.UptimeSeconds != 0 {
		t.Errorf("launching should report 0 uptime, got %d", v.UptimeSeconds)
	}
	if v.StartedAt == "" {
		t.Error("StartedAt should still be set for launching")
	}
}

func TestSpecCloneHandlesNilMaps(t *testing.T) {
	s := &Spec{Name: "x", Script: "/bin/true"}
	cp := s.Clone()
	if cp.Name != "x" {
		t.Errorf("clone broke: %+v", cp)
	}
}
