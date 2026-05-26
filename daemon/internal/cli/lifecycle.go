package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/huseinzol05/pm2-go/daemon/internal/ipc"
)

const allKeyword = "all"

func nameAction(verb string, do func(*ipc.Client, context.Context, string) error, allFn func(*ipc.Client, context.Context) error) func(context.Context, *ipc.Client, []string) error {
	return func(ctx context.Context, c *ipc.Client, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("usage: pm2-go %s <name|all>", verb)
		}
		for _, name := range args {
			if name == allKeyword {
				if allFn == nil {
					return errors.New(verb + " all not supported")
				}
				if err := allFn(c, ctx); err != nil {
					return err
				}
				fmt.Printf("[pm2-go] %s all\n", verb)
				continue
			}
			if err := do(c, ctx, name); err != nil {
				return fmt.Errorf("%s %s: %w", verb, name, err)
			}
			fmt.Printf("[pm2-go] %s %s\n", verb, name)
		}
		return nil
	}
}

var stopCmd = &Command{
	Name:        "stop",
	Summary:     "Stop a process by name (or 'all')",
	NeedsDaemon: true,
	Run:         nameAction("stop", (*ipc.Client).Stop, (*ipc.Client).StopAll),
}

var restartCmd = &Command{
	Name:        "restart",
	Summary:     "Restart a process by name (or 'all')",
	NeedsDaemon: true,
	Run: nameAction("restart", (*ipc.Client).Restart, func(c *ipc.Client, ctx context.Context) error {
		if err := c.StopAll(ctx); err != nil {
			return err
		}
		return c.StartAll(ctx)
	}),
}

var reloadCmd = &Command{
	Name:        "reload",
	Summary:     "Graceful reload (SIGUSR2) — fallback to restart",
	NeedsDaemon: true,
	Run:         nameAction("reload", (*ipc.Client).Reload, nil),
}

var deleteCmd = &Command{
	Name:        "delete",
	Aliases:     []string{"del", "rm"},
	Summary:     "Stop and remove a process by name (or 'all')",
	NeedsDaemon: true,
	Run:         nameAction("delete", (*ipc.Client).Delete, (*ipc.Client).DeleteAll),
}
