// Package daemon contains the in-process supervisor that owns all child
// processes. It exposes lifecycle operations (Start/Stop/Restart/Delete/etc.)
// to the IPC layer.
package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/logs"
	"github.com/huseinzol05/pm2-go/daemon/internal/metrics"
	"github.com/huseinzol05/pm2-go/daemon/internal/process"
	"github.com/huseinzol05/pm2-go/daemon/internal/store"
)

const (
	metricsTickInterval = 1 * time.Second
	metricsWindow       = 300 // 5 min at 1Hz
	autosaveInterval    = 2 * time.Second
)

var ErrNotFound = errors.New("not found")

// Supervisor owns Specs and their Runtimes.
type Supervisor struct {
	mu       sync.RWMutex
	specs    map[string]*process.Spec      // by Spec.ID
	runtimes map[string]*process.Runtime   // by Runtime.ID
	streams  map[string]*streamPair        // by Runtime.ID
	metrics  map[string]*metrics.Collector // by Runtime.ID
	watchers map[string]*Watcher           // by Spec.ID
	store    *store.Store

	ctx       context.Context
	cancel    context.CancelFunc
	saveDirty chan struct{}

	logger *log.Logger
}

type streamPair struct {
	Out *logs.Stream
	Err *logs.Stream
}

func NewSupervisor() *Supervisor {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Supervisor{
		specs:     map[string]*process.Spec{},
		runtimes:  map[string]*process.Runtime{},
		streams:   map[string]*streamPair{},
		metrics:   map[string]*metrics.Collector{},
		watchers:  map[string]*Watcher{},
		store:     store.New(),
		ctx:       ctx,
		cancel:    cancel,
		saveDirty: make(chan struct{}, 1),
		logger:    log.New(os.Stderr, "[supervisor] ", log.LstdFlags),
	}
	go s.metricsLoop()
	go s.autosaveLoop()
	return s
}

// Resurrect loads dump.json and starts every persisted spec.
func (s *Supervisor) Resurrect() error {
	d, err := s.store.Load()
	if err != nil {
		return err
	}
	for _, spec := range d.Specs {
		spec.Defaults()
		s.mu.Lock()
		s.specs[spec.ID] = spec
		s.mu.Unlock()
		if err := s.startSpec(spec); err != nil {
			s.logger.Printf("resurrect %s: %v", spec.Name, err)
		}
		s.maybeStartWatcher(spec)
	}
	return nil
}

func (s *Supervisor) Shutdown() {
	s.cancel()
	s.mu.RLock()
	names := make([]string, 0, len(s.specs))
	for _, sp := range s.specs {
		names = append(names, sp.Name)
	}
	s.mu.RUnlock()
	for _, n := range names {
		_ = s.Stop(n)
	}
	s.requestSave()
	// final synchronous save
	s.save()
}

// ---- public API ----

// Add registers or replaces a Spec, then starts it. If a spec with the same
// Name exists, it is stopped, replaced, and (re)started.
func (s *Supervisor) Add(spec *process.Spec) error {
	spec.Defaults()
	if err := spec.Validate(); err != nil {
		return err
	}
	if spec.ID == "" {
		spec.ID = spec.Name
	}
	s.mu.Lock()
	existing, ok := s.specs[spec.ID]
	s.mu.Unlock()
	if ok {
		_ = s.Stop(existing.Name)
		s.removeRuntimes(existing.ID)
	}
	s.mu.Lock()
	s.specs[spec.ID] = spec
	s.mu.Unlock()
	if err := s.startSpec(spec); err != nil {
		return err
	}
	s.maybeStartWatcher(spec)
	s.requestSave()
	return nil
}

func (s *Supervisor) maybeStartWatcher(spec *process.Spec) {
	if len(spec.Watch) == 0 {
		return
	}
	s.mu.Lock()
	if old := s.watchers[spec.ID]; old != nil {
		old.Close()
	}
	s.mu.Unlock()
	name := spec.Name
	w, err := NewWatcher(spec, func() {
		s.logger.Printf("watch fired for %s — restarting", name)
		_ = s.Restart(name)
	})
	if err != nil {
		s.logger.Printf("watch %s: %v", spec.Name, err)
		return
	}
	s.mu.Lock()
	s.watchers[spec.ID] = w
	s.mu.Unlock()
}

