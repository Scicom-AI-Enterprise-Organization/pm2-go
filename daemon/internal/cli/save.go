package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/huseinzol05/pm2-go/daemon/internal/ipc"
	"github.com/huseinzol05/pm2-go/daemon/internal/paths"
)

var saveCmd = &Command{
	Name:        "save",
	Summary:     "Persist the current process list to ~/.pm2-go/dump.json",
	NeedsDaemon: true,
	Run: func(ctx context.Context, c *ipc.Client, _ []string) error {
		if err := c.Save(ctx); err != nil {
			return err
		}
		fmt.Printf("[pm2-go] saved %s\n", paths.DumpFile())
		return nil
	},
}

var resurrectCmd = &Command{
	Name:        "resurrect",
	Summary:     "Start all processes from the saved dump (re-spawns daemon if needed)",
	NeedsDaemon: true,
	Run: func(ctx context.Context, c *ipc.Client, _ []string) error {
		// EnsureRunning already happened (NeedsDaemon=true). The daemon calls
		// Resurrect() once on boot. If the daemon was already running, the
		// dump.json is already loaded — so resurrect is a no-op aside from
		// confirming the state.
		procs, err := c.List(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("[pm2-go] %d process(es) live\n", len(procs))
		return nil
	},
}

var dumpCmd = &Command{
	Name:        "dump",
	Summary:     "Print the saved specs as JSON",
	NeedsDaemon: true,
	Run: func(ctx context.Context, c *ipc.Client, _ []string) error {
		specs, err := c.Specs(ctx)
		if err != nil {
			return err
		}
		b, _ := json.MarshalIndent(specs, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}
