package cli

import (
	"context"
	"fmt"

	"github.com/huseinzol05/pm2-go/daemon/internal/ipc"
)

var helpCmd = &Command{
	Name:    "help",
	Summary: "Show help for a command",
	Run: func(_ context.Context, _ *ipc.Client, args []string) error {
		if len(args) == 0 {
			printRootHelp()
			return nil
		}
		c := lookup(args[0])
		if c == nil {
			return fmt.Errorf("unknown command %q", args[0])
		}
		fmt.Printf("%s — %s\n", c.Name, c.Summary)
		if len(c.Aliases) > 0 {
			fmt.Printf("Aliases: %s\n", c.Aliases)
		}
		return nil
	},
}

var versionCmd = &Command{
	Name:    "version",
	Aliases: []string{"-v", "--version"},
	Summary: "Print pm2-go version",
	Run: func(_ context.Context, _ *ipc.Client, _ []string) error {
		fmt.Println(Version)
		return nil
	},
}
