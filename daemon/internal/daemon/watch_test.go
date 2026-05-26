package daemon

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

func TestWatcherFiresOnChange(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "watched.txt")
	if err := os.WriteFile(target, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	var fired int32
	w, err := NewWatcher(
		&process.Spec{Name: "x", Watch: []string{d}},
		func() { atomic.AddInt32(&fired, 1) },
	)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	// Need a slight beat to let fsnotify register the watch before our write.
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(target, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	// debounce is 500ms — wait up to 2s for the trigger.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&fired) >= 1 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("watcher never fired, fired=%d", atomic.LoadInt32(&fired))
}

func TestWatcherIgnoresPattern(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "noisy.log")
	if err := os.WriteFile(target, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	var fired int32
	w, err := NewWatcher(
		&process.Spec{Name: "x", Watch: []string{d}, IgnoreWatch: []string{"*.log"}},
		func() { atomic.AddInt32(&fired, 1) },
	)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	time.Sleep(50 * time.Millisecond)
	// Modify the .log file — should be ignored.
	if err := os.WriteFile(target, []byte("v2 v3 v4"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(800 * time.Millisecond) // > debounce window
	if got := atomic.LoadInt32(&fired); got != 0 {
		t.Errorf("watcher fired %d times despite ignore pattern", got)
	}
}

func TestWatcherSkipsHiddenAndNodeModules(t *testing.T) {
	d := t.TempDir()
	for _, sub := range []string{".git", "node_modules"} {
		if err := os.MkdirAll(filepath.Join(d, sub), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, sub, "f"), []byte("v1"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var fired int32
	w, err := NewWatcher(
		&process.Spec{Name: "x", Watch: []string{d}},
		func() { atomic.AddInt32(&fired, 1) },
	)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	time.Sleep(50 * time.Millisecond)
	for _, sub := range []string{".git", "node_modules"} {
		_ = os.WriteFile(filepath.Join(d, sub, "f"), []byte("v2"), 0o644)
	}
	time.Sleep(800 * time.Millisecond)
	if got := atomic.LoadInt32(&fired); got != 0 {
		t.Errorf("watcher fired for skipped dirs: %d", got)
	}
}
