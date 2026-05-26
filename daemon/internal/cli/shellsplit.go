package cli

import (
	"fmt"
	"strings"
)

// shellSplit tokenizes a string the way a POSIX shell would split a simple
// command line: handles double-quoted, single-quoted, and backslash-escaped
// tokens. It does NOT interpret pipes, redirects, expansions, or operators —
// use the `--shell` flag for that.
//
//   shellSplit(`node app.js --name "my app" --flag`)
//   → []string{"node", "app.js", "--name", "my app", "--flag"}
func shellSplit(s string) ([]string, error) {
	var out []string
	var cur strings.Builder
	inSingle, inDouble := false, false
	flushed := true // whether we just emitted a token

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && !inSingle && i+1 < len(s):
			cur.WriteByte(s[i+1])
			i++
			flushed = false
		case c == '\'' && !inDouble:
			inSingle = !inSingle
			flushed = false
		case c == '"' && !inSingle:
			inDouble = !inDouble
			flushed = false
		case (c == ' ' || c == '\t') && !inSingle && !inDouble:
			if !flushed {
				out = append(out, cur.String())
				cur.Reset()
				flushed = true
			}
		default:
			cur.WriteByte(c)
			flushed = false
		}
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote in %q", s)
	}
	if !flushed {
		out = append(out, cur.String())
	}
	return out, nil
}
