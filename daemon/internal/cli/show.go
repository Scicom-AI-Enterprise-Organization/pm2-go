package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/huseinzol05/pm2-go/daemon/internal/ipc"
)

var showCmd = &Command{
	Name:        "show",
	Aliases:     []string{"describe", "info"},
	Summary:     "Show detailed info for a process",
	NeedsDaemon: true,
	Run: func(ctx context.Context, c *ipc.Client, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("usage: pm2-go show <name>")
		}
		spec, procs, err := c.Describe(ctx, args[0])
		if err != nil {
			return err
		}
		fmt.Printf("name:        %s\n", spec.Name)
		fmt.Printf("script:      %s\n", spec.Script)
		if spec.Interpreter != "" {
			fmt.Printf("interpreter: %s\n", spec.Interpreter)
		}
		if len(spec.Args) > 0 {
			fmt.Printf("args:        %v\n", spec.Args)
		}
		fmt.Printf("cwd:         %s\n", spec.Cwd)
		fmt.Printf("namespace:   %s\n", spec.Namespace)
		fmt.Printf("instances:   %d\n", spec.Instances)
		fmt.Printf("autorestart: %v\n", spec.Autorestart())
		fmt.Printf("max_restart: %d\n", spec.MaxRestarts)
		fmt.Printf("kill_timeout:%s\n", spec.KillTimeout)
		if spec.MaxMemoryRestart > 0 {
			fmt.Printf("max_mem:     %s\n", humanBytes(spec.MaxMemoryRestart))
		}
		if len(spec.Watch) > 0 {
			fmt.Printf("watch:       %v\n", spec.Watch)
		}
		fmt.Println()
		fmt.Println("Runtime instances:")
		printProcsTable(procs)

		if os.Getenv("PM2_GO_DEBUG") != "" {
			b, _ := json.MarshalIndent(spec, "", "  ")
			fmt.Println()
			fmt.Println(string(b))
		}
		return nil
	},
}
