package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/ipc"
	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

var monitCmd = &Command{
	Name:        "monit",
	Summary:     "Live TTY dashboard of all processes",
	NeedsDaemon: true,
	Run: func(ctx context.Context, c *ipc.Client, _ []string) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() { <-sigCh; cancel() }()

		// hide cursor + alt-screen on enter, restore on exit
		fmt.Print("\x1b[?25l\x1b[?1049h")
		defer fmt.Print("\x1b[?1049l\x1b[?25h")

		t := time.NewTicker(1 * time.Second)
		defer t.Stop()
		render := func() error {
			procs, err := c.List(ctx)
			if err != nil {
				return err
			}
			fmt.Print("\x1b[H\x1b[J")
			fmt.Print("\x1b[1mpm2-go monit\x1b[0m — press Ctrl-C to exit  ")
			fmt.Printf("\x1b[2m(%s)\x1b[0m\n\n", time.Now().Format("15:04:05"))
			renderMonit(procs)
			return nil
		}
		_ = render()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				if err := render(); err != nil {
					return err
				}
			}
		}
	},
}

func renderMonit(procs []process.View) {
	if len(procs) == 0 {
		fmt.Println("(no processes)")
		return
	}
	for _, p := range procs {
		label := fmt.Sprintf("%-20s", p.Name)
		state := colorState(p.State)
		cpuBar := bar(p.CPU/100.0, 20)
		fmt.Printf("%s %s  cpu %5.1f%% %s  mem %s  pid %d  ↻%d\n",
			label, state, p.CPU, cpuBar, humanBytes(p.MemBytes), p.PID, p.Restarts)
	}
}

func colorState(s process.State) string {
	switch s {
	case process.StateOnline:
		return "\x1b[32m●\x1b[0m online "
	case process.StateLaunching:
		return "\x1b[33m●\x1b[0m launch "
	case process.StateWaitRestart:
		return "\x1b[33m●\x1b[0m waitR  "
	case process.StateStopping:
		return "\x1b[33m●\x1b[0m stopng "
	case process.StateStopped:
		return "\x1b[37m●\x1b[0m stoppd "
	case process.StateErrored:
		return "\x1b[31m●\x1b[0m errord "
	}
	return string(s)
}

func bar(frac float64, width int) string {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac * float64(width))
	return "\x1b[36m" + strings.Repeat("█", filled) + "\x1b[2m" + strings.Repeat("░", width-filled) + "\x1b[0m"
}
