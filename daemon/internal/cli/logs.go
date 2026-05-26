package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/ipc"
	"github.com/huseinzol05/pm2-go/daemon/internal/logs"
)

var logsCmd = &Command{
	Name:        "logs",
	Summary:     "Tail logs for one or all processes",
	NeedsDaemon: true,
	Run: func(ctx context.Context, c *ipc.Client, args []string) error {
		fs := newFlagSet("logs")
		lines := fs.Int("lines", 20, "lines of recent history to show")
		stream := fs.String("stream", "", "only 'out' or 'err'")
		follow := fs.Bool("follow", true, "follow new lines after history")
		noStream := fs.Bool("nostream", false, "print only history, don't follow")
		if err := fs.Parse(reorderArgs(args)); err != nil {
			return err
		}
		if *noStream {
			*follow = false
		}
		positional := fs.Args()
		if len(positional) == 0 {
			return fmt.Errorf("usage: pm2-go logs <name>")
		}
		name := positional[0]
		// history
		streams, err := c.Tail(ctx, name, *stream, *lines)
		if err != nil {
			return err
		}
		printTail(streams)
		if !*follow {
			return nil
		}
		body, err := c.StreamLogs(ctx, name)
		if err != nil {
			return err
		}
		defer body.Close()
		sc := bufio.NewScanner(body)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			var l logs.Line
			if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
				continue
			}
			ts := time.UnixMilli(l.Time).Format("15:04:05")
			color := "37" // gray-out for stdout
			if l.Stream == "err" {
				color = "31" // red for stderr
			}
			fmt.Printf("\x1b[2m%s\x1b[0m \x1b[%sm[%s/%s]\x1b[0m %s", ts, color, l.Runtime, l.Stream, l.Text)
			if !strings.HasSuffix(l.Text, "\n") {
				fmt.Println()
			}
		}
		return nil
	},
}

func printTail(streams map[string][]string) {
	if len(streams) == 0 {
		fmt.Fprintln(os.Stderr, "No log history yet.")
		return
	}
	for key, lines := range streams {
		if len(lines) == 0 {
			continue
		}
		fmt.Printf("─── %s ───\n", key)
		for _, l := range lines {
			fmt.Println(l)
		}
	}
}
