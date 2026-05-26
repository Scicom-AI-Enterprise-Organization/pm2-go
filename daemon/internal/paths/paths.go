// Package paths centralizes all filesystem locations used by pm2-go.
package paths

import (
	"os"
	"path/filepath"
)

const dirName = ".pm2-go"

// Home returns the pm2-go state directory (~/.pm2-go), creating it on demand.
func Home() string {
	if v := os.Getenv("PM2_GO_HOME"); v != "" {
		_ = os.MkdirAll(v, 0o755)
		return v
	}
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, dirName)
	_ = os.MkdirAll(p, 0o755)
	return p
}

func sub(name string) string {
	p := filepath.Join(Home(), name)
	_ = os.MkdirAll(p, 0o755)
	return p
}

func LogsDir() string { return sub("logs") }
func PidsDir() string { return sub("pids") }

func RPCSocket() string  { return filepath.Join(Home(), "rpc.sock") }
func DumpFile() string   { return filepath.Join(Home(), "dump.json") }
func DaemonPid() string  { return filepath.Join(Home(), "daemon.pid") }
func DaemonLog() string  { return filepath.Join(Home(), "daemon.log") }
func APIToken() string   { return filepath.Join(Home(), "api-token") }
func APIPortFile() string { return filepath.Join(Home(), "api-port") }
