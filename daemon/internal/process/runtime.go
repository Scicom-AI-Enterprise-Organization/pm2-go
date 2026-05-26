package process

import (
	"os/exec"
	"sync"
	"time"
)

type State string

const (
	StateLaunching     State = "launching"
	StateOnline        State = "online"
	StateStopping      State = "stopping"
	StateStopped       State = "stopped"
	StateErrored       State = "errored"
	StateWaitRestart   State = "waiting_restart"
	StateOnlineRestart State = "online_restarting"
)

// Runtime is a single running instance of a Spec. For a Spec with Instances=N,
// the daemon creates N Runtimes (instance ids 0..N-1).
type Runtime struct {
	mu sync.RWMutex

	ID         string    `json:"id"`         // unique runtime id, e.g. "myapp-0"
	AppID      string    `json:"app_id"`     // parent Spec.ID
	Name       string    `json:"name"`       // Spec.Name
	InstanceID int       `json:"instance_id"`
	Namespace  string    `json:"namespace"`

	State       State     `json:"state"`
	PID         int       `json:"pid"`
	StartedAt   time.Time `json:"started_at"`
	ExitedAt    time.Time `json:"exited_at,omitempty"`
	ExitCode    int       `json:"exit_code"`
	Restarts    int       `json:"restarts"`
	UnstableRestarts int  `json:"unstable_restarts"`
	LastError   string    `json:"last_error,omitempty"`

	CPU      float64 `json:"cpu"`     // %
	MemBytes uint64  `json:"mem"`     // RSS
	Uptime   string  `json:"uptime"`  // formatted for display

	cmd       *exec.Cmd
	stopCh    chan struct{}
	manualStop bool // set when user-initiated stop, so supervisor doesn't restart
}

// View is the read-only snapshot exposed over IPC/API.
type View struct {
	ID               string  `json:"id"`
	AppID            string  `json:"app_id"`
	Name             string  `json:"name"`
	InstanceID       int     `json:"instance_id"`
	Namespace        string  `json:"namespace"`
	State            State   `json:"state"`
	PID              int     `json:"pid"`
	StartedAt        string  `json:"started_at,omitempty"`
	UptimeSeconds    int64   `json:"uptime_seconds"`
	Restarts         int     `json:"restarts"`
	UnstableRestarts int     `json:"unstable_restarts"`
	CPU              float64 `json:"cpu"`
	MemBytes         uint64  `json:"mem"`
	ExitCode         int     `json:"exit_code"`
	LastError        string  `json:"last_error,omitempty"`
}

func (r *Runtime) Snapshot() View {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v := View{
		ID:               r.ID,
		AppID:            r.AppID,
		Name:             r.Name,
		InstanceID:       r.InstanceID,
		Namespace:        r.Namespace,
		State:            r.State,
		PID:              r.PID,
		Restarts:         r.Restarts,
		UnstableRestarts: r.UnstableRestarts,
		CPU:              r.CPU,
		MemBytes:         r.MemBytes,
		ExitCode:         r.ExitCode,
		LastError:        r.LastError,
	}
	if !r.StartedAt.IsZero() {
		v.StartedAt = r.StartedAt.Format(time.RFC3339)
		if r.State == StateOnline {
			v.UptimeSeconds = int64(time.Since(r.StartedAt).Seconds())
		}
	}
	return v
}

// Lock helpers, only used by daemon package.
func (r *Runtime) Lock()       { r.mu.Lock() }
func (r *Runtime) Unlock()     { r.mu.Unlock() }
func (r *Runtime) RLock()      { r.mu.RLock() }
func (r *Runtime) RUnlock()    { r.mu.RUnlock() }

func (r *Runtime) Cmd() *exec.Cmd     { return r.cmd }
func (r *Runtime) SetCmd(c *exec.Cmd) { r.cmd = c }

func (r *Runtime) StopCh() chan struct{}     { return r.stopCh }
func (r *Runtime) SetStopCh(c chan struct{}) { r.stopCh = c }

func (r *Runtime) IsManualStop() bool   { return r.manualStop }
func (r *Runtime) SetManualStop(v bool) { r.manualStop = v }
