package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/ipc"
	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

var listCmd = &Command{
	Name:        "list",
	Aliases:     []string{"ls", "ps", "status"},
	Summary:     "List all managed processes",
	NeedsDaemon: true,
	Run: func(ctx context.Context, c *ipc.Client, _ []string) error {
		procs, err := c.List(ctx)
		if err != nil {
			return err
		}
		printProcsTable(procs)
		return nil
	},
}

func printProcsTable(procs []process.View) {
	if len(procs) == 0 {
		fmt.Println("No processes. Use `pm2-go start <script>` to add one.")
		return
	}
	cols := []struct {
		name  string
		width int
		get   func(process.View) string
	}{
		{"id", 16, func(v process.View) string { return v.ID }},
		{"name", 20, func(v process.View) string { return v.Name }},
		{"ns", 10, func(v process.View) string { return v.Namespace }},
		{"i", 3, func(v process.View) string { return fmt.Sprint(v.InstanceID) }},
		{"pid", 7, func(v process.View) string {
			if v.PID == 0 {
				return "-"
			}
			return fmt.Sprint(v.PID)
		}},
		{"state", 10, func(v process.View) string { return string(v.State) }},
		{"uptime", 10, func(v process.View) string {
			if v.UptimeSeconds == 0 {
				return "-"
			}
			return humanDuration(time.Duration(v.UptimeSeconds) * time.Second)
		}},
		{"↻", 4, func(v process.View) string { return fmt.Sprint(v.Restarts) }},
		{"cpu%", 6, func(v process.View) string { return fmt.Sprintf("%.1f", v.CPU) }},
		{"mem", 9, func(v process.View) string { return humanBytes(v.MemBytes) }},
	}

	// header
	var hdr strings.Builder
	for _, c := range cols {
		hdr.WriteString(fmt.Sprintf("%-*s ", c.width, c.name))
	}
	fmt.Println(strings.TrimRight(hdr.String(), " "))
	fmt.Println(strings.Repeat("─", hdr.Len()-1))

	for _, p := range procs {
		var b strings.Builder
		for _, col := range cols {
			val := col.get(p)
			if len(val) > col.width {
				val = val[:col.width]
			}
			b.WriteString(fmt.Sprintf("%-*s ", col.width, val))
		}
		fmt.Println(strings.TrimRight(b.String(), " "))
	}
}

func humanDuration(d time.Duration) string {
	if d < time.Second {
		return d.String()
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func humanBytes(n uint64) string {
	const k = 1024
	if n == 0 {
		return "-"
	}
	switch {
	case n < k:
		return fmt.Sprintf("%dB", n)
	case n < k*k:
		return fmt.Sprintf("%.1fK", float64(n)/k)
	case n < k*k*k:
		return fmt.Sprintf("%.1fM", float64(n)/(k*k))
	default:
		return fmt.Sprintf("%.2fG", float64(n)/(k*k*k))
	}
}
