package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

// chdirHome forces paths.Home() to use a per-test temp dir via PM2_GO_HOME.
func chdirHome(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	t.Setenv("PM2_GO_HOME", d)
	return d
}

func TestStoreLoadEmpty(t *testing.T) {
	chdirHome(t)
	s := New()
	d, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Specs) != 0 || d.Version != 1 {
		t.Errorf("empty load: %+v", d)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	chdirHome(t)
	s := New()
	specs := []*process.Spec{
		{Name: "a", Script: "/bin/true", Instances: 1, Namespace: "default"},
		{Name: "b", Script: "/bin/false", Instances: 2, Namespace: "prod"},
	}
	for _, sp := range specs {
		sp.Defaults()
	}
	if err := s.Save(specs); err != nil {
		t.Fatal(err)
	}

	// Fresh store reads from disk.
	s2 := New()
	d, err := s2.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Specs) != 2 {
		t.Fatalf("len=%d", len(d.Specs))
	}
	if d.Specs[0].Name != "a" || d.Specs[1].Namespace != "prod" {
		t.Errorf("round-trip mismatch: %+v", d.Specs)
	}
}

func TestStoreSaveAtomic(t *testing.T) {
	chdirHome(t)
	s := New()
	if err := s.Save([]*process.Spec{{Name: "x", Script: "/bin/true"}}); err != nil {
		t.Fatal(err)
	}
	// Confirm no leftover .tmp file.
	matches, _ := filepath.Glob(s.Path() + ".tmp")
	if len(matches) > 0 {
		t.Errorf(".tmp file leaked: %v", matches)
	}
}

func TestStoreLoadCorrupt(t *testing.T) {
	dir := chdirHome(t)
	if err := os.WriteFile(filepath.Join(dir, "dump.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New()
	_, err := s.Load()
	if err == nil {
		t.Fatal("expected error on corrupt dump.json")
	}
}

func TestStoreCacheReturnsSameInstance(t *testing.T) {
	chdirHome(t)
	s := New()
	a, _ := s.Load()
	b, _ := s.Load()
	if a != b {
		t.Error("Load() should return the same cached dump")
	}
}
