// pm2-go: process manager (PM2-compatible-ish) in a single Go binary.
//
// Modes:
//   pm2-go <command> [args...]   normal CLI; auto-spawns the daemon if missing
//   pm2-go __daemon              internal: runs the daemon in the foreground
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/huseinzol05/pm2-go/daemon/internal/api"
	"github.com/huseinzol05/pm2-go/daemon/internal/cli"
	"github.com/huseinzol05/pm2-go/daemon/internal/daemon"
	"github.com/huseinzol05/pm2-go/daemon/internal/ipc"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	cli.Version = Version
	if len(os.Args) >= 2 && os.Args[1] == "__daemon" {
		os.Exit(runDaemon())
	}
	os.Exit(cli.Main())
}

func runDaemon() int {
	ctx := context.Background()
	err := daemon.Run(ctx, func(ctx context.Context, sup *daemon.Supervisor) error {
		// IPC server (unix socket — used by the CLI). This is the primary
		// blocker; the daemon stays alive as long as it serves.
		ipcSrv := ipc.NewServer(sup)
		ipcErr := make(chan error, 1)
		go func() { ipcErr <- ipcSrv.Serve(ctx) }()

		// Web API server (TCP — used by the Next.js UI; only if configured).
		// We launch it but ignore a nil return (means "not configured").
		go func() {
			if err := api.Serve(ctx, sup); err != nil {
				fmt.Fprintf(os.Stderr, "web api: %v\n", err)
			}
		}()

		select {
		case e := <-ipcErr:
			return e
		case <-ctx.Done():
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
		return 1
	}
	return 0
}
