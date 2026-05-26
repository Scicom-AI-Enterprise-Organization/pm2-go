package api

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/daemon"
	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

// freePort returns a port that's free at the moment of the call.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

func setupHome(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	t.Setenv("PM2_GO_HOME", d)
	return d
}

// startAPI configures api-port + api-token, boots the API server, and returns
// the bound URL + token.
func startAPI(t *testing.T) (string, string, *daemon.Supervisor, func()) {
	t.Helper()
	home := setupHome(t)
	port := freePort(t)
	addr := "127.0.0.1:" + itoa(port)
	if err := os.WriteFile(filepath.Join(home, "api-port"), []byte(addr), 0o644); err != nil {
		t.Fatal(err)
	}
	token := "testtoken123"
	if err := os.WriteFile(filepath.Join(home, "api-token"), []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
	sup := daemon.NewSupervisor()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = Serve(ctx, sup)
		close(done)
	}()
	url := "http://" + addr
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if r, err := http.Get(url + "/healthz"); err == nil {
			_ = r.Body.Close()
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	cleanup := func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("api server hang")
		}
		sup.Shutdown()
	}
	return url, token, sup, cleanup
}

func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [16]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = digits[i%10]
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func get(t *testing.T, url, token string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func post(t *testing.T, url, token, body string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, _ := http.NewRequest(http.MethodPost, url, reader)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestAPIHealthzNoAuth(t *testing.T) {
	url, _, _, cleanup := startAPI(t)
	defer cleanup()
	r := get(t, url+"/healthz", "")
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		t.Errorf("status=%d", r.StatusCode)
	}
}

func TestAPIRequiresAuth(t *testing.T) {
	url, _, _, cleanup := startAPI(t)
	defer cleanup()
	r := get(t, url+"/v1/procs", "")
	defer r.Body.Close()
	if r.StatusCode != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", r.StatusCode)
	}
}

func TestAPIRejectsWrongToken(t *testing.T) {
	url, _, _, cleanup := startAPI(t)
	defer cleanup()
	r := get(t, url+"/v1/procs", "wrong")
	defer r.Body.Close()
	if r.StatusCode != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", r.StatusCode)
	}
}

func TestAPIStartAndList(t *testing.T) {
	url, token, _, cleanup := startAPI(t)
	defer cleanup()

	spec := process.Spec{
		Name:        "api-test",
		Script:      "/bin/sleep",
		Args:        []string{"30"},
		KillTimeout: 300 * time.Millisecond,
		MinUptime:   100 * time.Millisecond,
		Instances:   1,
	}
	body, _ := json.Marshal(spec)
	r := post(t, url+"/v1/start", token, string(body))
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(r.Body)
		t.Fatalf("start: %d %s", r.StatusCode, b)
	}

	// Wait for it to appear in /v1/procs.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		r := get(t, url+"/v1/procs", token)
		var data struct {
			Procs []process.View `json:"procs"`
		}
		_ = json.NewDecoder(r.Body).Decode(&data)
		r.Body.Close()
		for _, p := range data.Procs {
			if p.Name == "api-test" && p.State == process.StateOnline {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("api-test never appeared online")
}

func TestAPIStopRestartDelete(t *testing.T) {
	url, token, _, cleanup := startAPI(t)
	defer cleanup()

	spec := process.Spec{
		Name:        "lifecycle",
		Script:      "/bin/sleep",
		Args:        []string{"30"},
		KillTimeout: 300 * time.Millisecond,
		MinUptime:   100 * time.Millisecond,
		Instances:   1,
	}
	body, _ := json.Marshal(spec)
	r := post(t, url+"/v1/start", token, string(body))
	r.Body.Close()

	// Wait for online.
	waitUntilOnline := func(name string) {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			r := get(t, url+"/v1/procs", token)
			var data struct {
				Procs []process.View `json:"procs"`
			}
			_ = json.NewDecoder(r.Body).Decode(&data)
			r.Body.Close()
			for _, p := range data.Procs {
				if p.Name == name && p.State == process.StateOnline {
					return
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatalf("%s never online", name)
	}
	waitUntilOnline("lifecycle")

	// Restart bumps the count.
	r = post(t, url+"/v1/restart?name=lifecycle", token, "")
	if r.StatusCode != http.StatusOK {
		t.Errorf("restart status=%d", r.StatusCode)
	}
	r.Body.Close()
	waitUntilOnline("lifecycle")

	// Reload returns 200 even if SIGUSR2 is a no-op for sleep.
	r = post(t, url+"/v1/reload?name=lifecycle", token, "")
	if r.StatusCode != http.StatusOK {
		t.Errorf("reload status=%d", r.StatusCode)
	}
	r.Body.Close()

	// Stop.
	r = post(t, url+"/v1/stop?name=lifecycle", token, "")
	if r.StatusCode != http.StatusOK {
		t.Errorf("stop status=%d", r.StatusCode)
	}
	r.Body.Close()

	// Delete.
	r = post(t, url+"/v1/delete?name=lifecycle", token, "")
	if r.StatusCode != http.StatusOK {
		t.Errorf("delete status=%d", r.StatusCode)
	}
	r.Body.Close()
}

func TestAPIBulkEndpoints(t *testing.T) {
	url, token, _, cleanup := startAPI(t)
	defer cleanup()

	// Bulk endpoints accept POST with no body.
	for _, path := range []string{"/v1/save", "/v1/start-all", "/v1/stop-all", "/v1/delete-all"} {
		r := post(t, url+path, token, "")
		if r.StatusCode != http.StatusOK {
			t.Errorf("%s: status=%d", path, r.StatusCode)
		}
		r.Body.Close()
	}
}

func TestAPIMetricsEndpoint(t *testing.T) {
	url, token, sup, cleanup := startAPI(t)
	defer cleanup()

	// Use the supervisor directly to add a process, then poll /v1/metrics.
	spec := &process.Spec{
		Name: "metric-target", Script: "/bin/sleep", Args: []string{"30"},
		KillTimeout: 300 * time.Millisecond, MinUptime: 100 * time.Millisecond,
		Instances: 1,
	}
	if err := sup.Add(spec); err != nil {
		t.Fatal(err)
	}
	// Wait briefly for at least one metrics tick.
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		r := get(t, url+"/v1/metrics?name=metric-target", token)
		var data struct {
			Series map[string][]struct{ CPU float64 } `json:"series"`
		}
		_ = json.NewDecoder(r.Body).Decode(&data)
		r.Body.Close()
		for _, s := range data.Series {
			if len(s) > 0 {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("metrics never reported samples")
}

func TestAPINotFoundReturns404(t *testing.T) {
	url, token, _, cleanup := startAPI(t)
	defer cleanup()
	r := post(t, url+"/v1/stop?name=does-not-exist", token, "")
	defer r.Body.Close()
	if r.StatusCode != http.StatusNotFound {
		t.Errorf("status=%d, want 404", r.StatusCode)
	}
}

func TestAPIDisabledWithoutConfig(t *testing.T) {
	setupHome(t) // no api-port / api-token files
	sup := daemon.NewSupervisor()
	defer sup.Shutdown()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Serve should return nil immediately.
	err := Serve(ctx, sup)
	if err != nil {
		t.Errorf("expected nil for unconfigured api, got %v", err)
	}
}
