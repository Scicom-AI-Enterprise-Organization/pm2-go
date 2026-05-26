package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	"github.com/huseinzol05/pm2-go/daemon/internal/ipc"
	"github.com/huseinzol05/pm2-go/daemon/internal/paths"
)

var webCmd = &Command{
	Name:        "web",
	Summary:     "Configure the public web API (manages ~/.pm2-go/api-port + api-token)",
	NeedsDaemon: false,
	Run: func(_ context.Context, _ *ipc.Client, args []string) error {
		fs := newFlagSet("web")
		port := fs.Int("port", 9615, "port the daemon should expose the web API on")
		host := fs.String("host", "127.0.0.1", "bind address")
		regen := fs.Bool("regenerate-token", false, "create a fresh API token, invalidating the old one")
		showToken := fs.Bool("token", false, "print the current API token")
		if err := fs.Parse(reorderArgs(args, "regenerate-token", "token")); err != nil {
			return err
		}
		if *showToken {
			b, err := os.ReadFile(paths.APIToken())
			if err != nil {
				return err
			}
			fmt.Print(string(b))
			return nil
		}

		token, err := ensureToken(*regen)
		if err != nil {
			return err
		}
		addr := fmt.Sprintf("%s:%d", *host, *port)
		if err := os.WriteFile(paths.APIPortFile(), []byte(addr), 0o644); err != nil {
			return err
		}
		fmt.Printf("Web API endpoint: http://%s\n", addr)
		fmt.Printf("Auth header:      Authorization: Bearer %s\n", token)
		fmt.Println()
		fmt.Println("Restart the daemon to apply (pm2-go kill && pm2-go ping).")
		return nil
	},
}

func ensureToken(regen bool) (string, error) {
	if !regen {
		if b, err := os.ReadFile(paths.APIToken()); err == nil && len(b) > 0 {
			return string(b), nil
		}
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(buf)
	if err := os.WriteFile(paths.APIToken(), []byte(tok), 0o600); err != nil {
		return "", err
	}
	return tok, nil
}

// ReadWebConfig returns the configured address and token, or empty strings if
// the web API isn't configured.
func ReadWebConfig() (addr, token string) {
	if b, err := os.ReadFile(paths.APIPortFile()); err == nil {
		addr = string(b)
	}
	if b, err := os.ReadFile(paths.APIToken()); err == nil {
		token = string(b)
	}
	return
}

// portFromAddr returns the trailing :port number, or 0 if malformed.
func portFromAddr(addr string) int {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			n, err := strconv.Atoi(addr[i+1:])
			if err != nil {
				return 0
			}
			return n
		}
	}
	return 0
}

var _ = portFromAddr // reserved for future use
