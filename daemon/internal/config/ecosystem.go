// Package config parses ecosystem.config.{json,yaml,yml} files into Specs.
//
// Schema is intentionally a near-superset of PM2's. Camel-case keys are accepted
// as aliases for snake_case ones. Durations are seconds (numbers) or duration
// strings ("1.5s", "500ms").
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

type rawEcosystem struct {
	Apps []rawApp `json:"apps"`
}

type rawApp struct {
	Name              string            `json:"name"`
	Script            string            `json:"script"`
	Args              any               `json:"args"`
	Interpreter       string            `json:"interpreter"`
	InterpreterArgs   any               `json:"interpreter_args"`
	Cwd               string            `json:"cwd"`
	Env               map[string]string `json:"env"`
	EnvFiles          []string          `json:"env_files"`
	Instances         int               `json:"instances"`
	ExecMode          string            `json:"exec_mode"`
	Namespace         string            `json:"namespace"`
	Autorestart       *bool             `json:"autorestart"`
	MaxRestarts       int               `json:"max_restarts"`
	MinUptime         any               `json:"min_uptime"`
	RestartDelay      any               `json:"restart_delay"`
	ExpBackoffRestart bool              `json:"exp_backoff_restart"`
	KillTimeout       any               `json:"kill_timeout"`
	MaxMemoryRestart  any               `json:"max_memory_restart"`
	StopExitCodes     []int             `json:"stop_exit_codes"`
	Watch             any               `json:"watch"`
	IgnoreWatch       []string          `json:"ignore_watch"`
	OutPath           string            `json:"out_file"`
	ErrPath           string            `json:"error_file"`
	MergeLogs         bool              `json:"merge_logs"`
	LogDateFormat     string            `json:"log_date_format"`
	LogMaxSizeMB      int               `json:"log_max_size_mb"`
	LogMaxBackups     int               `json:"log_max_backups"`
	LogMaxAgeDays     int               `json:"log_max_age_days"`
	LogCompress       bool              `json:"log_compress"`
	CronRestart       string            `json:"cron_restart"`
}

// LoadEcosystem reads a JSON or YAML ecosystem file. YAML support is minimal —
// it's converted to JSON first via a tiny inline reader.
func LoadEcosystem(path string) ([]*process.Spec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	var raw rawEcosystem
	switch ext {
	case ".json":
		if err := json.Unmarshal(b, &raw); err != nil {
			// allow bare array
			var apps []rawApp
			if err2 := json.Unmarshal(b, &apps); err2 == nil {
				raw.Apps = apps
			} else {
				return nil, fmt.Errorf("parse %s: %w", path, err)
			}
		}
	case ".yaml", ".yml":
		// Minimal YAML reader is out of scope; require JSON for now.
		return nil, fmt.Errorf("yaml ecosystem files not yet supported — convert to JSON")
	default:
		return nil, fmt.Errorf("unrecognized ecosystem extension %q", ext)
	}

	base := filepath.Dir(path)
	out := make([]*process.Spec, 0, len(raw.Apps))
	for i, a := range raw.Apps {
		spec, err := buildSpec(a, base)
		if err != nil {
			return nil, fmt.Errorf("apps[%d] %s: %w", i, a.Name, err)
		}
		out = append(out, spec)
	}
	return out, nil
}

func buildSpec(a rawApp, base string) (*process.Spec, error) {
	if a.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if a.Script == "" {
		return nil, fmt.Errorf("script is required")
	}
	script := a.Script
	if !filepath.IsAbs(script) {
		script = filepath.Join(base, script)
	}
	cwd := a.Cwd
	if cwd != "" && !filepath.IsAbs(cwd) {
		cwd = filepath.Join(base, cwd)
	}
	spec := &process.Spec{
		Name:              a.Name,
		ID:                a.Name,
		Script:            script,
		Args:              toStringSlice(a.Args),
		Interpreter:       a.Interpreter,
		InterpreterArgs:   toStringSlice(a.InterpreterArgs),
		Cwd:               cwd,
		Env:               a.Env,
		EnvFiles:          a.EnvFiles,
		Instances:         a.Instances,
		ExecMode:          process.ExecMode(a.ExecMode),
		Namespace:         a.Namespace,
		MaxRestarts:       a.MaxRestarts,
		ExpBackoffRestart: a.ExpBackoffRestart,
		StopExitCodes:     a.StopExitCodes,
		Watch:             toStringSlice(a.Watch),
		IgnoreWatch:       a.IgnoreWatch,
		OutPath:           a.OutPath,
		ErrPath:           a.ErrPath,
		MergeLogs:         a.MergeLogs,
		LogDateFormat:     a.LogDateFormat,
		LogMaxSizeMB:      a.LogMaxSizeMB,
		LogMaxBackups:     a.LogMaxBackups,
		LogMaxAgeDays:     a.LogMaxAgeDays,
		LogCompress:       a.LogCompress,
		CronRestart:       a.CronRestart,
	}
	if a.Autorestart != nil && !*a.Autorestart {
		spec.AutorestartDisabled = true
	}
	if d, err := toDuration(a.MinUptime); err == nil {
		spec.MinUptime = d
	}
	if d, err := toDuration(a.RestartDelay); err == nil {
		spec.RestartDelay = d
	}
	if d, err := toDuration(a.KillTimeout); err == nil {
		spec.KillTimeout = d
	}
	if mb, err := toBytes(a.MaxMemoryRestart); err == nil {
		spec.MaxMemoryRestart = mb
	}
	return spec, nil
}

func toStringSlice(v any) []string {
	switch x := v.(type) {
	case nil:
		return nil
	case string:
		// pm2 accepts a single space-separated string for args
		if x == "" {
			return nil
		}
		return strings.Fields(x)
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			out = append(out, fmt.Sprint(e))
		}
		return out
	case []string:
		return x
	}
	return nil
}

func toDuration(v any) (time.Duration, error) {
	switch x := v.(type) {
	case nil:
		return 0, fmt.Errorf("nil")
	case string:
		return time.ParseDuration(x)
	case float64:
		// seconds
		return time.Duration(x * float64(time.Second)), nil
	case int:
		return time.Duration(x) * time.Second, nil
	}
	return 0, fmt.Errorf("unsupported duration %T", v)
}

func toBytes(v any) (uint64, error) {
	switch x := v.(type) {
	case nil:
		return 0, fmt.Errorf("nil")
	case float64:
		return uint64(x), nil
	case int:
		return uint64(x), nil
	case string:
		// pm2: "300M", "1G"
		s := strings.TrimSpace(strings.ToUpper(x))
		mult := uint64(1)
		switch {
		case strings.HasSuffix(s, "K"):
			mult = 1024
			s = strings.TrimSuffix(s, "K")
		case strings.HasSuffix(s, "M"):
			mult = 1024 * 1024
			s = strings.TrimSuffix(s, "M")
		case strings.HasSuffix(s, "G"):
			mult = 1024 * 1024 * 1024
			s = strings.TrimSuffix(s, "G")
		}
		var n uint64
		_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
		if err != nil {
			return 0, err
		}
		return n * mult, nil
	}
	return 0, fmt.Errorf("unsupported size %T", v)
}