func (s *Supervisor) Start(name string) error {
	spec := s.findSpec(name)
	if spec == nil {
		return fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return s.startSpec(spec)
}

func (s *Supervisor) Stop(name string) error {
	spec := s.findSpec(name)
	if spec == nil {
		return fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	for _, rt := range s.runtimesOf(spec.ID) {
		s.stopRuntime(rt, spec.KillTimeout)
	}
	s.requestSave()
	return nil
}

func (s *Supervisor) Restart(name string) error {
	if err := s.Stop(name); err != nil {
		return err
	}
	return s.Start(name)
}

// Reload sends SIGUSR2 to instances (graceful reload for apps that handle it).
// Falls back to Restart after kill_timeout if the process doesn't exit.
func (s *Supervisor) Reload(name string) error {
	spec := s.findSpec(name)
	if spec == nil {
		return fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	for _, rt := range s.runtimesOf(spec.ID) {
		rt.RLock()
		pid := rt.PID
		rt.RUnlock()
		if pid > 0 {
			_ = syscall.Kill(pid, syscall.SIGUSR2)
		}
	}
	return nil
}

func (s *Supervisor) Delete(name string) error {
	spec := s.findSpec(name)
	if spec == nil {
		return fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	_ = s.Stop(spec.Name)
	s.removeRuntimes(spec.ID)
	s.mu.Lock()
	if w := s.watchers[spec.ID]; w != nil {
		w.Close()
		delete(s.watchers, spec.ID)
	}
	delete(s.specs, spec.ID)
	s.mu.Unlock()
	s.requestSave()
	return nil
}

func (s *Supervisor) DeleteAll() {
	s.mu.RLock()
	names := make([]string, 0, len(s.specs))
	for _, sp := range s.specs {
		names = append(names, sp.Name)
	}
	s.mu.RUnlock()
	for _, n := range names {
		_ = s.Delete(n)
	}
}

func (s *Supervisor) StopAll() {
	s.mu.RLock()
	names := make([]string, 0, len(s.specs))
	for _, sp := range s.specs {
		names = append(names, sp.Name)
	}
	s.mu.RUnlock()
	for _, n := range names {
		_ = s.Stop(n)
	}
}

func (s *Supervisor) StartAll() {
	s.mu.RLock()
	specs := make([]*process.Spec, 0, len(s.specs))
	for _, sp := range s.specs {
		specs = append(specs, sp)
	}
	s.mu.RUnlock()
	for _, sp := range specs {
		_ = s.startSpec(sp)
	}
}

// List returns a stable-sorted view of all runtimes.
func (s *Supervisor) List() []process.View {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]process.View, 0, len(s.runtimes))
	for _, rt := range s.runtimes {
		out = append(out, rt.Snapshot())
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].InstanceID < out[j].InstanceID
	})
	return out
}

