package cli

import "strings"

// reorderArgs separates positional arguments from flag arguments so that
// `pm2-go start ./script.sh --name foo` parses the same as
// `pm2-go start --name foo ./script.sh`.
//
// Go's stdlib flag package stops at the first non-flag token; this helper
// moves flags to the front. A bare `--` terminator marks the rest as positional.
//
// `boolFlags` lists the names (without leading dashes) of boolean flags that
// don't consume the following token — without this list, `--shell "cmd"` would
// incorrectly treat "cmd" as the value of --shell.
func reorderArgs(args []string, boolFlags ...string) []string {
	bools := make(map[string]bool, len(boolFlags))
	for _, n := range boolFlags {
		bools[n] = true
	}
	var flags, positional []string
	terminator := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if terminator {
			positional = append(positional, a)
			continue
		}
		if a == "--" {
			// Keep the terminator in the positional list so downstream
			// flag.Parse still sees it and stops processing the same way.
			positional = append(positional, args[i:]...)
			break
		}
		if strings.HasPrefix(a, "-") && len(a) > 1 {
			flags = append(flags, a)
			// inline value with = -> done
			if strings.Contains(a, "=") {
				continue
			}
			name := strings.TrimLeft(a, "-")
			if bools[name] {
				continue
			}
			// otherwise look at the next token; if it doesn't start with '-' it's the value
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		positional = append(positional, a)
	}
	return append(flags, positional...)
}
