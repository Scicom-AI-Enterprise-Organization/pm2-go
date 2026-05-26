package daemon

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

func setupHome(t *testing.T) {
	t.Helper()
	t.Setenv("PM2_GO_HOME", t.TempDir())
}

func waitUntil(t *testing.T, d time.Duration, label string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", label)
}

func sleepSpec(name string, seconds int) *process.Spec {
	s := &process.Spec{
		Name:        name,
		Script:      "/bin/sleep",
		Args:        []string{strconvItoa(seconds)},
		KillTimeout: 300 * time.Millisecond,
		MinUptime:   100 * time.Millisecond,
	}
	return s
}

// strconvItoa avoids a separate import.
func strconvItoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func TestSupervisorAddAndStop(t *testing.T) {
	setupHome(t)
	sup := NewSupervisor()
	defer sup.Shutdown()

	if err := sup.Add(sleepSpec("alpha", 60)); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 3*time.Second, "alpha to be online", func() bool {
		for _, v := range sup.List() {
			if v.Name == "alpha" && v.State == process.StateOnline {
				return true
			}
		}
		return false
	})

	if err := sup.Stop("alpha"); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 3*time.Second, "alpha to be stopped", func() bool {
		for _, v := range sup.List() {
			if v.Name == "alpha" && v.State == process.StateStopped {
				return true
			}
		}
		return false
	})
}

func TestSupervisorRestartIncreasesRestartCount(t *testing.T) {
	setupHome(t)
	sup := NewSupervisor()
	defer sup.Shutdown()

	if err := sup.Add(sleepSpec("beta", 60)); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 3*time.Second, "beta online", func() bool {
		for _, v := range sup.List() {
			if v.Name == "beta" && v.State == process.StateOnline {
				return true
			}
		}
		return false
	})

	if err := sup.Restart("beta"); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 3*time.Second, "beta online again", func() bool {
		for _, v := range sup.List() {
			if v.Name == "beta" && v.State == process.StateOnline && v.Restarts >= 1 {
				return true
			}
		}
		return false
	})
}

func TestSupervisorDeleteRemovesRuntimes(t *testing.T) {
	setupHome(t)
	sup := NewSupervisor()
	defer sup.Shutdown()

	if err := sup.Add(sleepSpec("gamma", 60)); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 3*time.Second, "gamma online", func() bool {
		for _, v := range sup.List() {
			if v.Name == "gamma" && v.State == process.StateOnline {
				return true
			}
		}
		return false
	})
	if err := sup.Delete("gamma"); err != nil {
		t.Fatal(err)
	}
	for _, v := range sup.List() {
		if v.Name == "gamma" {
			t.Fatalf("gamma still present: %+v", v)
		}
	}
	if sup.findSpec("gamma") != nil {
		t.Fatal("spec still registered")
	}
}

func TestSupervisorAddReplacesExisting(t *testing.T) {
	setupHome(t)
	sup := NewSupervisor()
	defer sup.Shutdown()

	if err := sup.Add(sleepSpec("delta", 60)); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 3*time.Second, "delta v1 online", func() bool {
		for _, v := range sup.List() {
			if v.Name == "delta" && v.State == process.StateOnline {
				return true
			}
		}
		return false
	})

	// Add again with a different sleep time; daemon should stop + restart.
	if err := sup.Add(sleepSpec("delta", 30)); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 5*time.Second, "delta v2 online", func() bool {
		for _, v := range sup.List() {
			if v.Name == "delta" && v.State == process.StateOnline {
				return true
			}
		}
		return false
	})
}

func TestSupervisorNotFoundError(t *testing.T) {
	setupHome(t)
	sup := NewSupervisor()
	defer sup.Shutdown()
	err := sup.Stop("nope")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestSupervisorClusterMode(t *testing.T) {
	setupHome(t)
	sup := NewSupervisor()
	defer sup.Shutdown()
	s := sleepSpec("cluster", 60)
	s.Instances = 3
	if err := sup.Add(s); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 4*time.Second, "3 cluster instances online", func() bool {
		online := 0
		for _, v := range sup.List() {
			if v.Name == "cluster" && v.State == process.StateOnline {
				online++
			}
		}
		return online == 3
	})
	// Each instance should have a distinct InstanceID.
	ids := map[int]bool{}
	for _, v := range sup.List() {
		if v.Name == "cluster" {
			ids[v.InstanceID] = true
		}
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 distinct InstanceIDs, got %v", ids)
	}
}

func TestSupervisorAutorestartDisabled(t *testing.T) {
	setupHome(t)
	sup := NewSupervisor()
	defer sup.Shutdown()
	// /bin/true exits immediately with code 0.
	s := &process.Spec{
		Name:                "noresurrect",
		Script:              "/bin/true",
		KillTimeout:         200 * time.Millisecond,
		MinUptime:           50 * time.Millisecond,
		AutorestartDisabled: true,
	}
	if err := sup.Add(s); err != nil {
		t.Fatal(err)
	}
	// After a moment we expect Stopped, no restarts.
	waitUntil(t, 3*time.Second, "stopped with 0 restarts", func() bool {
		for _, v := range sup.List() {
			if v.Name == "noresurrect" && v.State == process.StateStopped && v.Restarts == 0 {
				return true
			}
		}
		return false
	})
}

