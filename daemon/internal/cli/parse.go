package cli

import "strings"

// reorderArgs separates positional arguments from flag arguments so that
// `pm2-go start ./script.sh --name foo` parses the same as
// `pm2-go start --name foo ./script.sh`.
//
// Go's stdlib flag package stops at the first non-flag token; this helper
// moves flags to the front. A bare `--` terminator marks the rest as positional.
//
// Caveat: for flags that take a value as a separate token (e.g. `--name foo`),
// we need to know whether the flag is boolean or not. Since we use mixed-style
// flagsets, the rule is: a token that looks like `--key=value` is one piece;
// `--key value` is two pieces and we always treat the following token as the
// value unless it itself starts with `-`. Boolean flags must be written as
// `--bool` or `--bool=true`.
func reorderArgs(args []string) []string {
	var flags, positional []string
	terminator := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if terminator {
			positional = append(positional, a)
			continue
		}
		if a == "--" {
			terminator = true
			positional = append(positional, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") && len(a) > 1 {
			flags = append(flags, a)
			// inline value with = -> done
			if strings.Contains(a, "=") {
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
