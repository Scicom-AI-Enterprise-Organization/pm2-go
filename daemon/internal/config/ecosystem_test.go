package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTemp(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadEcosystemBasic(t *testing.T) {
	body := `{
		"apps": [
			{
				"name": "api",
				"script": "./index.js",
				"interpreter": "node",
				"instances": 2,
				"namespace": "prod",
				"env": {"NODE_ENV": "production"},
				"watch": ["src"],
				"ignore_watch": ["*.log"],
				"max_memory_restart": "300M",
				"min_uptime": "5s",
				"restart_delay": 0.5,
				"kill_timeout": "3s",
				"max_restarts": 8,
				"exp_backoff_restart": true,
				"stop_exit_codes": [0, 42],
				"autorestart": true
			}
		]
	}`
	p := writeTemp(t, "ecosystem.config.json", body)
	specs, err := LoadEcosystem(p)
	if err != nil {
		t.Fatalf("LoadEcosystem: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("len(specs)=%d", len(specs))
	}
	s := specs[0]
	if s.Name != "api" || s.Interpreter != "node" || s.Instances != 2 {
		t.Errorf("basic fields: %+v", s)
	}
	if !filepath.IsAbs(s.Script) {
		t.Errorf("Script should be made absolute: %q", s.Script)
	}
	if s.Env["NODE_ENV"] != "production" {
		t.Errorf("env not loaded: %v", s.Env)
	}
	if s.MaxMemoryRestart != 300*1024*1024 {
		t.Errorf("max_memory_restart = %d, want %d", s.MaxMemoryRestart, 300*1024*1024)
	}
	if s.MinUptime != 5*time.Second {
		t.Errorf("min_uptime = %v", s.MinUptime)
	}
	if s.RestartDelay != 500*time.Millisecond {
		t.Errorf("restart_delay = %v", s.RestartDelay)
	}
	if s.KillTimeout != 3*time.Second {
		t.Errorf("kill_timeout = %v", s.KillTimeout)
	}
	if s.MaxRestarts != 8 {
		t.Errorf("max_restarts = %d", s.MaxRestarts)
	}
	if !s.ExpBackoffRestart {
		t.Error("exp_backoff_restart not loaded")
	}
	if len(s.StopExitCodes) != 2 || s.StopExitCodes[1] != 42 {
		t.Errorf("stop_exit_codes = %v", s.StopExitCodes)
	}
	if s.AutorestartDisabled {
		t.Error("autorestart=true should leave Disabled=false")
	}
}

func TestLoadEcosystemAutorestartFalse(t *testing.T) {
	body := `{"apps":[{"name":"x","script":"/bin/true","autorestart":false}]}`
	p := writeTemp(t, "e.json", body)
	specs, err := LoadEcosystem(p)
	if err != nil {
		t.Fatal(err)
	}
	if !specs[0].AutorestartDisabled {
		t.Error("autorestart=false should set AutorestartDisabled=true")
	}
}

func TestLoadEcosystemArgsVariants(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{"string", `{"apps":[{"name":"x","script":"/bin/true","args":"--port 3000 --verbose"}]}`, []string{"--port", "3000", "--verbose"}},
		{"array", `{"apps":[{"name":"x","script":"/bin/true","args":["--port","3000"]}]}`, []string{"--port", "3000"}},
		{"empty string", `{"apps":[{"name":"x","script":"/bin/true","args":""}]}`, nil},
		{"missing", `{"apps":[{"name":"x","script":"/bin/true"}]}`, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := writeTemp(t, "e.json", c.body)
			specs, err := LoadEcosystem(p)
			if err != nil {
				t.Fatal(err)
			}
			if len(specs[0].Args) != len(c.want) {
				t.Fatalf("len got=%d want=%d (%v vs %v)", len(specs[0].Args), len(c.want), specs[0].Args, c.want)
			}
			for i := range c.want {
				if specs[0].Args[i] != c.want[i] {
					t.Errorf("args[%d]=%q want %q", i, specs[0].Args[i], c.want[i])
				}
			}
		})
	}
}

func TestLoadEcosystemMemoryVariants(t *testing.T) {
	cases := []struct {
		body string
		want uint64
	}{
		{`{"apps":[{"name":"x","script":"/bin/true","max_memory_restart":"500M"}]}`, 500 * 1024 * 1024},
		{`{"apps":[{"name":"x","script":"/bin/true","max_memory_restart":"1G"}]}`, 1024 * 1024 * 1024},
		{`{"apps":[{"name":"x","script":"/bin/true","max_memory_restart":"512k"}]}`, 512 * 1024},
		{`{"apps":[{"name":"x","script":"/bin/true","max_memory_restart":1048576}]}`, 1048576},
	}
	for _, c := range cases {
		p := writeTemp(t, "e.json", c.body)
		specs, err := LoadEcosystem(p)
		if err != nil {
			t.Fatal(err)
		}
		if specs[0].MaxMemoryRestart != c.want {
			t.Errorf("got %d want %d for body %s", specs[0].MaxMemoryRestart, c.want, c.body)
		}
	}
}

func TestLoadEcosystemBareArray(t *testing.T) {
	body := `[{"name":"a","script":"/bin/true"},{"name":"b","script":"/bin/false"}]`
	p := writeTemp(t, "e.json", body)
	specs, err := LoadEcosystem(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 2 || specs[0].Name != "a" || specs[1].Name != "b" {
		t.Errorf("got %+v", specs)
	}
}

func TestLoadEcosystemValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string // substring of expected error
	}{
		{"missing name", `{"apps":[{"script":"/bin/true"}]}`, "name is required"},
		{"missing script", `{"apps":[{"name":"x"}]}`, "script is required"},
		{"malformed", `not json`, "parse"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := writeTemp(t, "e.json", c.body)
			_, err := LoadEcosystem(p)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("err=%q want substring %q", err.Error(), c.want)
			}
		})
	}
}

func TestLoadEcosystemYAMLRejected(t *testing.T) {
	p := writeTemp(t, "e.yaml", "apps:\n  - name: a\n    script: /bin/true\n")
	_, err := LoadEcosystem(p)
	if err == nil {
		t.Fatal("expected yaml-not-supported error")
	}
}

func TestLoadEcosystemRelativeCwd(t *testing.T) {
	body := `{"apps":[{"name":"x","script":"./bin/x","cwd":"./pkg/x"}]}`
	p := writeTemp(t, "e.json", body)
	specs, err := LoadEcosystem(p)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(specs[0].Cwd) {
		t.Errorf("Cwd should be made absolute: %q", specs[0].Cwd)
	}
}
