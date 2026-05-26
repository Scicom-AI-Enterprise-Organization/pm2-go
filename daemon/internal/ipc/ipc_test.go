package ipc

import (
	"context"
	"testing"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/daemon"
	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

func setupHome(t *testing.T) {
	t.Helper()
	t.Setenv("PM2_GO_HOME", t.TempDir())
}

// startServer spins up a real supervisor + IPC server on a temp socket and
// returns a client. The cleanup func shuts everything down.
func startServer(t *testing.T) (*Client, *daemon.Supervisor, func()) {
	t.Helper()
	setupHome(t)
	sup := daemon.NewSupervisor()
	srv := NewServer(sup)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = srv.Serve(ctx)
		close(done)
	}()
	// Wait for socket to be ready.
	c := NewClient()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if err := c.Ping(context.Background()); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cleanup := func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("server did not stop")
		}
		sup.Shutdown()
	}
	return c, sup, cleanup
}

func TestIPCRoundTrip(t *testing.T) {
	c, _, cleanup := startServer(t)
	defer cleanup()
	ctx := context.Background()

	if err := c.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	spec := &process.Spec{
		Name:        "sleeper",
		Script:      "/bin/sleep",
		Args:        []string{"30"},
		KillTimeout: 300 * time.Millisecond,
		MinUptime:   100 * time.Millisecond,
		Instances:   1,
	}
	if err := c.StartSpec(ctx, spec); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Wait for the runtime to appear online.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		procs, err := c.List(ctx)
		if err == nil {
			for _, p := range procs {
				if p.Name == "sleeper" && p.State == process.StateOnline {
					goto online
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("sleeper never came online")
online:

	// Describe should include the spec + procs.
	sp, procs, err := c.Describe(ctx, "sleeper")
	if err != nil {
		t.Fatal(err)
	}
	if sp.Name != "sleeper" || len(procs) != 1 {
		t.Errorf("describe = %+v %+v", sp, procs)
	}

	// Stop / verify state.
	if err := c.Stop(ctx, "sleeper"); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		procs, _ := c.List(ctx)
		for _, p := range procs {
			if p.Name == "sleeper" && p.State == process.StateStopped {
				goto stopped
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("sleeper never stopped")
stopped:

	// Delete.
	if err := c.Delete(ctx, "sleeper"); err != nil {
		t.Fatal(err)
	}
	procs, err = c.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range procs {
		if p.Name == "sleeper" {
			t.Fatalf("sleeper still present: %+v", p)
		}
	}
}

func TestIPCNotFound(t *testing.T) {
	c, _, cleanup := startServer(t)
	defer cleanup()
	err := c.Stop(context.Background(), "no-such-app")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIPCSaveAndSpecs(t *testing.T) {
	c, _, cleanup := startServer(t)
	defer cleanup()
	ctx := context.Background()
	spec := &process.Spec{Name: "stored", Script: "/bin/sleep", Args: []string{"60"}}
	if err := c.StartSpec(ctx, spec); err != nil {
		t.Fatal(err)
	}
	if err := c.Save(ctx); err != nil {
		t.Fatal(err)
	}
	specs, err := c.Specs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range specs {
		if s.Name == "stored" {
			found = true
		}
	}
	if !found {
		t.Errorf("Specs returned %+v", specs)
	}
}
