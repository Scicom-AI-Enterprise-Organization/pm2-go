package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/config"
	"github.com/huseinzol05/pm2-go/daemon/internal/ipc"
	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

type kvFlag map[string]string

func (k kvFlag) String() string { return "" }
func (k kvFlag) Set(s string) error {
	eq := strings.IndexByte(s, '=')
	if eq <= 0 {
		return errors.New("expected KEY=VALUE")
	}
	k[s[:eq]] = s[eq+1:]
	return nil
}

type sliceFlag []string

func (s *sliceFlag) String() string { return strings.Join(*s, ",") }
func (s *sliceFlag) Set(v string)   error { *s = append(*s, v); return nil }

var startCmd = &Command{
	Name:        "start",
	Summary:     "Start a script, executable, or ecosystem file",
	NeedsDaemon: true,
	Run: func(ctx context.Context, c *ipc.Client, args []string) error {
		fs := newFlagSet("start")
		name := fs.String("name", "", "name for the process (defaults to script basename)")
		instances := fs.Int("instances", 1, "number of instances to run")
		instancesShort := fs.Int("i", 0, "alias for --instances")
		interpreter := fs.String("interpreter", "", "interpreter to run the script with (e.g. node, python3)")
		cwd := fs.String("cwd", "", "working directory")
		env := kvFlag{}
		fs.Var(env, "env", "set environment variable KEY=VALUE (repeatable)")
		var envFiles sliceFlag
		fs.Var(&envFiles, "env-file", "path to .env file (repeatable)")
		namespace := fs.String("namespace", "default", "namespace label")
		noAutorestart := fs.Bool("no-autorestart", false, "disable automatic restart on exit")
		maxRestarts := fs.Int("max-restarts", 16, "max restarts before marking errored (0 = unlimited)")
		maxMemBytes := fs.Uint64("max-memory-restart", 0, "restart if RSS exceeds N bytes")
		killTimeoutMS := fs.Int("kill-timeout", 1600, "milliseconds to wait after SIGTERM before SIGKILL")
		var watch sliceFlag
		fs.Var(&watch, "watch", "watch path for changes and restart (repeatable)")
		var ignore sliceFlag
		fs.Var(&ignore, "ignore-watch", "glob to ignore in --watch (repeatable)")
		var scriptArgs sliceFlag
		fs.Var(&scriptArgs, "arg", "argument to pass to the script (repeatable)")
		expBackoff := fs.Bool("exp-backoff", false, "exponential backoff on unstable restarts")
		shell := fs.Bool("shell", false, "run the command through /bin/sh -c (pipes, redirects, expansions)")
		if err := fs.Parse(reorderArgs(args, "no-autorestart", "exp-backoff", "shell")); err != nil {
			return err
		}
		positional := fs.Args()
		if len(positional) == 0 {
			fs.Usage()
			return errors.New("missing script, command, or ecosystem file")
		}

		// --shell: join all positionals and wrap them in /bin/sh -c "..."
		// pm2-go start --shell "cd /srv && node app.js | tee log"
		if *shell {
			cmd := strings.Join(positional, " ")
			displayName := deriveShellName(*name, cmd)
			if *instancesShort > 0 {
				*instances = *instancesShort
			}
			spec := &process.Spec{
				Name:                displayName,
				Script:              "/bin/sh",
				Args:                []string{"-c", cmd},
				Cwd:                 *cwd,
				Env:                 env,
				EnvFiles:            envFiles,
				Instances:           *instances,
				Namespace:           *namespace,
				AutorestartDisabled: *noAutorestart,
				MaxRestarts:         *maxRestarts,
				MaxMemoryRestart:    *maxMemBytes,
				KillTimeout:         time.Duration(*killTimeoutMS) * time.Millisecond,
				Watch:               watch,
				IgnoreWatch:         ignore,
				ExpBackoffRestart:   *expBackoff,
			}
			spec.ID = spec.Name
			if err := c.StartSpec(ctx, spec); err != nil {
				return err
			}
			fmt.Printf("[pm2-go] started %s\n", spec.Name)
			return nil
		}

		target := positional[0]
		extraArgs := positional[1:]

		// Auto-split: `pm2-go start "node app.js --port 3000"` — if the single
		// positional contains whitespace AND isn't an existing file, parse it
		// shell-style into [script, ...args]. This is the natural form for
		// long command lines pasted as a quoted string.
		if len(positional) == 1 && containsSpace(target) && !fileExists(target) && !isEcosystem(target) {
			toks, err := shellSplit(target)
			if err != nil {
				return err
			}
			if len(toks) > 0 {
				target = toks[0]
				extraArgs = append(toks[1:], extraArgs...)
			}
		}
		scriptArgs = append(scriptArgs, extraArgs...)

		if isEcosystem(target) {
			specs, err := config.LoadEcosystem(target)
			if err != nil {
				return err
			}
			for _, sp := range specs {
				if err := c.StartSpec(ctx, sp); err != nil {
					return fmt.Errorf("%s: %w", sp.Name, err)
				}
				fmt.Printf("[pm2-go] started %s\n", sp.Name)
			}
			return nil
		}

		if *instancesShort > 0 {
			*instances = *instancesShort
		}
		spec := &process.Spec{
			Name:                deriveName(*name, target),
			Script:              absScript(target),
			Args:                scriptArgs,
			Interpreter:         *interpreter,
			Cwd:                 *cwd,
			Env:                 env,
			EnvFiles:            envFiles,
			Instances:           *instances,
			Namespace:           *namespace,
			AutorestartDisabled: *noAutorestart,
			MaxRestarts:         *maxRestarts,
			MaxMemoryRestart:    *maxMemBytes,
			KillTimeout:         time.Duration(*killTimeoutMS) * time.Millisecond,
			Watch:               watch,
			IgnoreWatch:         ignore,
			ExpBackoffRestart:   *expBackoff,
		}
		spec.ID = spec.Name
		if err := c.StartSpec(ctx, spec); err != nil {
			return err
		}
		fmt.Printf("[pm2-go] started %s\n", spec.Name)
		return nil
	},
}

func isEcosystem(target string) bool {
	t := strings.ToLower(target)
	return strings.HasSuffix(t, ".json") ||
		strings.HasSuffix(t, ".yaml") ||
		strings.HasSuffix(t, ".yml")
}

func deriveName(name, target string) string {
	if name != "" {
		return name
	}
	base := filepath.Base(target)
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	return sanitizeName(base)
}

// deriveShellName picks a default name for `--shell` invocations. It uses the
// first whitespace-delimited token of the command line, since the full command
// is too long to be a name.
func deriveShellName(name, cmd string) string {
	if name != "" {
		return name
	}
	first := strings.TrimSpace(cmd)
	if first == "" {
		return "app"
	}
	if i := strings.IndexAny(first, " \t"); i > 0 {
		first = first[:i]
	}
	return sanitizeName(filepath.Base(first))
}

func sanitizeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	out := b.String()
	if out == "" {
		out = "app"
	}
	return out
}

func absScript(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

func containsSpace(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			return true
		}
	}
	return false
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
