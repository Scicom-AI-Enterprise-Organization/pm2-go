package metrics

import (
	"os"
	"runtime"
	"testing"
	"time"
)

func TestReadProcSelf(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("non-linux")
	}
	s, err := readProc(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if s.rssBytes == 0 {
		t.Error("rssBytes == 0 for self")
	}
}

func TestReadProcMissingPID(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("non-linux")
	}
	_, err := readProc(1 << 30) // unrealistically large PID
	if err == nil {
		t.Error("expected error for missing pid")
	}
}

func TestCollectorTickAndLatest(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("non-linux")
	}
	c := New(os.Getpid(), 4)
	// First tick has no delta, so CPU should be 0.
	c.Tick()
	if l := c.Latest(); l.MemBytes == 0 {
		t.Error("first sample mem == 0")
	}
	time.Sleep(50 * time.Millisecond)
	// Spin to generate cpu usage.
	for i := 0; i < 1_000_000; i++ {
		_ = i * i
	}
	c.Tick()
	if l := c.Latest(); l.Time.IsZero() {
		t.Error("Latest.Time should be set")
	}
}

func TestCollectorWindowRing(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("non-linux")
	}
	c := New(os.Getpid(), 3)
	for i := 0; i < 5; i++ {
		c.Tick()
		time.Sleep(5 * time.Millisecond)
	}
	w := c.Window()
	if len(w) != 3 {
		t.Errorf("window len = %d, want 3 (rolling)", len(w))
	}
}

func TestCollectorTickIgnoresUnreadable(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("non-linux")
	}
	c := New(1<<30, 4) // bogus PID
	c.Tick()           // must not panic
	if l := c.Latest(); l.MemBytes != 0 {
		t.Errorf("bogus PID produced sample: %+v", l)
	}
}
