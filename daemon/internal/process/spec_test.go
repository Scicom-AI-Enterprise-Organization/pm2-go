package process

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSpecDefaults(t *testing.T) {
	t.Run("fills in unset fields", func(t *testing.T) {
		s := &Spec{Name: "x", Script: "/bin/true"}
		s.Defaults()
		if s.Instances != 1 {
			t.Errorf("Instances = %d, want 1", s.Instances)
		}
		if s.ExecMode != ExecFork {
			t.Errorf("ExecMode = %q, want fork", s.ExecMode)
		}
		if s.Namespace != "default" {
			t.Errorf("Namespace = %q, want default", s.Namespace)
		}
		if s.KillTimeout == 0 {
			t.Error("KillTimeout was not set")
		}
		if s.MinUptime == 0 {
			t.Error("MinUptime was not set")
		}
		if s.MaxRestarts == 0 {
			t.Error("MaxRestarts was not set")
		}
		if s.CreatedAt.IsZero() || s.UpdatedAt.IsZero() {
			t.Error("timestamps were not set")
		}
	})

	t.Run("preserves explicit values", func(t *testing.T) {
		now := time.Now()
		s := &Spec{
			Name:        "x",
			Script:      "/bin/true",
			Instances:   3,
			Namespace:   "prod",
			KillTimeout: 7 * time.Second,
			MinUptime:   500 * time.Millisecond,
			MaxRestarts: 99,
			CreatedAt:   now,
		}
		s.Defaults()
		if s.Instances != 3 || s.Namespace != "prod" || s.MaxRestarts != 99 {
			t.Errorf("Defaults overwrote explicit values: %+v", s)
		}
		if !s.CreatedAt.Equal(now) {
			t.Error("CreatedAt was clobbered")
		}
	})

	t.Run("autorestart helper reflects disabled flag", func(t *testing.T) {
		s := &Spec{}
		if !s.Autorestart() {
			t.Error("default Autorestart() should be true")
		}
		s.AutorestartDisabled = true
		if s.Autorestart() {
			t.Error("AutorestartDisabled=true should yield false")
		}
	})
}

func TestSpecValidate(t *testing.T) {
	cases := []struct {
		name    string
		spec    Spec
		wantErr bool
	}{
		{"missing name", Spec{Script: "/bin/true"}, true},
		{"missing script", Spec{Name: "x"}, true},
		{"ok", Spec{Name: "x", Script: "/bin/true"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.spec.Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("err=%v want=%v", err, c.wantErr)
			}
		})
	}
}

func TestSpecCloneIsDeep(t *testing.T) {
	original := &Spec{
		Name:   "x",
		Script: "/bin/true",
		Args:   []string{"a", "b"},
		Env:    map[string]string{"K": "v"},
	}
	cp := original.Clone()
	cp.Args[0] = "mutated"
	cp.Env["K"] = "mutated"
	if original.Args[0] == "mutated" {
		t.Error("Args were shared, not cloned")
	}
	if original.Env["K"] == "mutated" {
		t.Error("Env was shared, not cloned")
	}
}

func TestRuntimeSnapshotUptime(t *testing.T) {
	rt := &Runtime{
		ID:        "x-0",
		Name:      "x",
		State:     StateOnline,
		StartedAt: time.Now().Add(-3 * time.Second),
		PID:       1234,
	}
	v := rt.Snapshot()
	if v.UptimeSeconds < 2 || v.UptimeSeconds > 4 {
		t.Errorf("UptimeSeconds = %d, want ~3", v.UptimeSeconds)
	}
	if v.PID != 1234 {
		t.Errorf("PID = %d", v.PID)
	}
}

func TestRuntimeSnapshotStoppedHasNoUptime(t *testing.T) {
	rt := &Runtime{State: StateStopped, StartedAt: time.Now().Add(-10 * time.Second)}
	v := rt.Snapshot()
	if v.UptimeSeconds != 0 {
		t.Errorf("stopped UptimeSeconds = %d, want 0", v.UptimeSeconds)
	}
}

func TestSpecJSONRoundTrip(t *testing.T) {
	original := &Spec{Name: "x", Script: "/bin/true", Args: []string{"-a"}, Instances: 2}
	original.Defaults()
	b, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var got Spec
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != original.Name || got.Instances != original.Instances {
		t.Errorf("round-trip mismatch: got=%+v want=%+v", got, original)
	}
}
