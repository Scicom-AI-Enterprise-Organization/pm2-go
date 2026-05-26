// Package metrics samples per-process CPU/mem from /proc. Linux only.
// Cross-platform support can be added later via gopsutil.
package metrics

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Sample struct {
	Time     time.Time
	CPU      float64 // percent of one core
	MemBytes uint64  // RSS
}

// Collector samples one PID at fixed intervals and keeps a rolling window.
type Collector struct {
	mu       sync.RWMutex
	pid      int
	window   []Sample
	max      int
	lastCPU  uint64 // jiffies
	lastTime time.Time
}

func New(pid int, windowSize int) *Collector {
	return &Collector{pid: pid, max: windowSize}
}

// Tick performs one sample. Safe to call concurrently with Latest/Window.
func (c *Collector) Tick() {
	s, err := readProc(c.pid)
	if err != nil {
		return
	}
	now := time.Now()
	var cpu float64
	c.mu.Lock()
	if !c.lastTime.IsZero() {
		dt := now.Sub(c.lastTime).Seconds()
		if dt > 0 {
			djiffies := float64(s.cpuJiffies - c.lastCPU)
			cpu = (djiffies / hertz()) / dt * 100.0
		}
	}
	c.lastCPU = s.cpuJiffies
	c.lastTime = now
	c.window = append(c.window, Sample{Time: now, CPU: cpu, MemBytes: s.rssBytes})
	if len(c.window) > c.max {
		c.window = c.window[len(c.window)-c.max:]
	}
	c.mu.Unlock()
}

func (c *Collector) Latest() Sample {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.window) == 0 {
		return Sample{}
	}
	return c.window[len(c.window)-1]
}

func (c *Collector) Window() []Sample {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Sample, len(c.window))
	copy(out, c.window)
	return out
}

type procStat struct {
	cpuJiffies uint64
	rssBytes   uint64
}

func readProc(pid int) (procStat, error) {
	var s procStat
	if runtime.GOOS != "linux" {
		return s, fmt.Errorf("metrics: only linux is supported")
	}
	statBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return s, err
	}
	stat := string(statBytes)
	// fields after the comm field can contain spaces in (comm); rest is space-delimited
	rparen := strings.LastIndex(stat, ")")
	if rparen < 0 {
		return s, fmt.Errorf("malformed stat")
	}
	rest := strings.Fields(stat[rparen+2:])
	// rest[11] = utime, rest[12] = stime  (0-indexed within rest; fields 14 and 15 in the original)
	if len(rest) < 22 {
		return s, fmt.Errorf("short stat: %d fields", len(rest))
	}
	utime, _ := strconv.ParseUint(rest[11], 10, 64)
	stime, _ := strconv.ParseUint(rest[12], 10, 64)
	s.cpuJiffies = utime + stime

	statusBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err == nil {
		for _, line := range strings.Split(string(statusBytes), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kb, _ := strconv.ParseUint(fields[1], 10, 64)
					s.rssBytes = kb * 1024
				}
				break
			}
		}
	}
	return s, nil
}

var hertzVal float64

func hertz() float64 {
	if hertzVal != 0 {
		return hertzVal
	}
	// USER_HZ on most Linux systems is 100. Reading SC_CLK_TCK via cgo would be
	// more correct but we avoid cgo here.
	hertzVal = 100.0
	return hertzVal
}