func TestSupervisorErroredAfterMaxUnstable(t *testing.T) {
	setupHome(t)
	sup := NewSupervisor()
	defer sup.Shutdown()
	// /bin/false exits with code 1 immediately; min_uptime is 1s so every exit is unstable.
	s := &process.Spec{
		Name:        "flapping",
		Script:      "/bin/false",
		KillTimeout: 200 * time.Millisecond,
		MinUptime:   1 * time.Second,
		MaxRestarts: 2, // very low cap so we hit errored fast
	}
	if err := sup.Add(s); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 5*time.Second, "errored", func() bool {
		for _, v := range sup.List() {
			if v.Name == "flapping" && v.State == process.StateErrored {
				return true
			}
		}
		return false
	})
}

func TestSupervisorStopExitCodes(t *testing.T) {
	setupHome(t)
	sup := NewSupervisor()
	defer sup.Shutdown()
	s := &process.Spec{
		Name:          "graceful-exit",
		Script:        "/bin/true", // exit 0
		KillTimeout:   200 * time.Millisecond,
		MinUptime:     50 * time.Millisecond,
		StopExitCodes: []int{0},
	}
	if err := sup.Add(s); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 3*time.Second, "stopped via stop_exit_codes", func() bool {
		for _, v := range sup.List() {
			if v.Name == "graceful-exit" && v.State == process.StateStopped && v.Restarts == 0 {
				return true
			}
		}
		return false
	})
}

func TestSupervisorLogsCapture(t *testing.T) {
	setupHome(t)
	sup := NewSupervisor()
	defer sup.Shutdown()
	s := &process.Spec{
		Name:                "echoer",
		Script:              "/bin/sh",
		Args:                []string{"-c", "echo hello && sleep 5"},
		KillTimeout:         200 * time.Millisecond,
		MinUptime:           50 * time.Millisecond,
		AutorestartDisabled: true,
	}
	if err := sup.Add(s); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 3*time.Second, "echoer logs present", func() bool {
		out, _ := sup.TailLogs("echoer", "out", 5)
		for _, lines := range out {
			for _, l := range lines {
				if strings.Contains(l, "hello") {
					return true
				}
			}
		}
		return false
	})
}

func TestSupervisorResurrect(t *testing.T) {
	setupHome(t)
	sup := NewSupervisor()
	s := sleepSpec("persist-me", 60)
	if err := sup.Add(s); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 3*time.Second, "persist-me online", func() bool {
		for _, v := range sup.List() {
			if v.Name == "persist-me" && v.State == process.StateOnline {
				return true
			}
		}
		return false
	})
	if err := sup.Save(); err != nil {
		t.Fatal(err)
	}
	sup.Shutdown()

	// New supervisor reads dump and brings it back up.
	sup2 := NewSupervisor()
	defer sup2.Shutdown()
	if err := sup2.Resurrect(); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 4*time.Second, "resurrected online", func() bool {
		for _, v := range sup2.List() {
			if v.Name == "persist-me" && v.State == process.StateOnline {
				return true
			}
		}
		return false
	})
}

func TestTailLogsReturnsEmptySliceNotNil(t *testing.T) {
	setupHome(t)
	sup := NewSupervisor()
	defer sup.Shutdown()
	// Spawn /bin/sleep with no output — both out and err files exist (created
	// at spawn time) but are empty. The map values must serialise as [] in JSON,
	// not null, so TS clients can iterate them safely.
	if err := sup.Add(sleepSpec("quiet", 60)); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, 3*time.Second, "quiet online", func() bool {
		for _, v := range sup.List() {
			if v.Name == "quiet" && v.State == process.StateOnline {
				return true
			}
		}
		return false
	})
	streams, err := sup.TailLogs("quiet", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) == 0 {
		t.Fatal("expected at least one stream key")
	}
	for k, v := range streams {
		if v == nil {
			t.Errorf("stream %q is nil; should be empty slice for JSON []", k)
		}
	}
}

func TestRuntimeID(t *testing.T) {
	single := &process.Spec{Name: "a", Instances: 1}
	if id := runtimeID(single, 0); id != "a" {
		t.Errorf("single instance: %q", id)
	}
	multi := &process.Spec{Name: "a", Instances: 3}
	if id := runtimeID(multi, 1); id != "a-1" {
		t.Errorf("cluster: %q", id)
	}
}

func TestExitCodeFromErr(t *testing.T) {
	if exitCodeFromErr(nil) != 0 {
		t.Error("nil err should be 0")
	}
}
