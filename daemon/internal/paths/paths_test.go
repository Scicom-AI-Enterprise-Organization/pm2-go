package paths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHomeEnvOverride(t *testing.T) {
	d := t.TempDir()
	t.Setenv("PM2_GO_HOME", d)
	got := Home()
	if got != d {
		t.Errorf("Home()=%q, want %q", got, d)
	}
}

func TestHomeCreatesDir(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "nested", "pm2-go-home")
	t.Setenv("PM2_GO_HOME", target)
	_ = Home()
	if _, err := os.Stat(target); err != nil {
		t.Errorf("Home() did not create dir: %v", err)
	}
}

func TestSubdirs(t *testing.T) {
	d := t.TempDir()
	t.Setenv("PM2_GO_HOME", d)
	for _, fn := range []func() string{LogsDir, PidsDir} {
		p := fn()
		if !strings.HasPrefix(p, d) {
			t.Errorf("subdir %q not under home %q", p, d)
		}
		if _, err := os.Stat(p); err != nil {
			t.Errorf("subdir %q not created: %v", p, err)
		}
	}
}

func TestPathsAreDeterministic(t *testing.T) {
	t.Setenv("PM2_GO_HOME", t.TempDir())
	if RPCSocket() != RPCSocket() {
		t.Error("RPCSocket not deterministic")
	}
	if DumpFile() != DumpFile() {
		t.Error("DumpFile not deterministic")
	}
}
