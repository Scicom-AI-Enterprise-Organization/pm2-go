package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/huseinzol05/pm2-go/daemon/internal/daemon"
	"github.com/huseinzol05/pm2-go/daemon/internal/ipc"
	"github.com/huseinzol05/pm2-go/daemon/internal/paths"
)

var pingCmd = &Command{
	Name:    "ping",
	Summary: "Check whether the daemon is alive",
	Run: func(ctx context.Context, c *ipc.Client, _ []string) error {
		if err := c.Ping(ctx); err != nil {
			return fmt.Errorf("daemon not reachable: %w", err)
		}
		fmt.Println("pong")
		return nil
	},
}

var killCmd = &Command{
	Name:    "kill",
	Summary: "Shut down the daemon (stops all managed processes)",
	Run: func(ctx context.Context, c *ipc.Client, _ []string) error {
		if !daemon.SocketAlive() {
			fmt.Println("[pm2-go] daemon not running")
			return nil
		}
		// stop all first so processes have a chance to exit cleanly
		_ = c.StopAll(ctx)
		_ = c.Shutdown(ctx)
		fmt.Println("[pm2-go] shutdown requested")
		return nil
	},
}

// startupCmd prints a systemd unit (or installs it with --install) so the
// daemon resurrects on boot.
var startupCmd = &Command{
	Name:    "startup",
	Summary: "Generate a systemd unit to start pm2-go on boot",
	Run: func(_ context.Context, _ *ipc.Client, args []string) error {
		fs := newFlagSet("startup")
		platform := fs.String("platform", "systemd", "systemd | launchd | openrc (only systemd implemented)")
		install := fs.Bool("install", false, "write the unit to ~/.config/systemd/user/ and enable it")
		if err := fs.Parse(reorderArgs(args, "install")); err != nil {
			return err
		}
		if *platform != "systemd" {
			return fmt.Errorf("only systemd is supported currently")
		}
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		exe, _ = filepath.EvalSymlinks(exe)
		user := os.Getenv("USER")
		home := os.Getenv("HOME")
		unit := strings.NewReplacer(
			"{{user}}", user,
			"{{home}}", home,
			"{{exe}}", exe,
			"{{pm2_home}}", paths.Home(),
		).Replace(systemdUnit)
		if !*install {
			fmt.Print(unit)
			fmt.Fprintln(os.Stderr, "\n# To install:")
			fmt.Fprintln(os.Stderr, "#   pm2-go startup --install")
			fmt.Fprintln(os.Stderr, "#   systemctl --user enable --now pm2-go")
			return nil
		}
		unitDir := filepath.Join(home, ".config", "systemd", "user")
		if err := os.MkdirAll(unitDir, 0o755); err != nil {
			return err
		}
		path := filepath.Join(unitDir, "pm2-go.service")
		if err := os.WriteFile(path, []byte(unit), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Wrote %s\nRun: systemctl --user daemon-reload && systemctl --user enable --now pm2-go\n", path)
		return nil
	},
}

const systemdUnit = `[Unit]
Description=pm2-go process manager
After=network.target

[Service]
Type=forking
Environment=PM2_GO_HOME={{pm2_home}}
ExecStart={{exe}} __daemon
ExecStop={{exe}} kill
Restart=on-failure
RestartSec=2

[Install]
WantedBy=default.target
`
