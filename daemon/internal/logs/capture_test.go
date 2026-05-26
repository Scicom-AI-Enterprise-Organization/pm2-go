package logs

import (
	"io"
	"strings"
	"testing"
	"time"
)

func setHome(t *testing.T) {
	t.Helper()
	t.Setenv("PM2_GO_HOME", t.TempDir())
}

func TestOpenAndAttach(t *testing.T) {
	setHome(t)
	s, err := Open("rt1", "out", RotateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	r, w := io.Pipe()
	s.Attach("rt1", "out", r)
	if _, err := w.Write([]byte("hello\nworld\n")); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	// give the goroutine a moment to flush
	deadline := time.Now().Add(2 * time.Second)
	var lines []string
	for time.Now().Before(deadline) {
		lines, _ = Tail("rt1", "out", 10)
		if len(lines) >= 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(lines) != 2 || lines[0] != "hello" || lines[1] != "world" {
		t.Errorf("tail=%v", lines)
	}
}

func TestSubscribeReceivesLines(t *testing.T) {
	setHome(t)
	s, err := Open("rt2", "out", RotateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	ch := s.Subscribe(8)
	r, w := io.Pipe()
	s.Attach("rt2", "out", r)
	go func() {
		_, _ = w.Write([]byte("subscribed line\n"))
		_ = w.Close()
	}()
	select {
	case l := <-ch:
		if !strings.Contains(l.Text, "subscribed line") {
			t.Errorf("got line %q", l.Text)
		}
		if l.Stream != "out" {
			t.Errorf("stream=%q", l.Stream)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subscriber")
	}
}

func TestTailMissingFile(t *testing.T) {
	setHome(t)
	lines, err := Tail("does-not-exist", "out", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("expected empty, got %v", lines)
	}
}

func TestTailLastN(t *testing.T) {
	setHome(t)
	s, err := Open("rt3", "out", RotateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	r, w := io.Pipe()
	s.Attach("rt3", "out", r)
	go func() {
		for i := 0; i < 50; i++ {
			_, _ = w.Write([]byte("line\n"))
		}
		_ = w.Close()
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		lines, _ := Tail("rt3", "out", 5)
		if len(lines) == 5 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("never saw 5 lines")
}

func TestRotationOptsCreatesLumberjack(t *testing.T) {
	setHome(t)
	s, err := Open("rt4", "out", RotateOpts{MaxSizeMB: 1, MaxBackups: 1, MaxAgeDays: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// Sanity-write that the path exists once we attach + flush.
	r, w := io.Pipe()
	s.Attach("rt4", "out", r)
	_, _ = w.Write([]byte("rot test\n"))
	_ = w.Close()
	time.Sleep(50 * time.Millisecond)
	lines, _ := Tail("rt4", "out", 1)
	if len(lines) == 0 {
		t.Error("rotated writer produced no readable output")
	}
}

func TestFilePathDeterministic(t *testing.T) {
	setHome(t)
	a := FilePath("foo", "out")
	b := FilePath("foo", "out")
	if a != b {
		t.Errorf("FilePath not deterministic: %q vs %q", a, b)
	}
}