func (s *Supervisor) Specs() []*process.Spec {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*process.Spec, 0, len(s.specs))
	for _, sp := range s.specs {
		out = append(out, sp.Clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Supervisor) Describe(name string) (*process.Spec, []process.View, error) {
	spec := s.findSpec(name)
	if spec == nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	rts := s.runtimesOf(spec.ID)
	views := make([]process.View, 0, len(rts))
	for _, rt := range rts {
		views = append(views, rt.Snapshot())
	}
	sort.Slice(views, func(i, j int) bool { return views[i].InstanceID < views[j].InstanceID })
	return spec.Clone(), views, nil
}

func (s *Supervisor) TailLogs(name, stream string, n int) (map[string][]string, error) {
	spec := s.findSpec(name)
	if spec == nil {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	out := map[string][]string{}
	for _, rt := range s.runtimesOf(spec.ID) {
		if stream == "" || stream == "out" {
			lines, _ := logs.Tail(rt.ID, "out", n)
			out[rt.ID+":out"] = lines
		}
		if stream == "" || stream == "err" {
			lines, _ := logs.Tail(rt.ID, "err", n)
			out[rt.ID+":err"] = lines
		}
	}
	return out, nil
}

// Subscribe returns a channel that receives Lines for the given app's runtimes.
// The caller must call the returned cancel func to release resources.
func (s *Supervisor) Subscribe(name string) (<-chan logs.Line, func(), error) {
	spec := s.findSpec(name)
	if spec == nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	rts := s.runtimesOf(spec.ID)
	merged := make(chan logs.Line, 256)
	type sub struct {
		stream *logs.Stream
		ch     chan logs.Line
	}
	var subs []sub
	s.mu.RLock()
	for _, rt := range rts {
		sp := s.streams[rt.ID]
		if sp == nil {
			continue
		}
		oc := sp.Out.Subscribe(64)
		ec := sp.Err.Subscribe(64)
		subs = append(subs, sub{sp.Out, oc}, sub{sp.Err, ec})
	}
	s.mu.RUnlock()
	for _, sb := range subs {
		go func(stream *logs.Stream, ch chan logs.Line) {
			for l := range ch {
				select {
				case merged <- l:
				default:
				}
			}
		}(sb.stream, sb.ch)
	}
	cancel := func() {
		for _, sb := range subs {
			sb.stream.Unsubscribe(sb.ch)
		}
	}
	return merged, cancel, nil
}

func (s *Supervisor) Save() error {
	return s.save()
}

// MetricsHistory returns the rolling CPU/mem window per runtime for an app.
// keys are runtime IDs.
func (s *Supervisor) MetricsHistory(name string) (map[string][]metrics.Sample, error) {
	spec := s.findSpec(name)
	if spec == nil {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	out := map[string][]metrics.Sample{}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, rt := range s.runtimes {
		if rt.AppID != spec.ID {
			continue
		}
		if c := s.metrics[rt.ID]; c != nil {
			out[rt.ID] = c.Window()
		}
	}
	return out, nil
}

// ---- internal ----

func (s *Supervisor) findSpec(name string) *process.Spec {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sp := range s.specs {
		if sp.Name == name || sp.ID == name {
			return sp
		}
	}
	return nil
}

func (s *Supervisor) runtimesOf(appID string) []*process.Runtime {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*process.Runtime
	for _, rt := range s.runtimes {
		if rt.AppID == appID {
			out = append(out, rt)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].InstanceID < out[j].InstanceID })
	return out
}

func (s *Supervisor) removeRuntimes(appID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, rt := range s.runtimes {
		if rt.AppID == appID {
			delete(s.runtimes, id)
			if sp := s.streams[id]; sp != nil {
				sp.Out.Close()
				sp.Err.Close()
				delete(s.streams, id)
			}
			delete(s.metrics, id)
		}
	}
}

func (s *Supervisor) startSpec(spec *process.Spec) error {
	for i := 0; i < spec.Instances; i++ {
		rtID := runtimeID(spec, i)
		s.mu.RLock()
		existing := s.runtimes[rtID]
		s.mu.RUnlock()
		var carryRestarts, carryUnstable int
		if existing != nil {
			existing.RLock()
			state := existing.State
			carryRestarts = existing.Restarts
			carryUnstable = existing.UnstableRestarts
			existing.RUnlock()
			if state == process.StateOnline || state == process.StateLaunching {
				continue
			}
			// User-initiated restart counts as one more restart so the UI/CLI
			// reflects the action.
			carryRestarts++
		}
		rt := &process.Runtime{
			ID:               rtID,
			AppID:            spec.ID,
			Name:             spec.Name,
			InstanceID:       i,
			Namespace:        spec.Namespace,
			State:            process.StateLaunching,
			Restarts:         carryRestarts,
			UnstableRestarts: carryUnstable,
		}
		rt.SetStopCh(make(chan struct{}))
		s.mu.Lock()
		s.runtimes[rt.ID] = rt
		s.mu.Unlock()
		go s.supervise(rt, spec)
	}
	return nil
}

func runtimeID(spec *process.Spec, instance int) string {
	if spec.Instances <= 1 {
		return spec.Name
	}
	return fmt.Sprintf("%s-%d", spec.Name, instance)
}

func (s *Supervisor) supervise(rt *process.Runtime, spec *process.Spec) {
	attempts := 0
	for {
		select {
		case <-rt.StopCh():
			s.setState(rt, process.StateStopped, "")
			return
		default:
		}
		cmd, outStream, errStream, err := s.buildCmd(rt, spec)
		if err != nil {
			s.setState(rt, process.StateErrored, err.Error())
			return
		}
		startedAt := time.Now()
		if err := cmd.Start(); err != nil {
			s.setState(rt, process.StateErrored, err.Error())
			outStream.Close()
			errStream.Close()
			return
		}
		rt.Lock()
		rt.SetCmd(cmd)
		rt.PID = cmd.Process.Pid
		rt.StartedAt = startedAt
		rt.State = process.StateLaunching
		rt.ExitedAt = time.Time{}
		rt.LastError = ""
		rt.Unlock()
		s.mu.Lock()
		s.streams[rt.ID] = &streamPair{Out: outStream, Err: errStream}
		s.metrics[rt.ID] = metrics.New(cmd.Process.Pid, metricsWindow)
		s.mu.Unlock()

		// promote to online after MinUptime
		promoteTimer := time.AfterFunc(spec.MinUptime, func() {
			rt.Lock()
			if rt.State == process.StateLaunching {
				rt.State = process.StateOnline
			}
			rt.Unlock()
		})

		err = cmd.Wait()
		promoteTimer.Stop()
		exitCode := exitCodeFromErr(err)
		rt.Lock()
		rt.ExitCode = exitCode
		rt.ExitedAt = time.Now()
		rt.PID = 0
		manual := rt.IsManualStop()
		rt.SetManualStop(false)
		rt.Unlock()
		uptime := time.Since(startedAt)
		s.logger.Printf("%s exited (code=%d uptime=%s)", rt.ID, exitCode, uptime.Truncate(time.Millisecond))

		if manual {
			s.setState(rt, process.StateStopped, "")
			return
		}
		if !spec.Autorestart() {
			s.setState(rt, process.StateStopped, "")
			return
		}
		for _, ec := range spec.StopExitCodes {
			if ec == exitCode {
				s.setState(rt, process.StateStopped, fmt.Sprintf("matched stop_exit_code %d", ec))
				return
			}
		}
		if uptime < spec.MinUptime {
			rt.Lock()
			rt.UnstableRestarts++
			rt.Unlock()
			if spec.MaxRestarts > 0 && rt.UnstableRestarts >= spec.MaxRestarts {
				s.setState(rt, process.StateErrored, fmt.Sprintf("max unstable restarts reached (%d)", spec.MaxRestarts))
				return
			}
		} else {
			rt.Lock()
			rt.UnstableRestarts = 0
			rt.Unlock()
		}
		rt.Lock()
		rt.Restarts++
		rt.Unlock()
		delay := spec.RestartDelay
		if spec.ExpBackoffRestart && uptime < spec.MinUptime {
			n := attempts
			if n > 6 {
				n = 6
			}
			delay = time.Duration(1<<n) * time.Second
		}
		attempts++
		s.setState(rt, process.StateWaitRestart, "")
		select {
		case <-rt.StopCh():
			s.setState(rt, process.StateStopped, "")
			return
		case <-time.After(delay):
		}
	}
}

func (s *Supervisor) buildCmd(rt *process.Runtime, spec *process.Spec) (*exec.Cmd, *logs.Stream, *logs.Stream, error) {
	var prog string
	var args []string
	if spec.Interpreter != "" {
		prog = spec.Interpreter
		args = append(args, spec.InterpreterArgs...)
		args = append(args, spec.Script)
		args = append(args, spec.Args...)
	} else {
		prog = spec.Script
		args = spec.Args
	}
	prog, err := exec.LookPath(prog)
	if err != nil {
		// allow absolute path that exists but isn't on PATH
		if filepath.IsAbs(spec.Script) {
			prog = spec.Script
		} else {
			return nil, nil, nil, fmt.Errorf("lookup %s: %w", spec.Script, err)
		}
	}
	cmd := exec.Command(prog, args...)
	cmd.Env = buildEnv(spec, rt.InstanceID)
	if spec.Cwd != "" {
		cmd.Dir = spec.Cwd
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	rotate := logs.RotateOpts{
		MaxSizeMB:  spec.LogMaxSizeMB,
		MaxBackups: spec.LogMaxBackups,
		MaxAgeDays: spec.LogMaxAgeDays,
		Compress:   spec.LogCompress,
	}
	outStream, err := logs.Open(rt.ID, "out", rotate)
	if err != nil {
		return nil, nil, nil, err
	}
	errStream, err := logs.Open(rt.ID, "err", rotate)
	if err != nil {
		outStream.Close()
		return nil, nil, nil, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		outStream.Close()
		errStream.Close()
		return nil, nil, nil, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		outStream.Close()
		errStream.Close()
		return nil, nil, nil, err
	}
	outStream.Attach(rt.ID, "out", stdoutPipe)
	errStream.Attach(rt.ID, "err", stderrPipe)
	return cmd, outStream, errStream, nil
}

func (s *Supervisor) setState(rt *process.Runtime, st process.State, lastErr string) {
	rt.Lock()
	rt.State = st
	if lastErr != "" {
		rt.LastError = lastErr
	}
	rt.Unlock()
}

func (s *Supervisor) stopRuntime(rt *process.Runtime, killTimeout time.Duration) {
	rt.Lock()
	rt.SetManualStop(true)
	pid := rt.PID
	rt.State = process.StateStopping
	rt.Unlock()
	// signal the supervise loop to exit any pending wait
	select {
	case rt.StopCh() <- struct{}{}:
	default:
	}
	if pid <= 0 {
		return
	}
	// SIGTERM to the process group
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	// wait up to killTimeout, then SIGKILL
	deadline := time.After(killTimeout)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			_ = syscall.Kill(-pid, syscall.SIGKILL)
			return
		case <-tick.C:
			rt.RLock()
			alive := rt.PID != 0
			rt.RUnlock()
			if !alive {
				return
			}
		}
	}
}

func exitCodeFromErr(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
			if ws.Signaled() {
				return 128 + int(ws.Signal())
			}
			return ws.ExitStatus()
		}
	}
	return -1
}

