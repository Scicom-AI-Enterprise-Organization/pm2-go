// Package process defines the data model for a managed application.
package process

import (
	"encoding/json"
	"fmt"
	"time"
)

type ExecMode string

const (
	ExecFork    ExecMode = "fork"
	ExecCluster ExecMode = "cluster"
)

// Spec is the desired configuration for a managed app. Multiple Runtimes
// (one per instance) are created from a single Spec.
type Spec struct {
	ID   string `json:"id"`   // stable id, derived from name
	Name string `json:"name"` // user-facing name

	Script      string            `json:"script"`                // path or executable
	Args        []string          `json:"args,omitempty"`        // script args
	Interpreter string            `json:"interpreter,omitempty"` // "node", "python3", "" for direct exec
	InterpreterArgs []string      `json:"interpreter_args,omitempty"`
	Cwd         string            `json:"cwd,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	EnvFiles    []string          `json:"env_files,omitempty"`

	Instances int      `json:"instances,omitempty"` // 0 or 1 = single; N = cluster
	ExecMode  ExecMode `json:"exec_mode,omitempty"` // fork (default) or cluster
	Namespace string   `json:"namespace,omitempty"` // for filtering

	AutorestartDisabled bool        `json:"autorestart_disabled,omitempty"` // default false = autorestart on
	MaxRestarts       int           `json:"max_restarts,omitempty"`     // 0 = unlimited
	MinUptime         time.Duration `json:"min_uptime,omitempty"`       // below = "unstable"
	RestartDelay      time.Duration `json:"restart_delay,omitempty"`
	ExpBackoffRestart bool          `json:"exp_backoff_restart,omitempty"`
	KillTimeout       time.Duration `json:"kill_timeout,omitempty"`     // SIGTERM grace
	MaxMemoryRestart  uint64        `json:"max_memory_restart,omitempty"` // bytes
	StopExitCodes     []int         `json:"stop_exit_codes,omitempty"`  // don't restart if exit ∈ this

	Watch       []string `json:"watch,omitempty"`        // dirs/files (relative to Cwd)
	IgnoreWatch []string `json:"ignore_watch,omitempty"` // glob patterns

	OutPath        string `json:"out_path,omitempty"`
	ErrPath        string `json:"err_path,omitempty"`
	MergeLogs      bool   `json:"merge_logs,omitempty"`
	LogDateFormat  string `json:"log_date_format,omitempty"`
	LogMaxSizeMB   int    `json:"log_max_size_mb,omitempty"` // 0 = no rotation
	LogMaxBackups  int    `json:"log_max_backups,omitempty"`
	LogMaxAgeDays  int    `json:"log_max_age_days,omitempty"`
	LogCompress    bool   `json:"log_compress,omitempty"`

	CronRestart string `json:"cron_restart,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Defaults fills in unset fields with sensible defaults.
func (s *Spec) Defaults() {
	if s.Instances <= 0 {
		s.Instances = 1
	}
	if s.ExecMode == "" {
		s.ExecMode = ExecFork
	}
	if s.Namespace == "" {
		s.Namespace = "default"
	}
	if s.KillTimeout == 0 {
		s.KillTimeout = 1600 * time.Millisecond
	}
	if s.MinUptime == 0 {
		s.MinUptime = 1 * time.Second
	}
	if s.RestartDelay == 0 {
		s.RestartDelay = 0
	}
	if s.MaxRestarts == 0 {
		s.MaxRestarts = 16
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	s.UpdatedAt = time.Now()
}

// Autorestart reports whether the spec should be restarted on exit.
func (s *Spec) Autorestart() bool { return !s.AutorestartDisabled }

// Validate returns an error if the spec is missing required fields.
func (s *Spec) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("name is required")
	}
	if s.Script == "" {
		return fmt.Errorf("script is required")
	}
	return nil
}

// Clone returns a deep copy.
func (s *Spec) Clone() *Spec {
	b, _ := json.Marshal(s)
	var out Spec
	_ = json.Unmarshal(b, &out)
	return &out
}
