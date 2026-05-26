// Package logs captures a process's stdout/stderr to disk while fanning out to
// any live subscribers (CLI `logs` followers, web UI WS clients).
package logs

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/huseinzol05/pm2-go/daemon/internal/paths"
	"gopkg.in/natefinch/lumberjack.v2"
)

// RotateOpts configures size/age based log rotation. Zero values disable
// rotation and fall back to a plain append-only file.
type RotateOpts struct {
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

// Stream represents a single log channel (stdout or stderr) for one runtime.
// Writes go to disk and to any subscribed in-memory channels.
type Stream struct {
	mu          sync.Mutex
	w           io.WriteCloser
	subscribers map[chan Line]struct{}
}

type Line struct {
	Runtime string `json:"runtime"`
	Stream  string `json:"stream"` // "out" or "err"
	Time    int64  `json:"time"`   // unix millis
	Text    string `json:"text"`
}

// FilePath returns the on-disk file path for a runtime's out or err stream.
func FilePath(runtimeID, stream string) string {
	return filepath.Join(paths.LogsDir(), runtimeID+"-"+stream+".log")
}

// Open creates a Stream that appends to <runtimeID>-<stream>.log and pipes
// any reader content into it. If opts.MaxSizeMB > 0 rotation is enabled.
func Open(runtimeID, stream string, opts RotateOpts) (*Stream, error) {
	p := FilePath(runtimeID, stream)
	var w io.WriteCloser
	if opts.MaxSizeMB > 0 {
		w = &lumberjack.Logger{
			Filename:   p,
			MaxSize:    opts.MaxSizeMB,
			MaxBackups: opts.MaxBackups,
			MaxAge:     opts.MaxAgeDays,
			Compress:   opts.Compress,
			LocalTime:  true,
		}
	} else {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, err
		}
		w = f
	}
	return &Stream{w: w, subscribers: map[chan Line]struct{}{}}, nil
}

// Attach starts a goroutine that copies r into the stream's file and broadcasts
// each line to subscribers. Closes the underlying file when r returns EOF.
func (s *Stream) Attach(runtimeID, name string, r io.Reader) {
	go func() {
		defer s.Close()
		br := bufio.NewReader(r)
		for {
			line, err := br.ReadString('\n')
			if len(line) > 0 {
				s.mu.Lock()
				_, _ = s.w.Write([]byte(line))
				if len(s.subscribers) > 0 {
					l := Line{Runtime: runtimeID, Stream: name, Time: nowMillis(), Text: line}
					for ch := range s.subscribers {
						select {
						case ch <- l:
						default: // drop if subscriber is slow
						}
					}
				}
				s.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
}

func (s *Stream) Subscribe(buf int) chan Line {
	ch := make(chan Line, buf)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

func (s *Stream) Unsubscribe(ch chan Line) {
	s.mu.Lock()
	delete(s.subscribers, ch)
	s.mu.Unlock()
	close(ch)
}

func (s *Stream) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.w != nil {
		_ = s.w.Close()
		s.w = nil
	}
}

// Tail reads the last n lines of a runtime's log file. Returns empty if missing.
func Tail(runtimeID, stream string, n int) ([]string, error) {
	p := FilePath(runtimeID, stream)
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	return tailReader(f, n)
}

func tailReader(f *os.File, n int) ([]string, error) {
	const chunk = 4096
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	var buf []byte
	for off := size; off > 0 && countLines(buf) <= n; {
		read := int64(chunk)
		if off-read < 0 {
			read = off
		}
		off -= read
		tmp := make([]byte, read)
		if _, err := f.ReadAt(tmp, off); err != nil && err != io.EOF {
			return nil, err
		}
		buf = append(tmp, buf...)
	}
	lines := splitLines(buf)
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}

func countLines(b []byte) int {
	c := 0
	for _, v := range b {
		if v == '\n' {
			c++
		}
	}
	return c
}

func splitLines(b []byte) []string {
	var out []string
	start := 0
	for i, v := range b {
		if v == '\n' {
			out = append(out, string(b[start:i]))
			start = i + 1
		}
	}
	if start < len(b) {
		out = append(out, string(b[start:]))
	}
	return out
}
