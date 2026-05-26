// Package cli is the user-facing command surface. Each verb maps to a handler
// that talks to the daemon over the IPC socket (auto-spawning it if needed).
package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/daemon"
	"github.com/huseinzol05/pm2-go/daemon/internal/ipc"
)

// Version is set at build time via -ldflags.
var Version = "dev"

type Command struct {
	Name    string
	Aliases []string
	Summary string
	Run     func(ctx context.Context, c *ipc.Client, args []string) error
	NeedsDaemon bool // auto-spawn the daemon before running
}

var commands []*Command

func register(cmds ...*Command) { commands = append(commands, cmds...) }

func init() {
	register(
		startCmd, stopCmd, restartCmd, reloadCmd, deleteCmd,
		listCmd, showCmd, logsCmd,
		saveCmd, resurrectCmd, dumpCmd,
		monitCmd, webCmd, startupCmd,
		pingCmd, killCmd, versionCmd, helpCmd,
	)
}

// Main is the program entrypoint after the daemon subcommand carve-out.
func Main() int {
	if len(os.Args) < 2 {
		printRootHelp()
		return 0
	}
	name := os.Args[1]
	args := os.Args[2:]
	cmd := lookup(name)
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "pm2-go: unknown command %q\n", name)
		printRootHelp()
		return 2
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := ipc.NewClient()
	if cmd.NeedsDaemon {
		if err := daemon.EnsureRunning(ctx, 5*time.Second); err != nil {
			fmt.Fprintf(os.Stderr, "pm2-go: %v\n", err)
			return 1
		}
	}
	if err := cmd.Run(ctx, client, args); err != nil {
		fmt.Fprintf(os.Stderr, "pm2-go: %v\n", err)
		return 1
	}
	return 0
}

func lookup(name string) *Command {
	for _, c := range commands {
		if c.Name == name {
			return c
		}
		for _, a := range c.Aliases {
			if a == name {
				return c
			}
		}
	}
	return nil
}

func printRootHelp() {
	fmt.Fprintln(os.Stderr, "pm2-go — process manager")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "USAGE")
	fmt.Fprintln(os.Stderr, "  pm2-go <command> [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "COMMANDS")
	rows := make([][2]string, 0, len(commands))
	for _, c := range commands {
		aliases := ""
		if len(c.Aliases) > 0 {
			aliases = " (" + strings.Join(c.Aliases, ", ") + ")"
		}
		rows = append(rows, [2]string{c.Name + aliases, c.Summary})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
	for _, r := range rows {
		fmt.Fprintf(os.Stderr, "  %-22s %s\n", r[0], r[1])
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Run 'pm2-go help <command>' for command-specific help.")
}

// newFlagSet builds a per-command flagset that prints usage prefixed with the
// command name.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet("pm2-go "+name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}