// ---- background loops ----

func (s *Supervisor) metricsLoop() {
	t := time.NewTicker(metricsTickInterval)
	defer t.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-t.C:
			s.tickMetrics()
		}
	}
}

func (s *Supervisor) tickMetrics() {
	s.mu.RLock()
	collectors := make(map[string]*metrics.Collector, len(s.metrics))
	for id, c := range s.metrics {
		collectors[id] = c
	}
	runtimes := make(map[string]*process.Runtime, len(s.runtimes))
	for id, rt := range s.runtimes {
		runtimes[id] = rt
	}
	specs := make(map[string]*process.Spec, len(s.specs))
	for id, sp := range s.specs {
		specs[id] = sp
	}
	s.mu.RUnlock()
	for id, c := range collectors {
		c.Tick()
		latest := c.Latest()
		rt := runtimes[id]
		if rt == nil {
			continue
		}
		rt.Lock()
		rt.CPU = latest.CPU
		rt.MemBytes = latest.MemBytes
		rt.Unlock()
		if sp := specs[rt.AppID]; sp != nil && sp.MaxMemoryRestart > 0 && latest.MemBytes > sp.MaxMemoryRestart {
			s.logger.Printf("%s exceeded max_memory_restart (%d > %d) — restarting", rt.ID, latest.MemBytes, sp.MaxMemoryRestart)
			go s.Restart(rt.Name)
		}
	}
}

func (s *Supervisor) autosaveLoop() {
	t := time.NewTicker(autosaveInterval)
	defer t.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-t.C:
			select {
			case <-s.saveDirty:
				_ = s.save()
			default:
			}
		}
	}
}

func (s *Supervisor) requestSave() {
	select {
	case s.saveDirty <- struct{}{}:
	default:
	}
}

func (s *Supervisor) save() error {
	specs := s.Specs()
	if err := s.store.Save(specs); err != nil {
		s.logger.Printf("save: %v", err)
		return err
	}
	return nil
}

// DescribeKnownNames helps the CLI offer suggestions for typos.
func (s *Supervisor) DescribeKnownNames() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var ns []string
	for _, sp := range s.specs {
		ns = append(ns, sp.Name)
	}
	sort.Strings(ns)
	return strings.Join(ns, ", ")
}
