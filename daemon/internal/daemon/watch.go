package daemon

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

// Watcher batches fs change events for a spec and triggers a restart when the
// debounce window expires.
type Watcher struct {
	spec     *process.Spec
	fs       *fsnotify.Watcher
	stop     chan struct{}
	once     sync.Once
	debounce time.Duration
	trigger  func()
}

// NewWatcher creates and starts a watcher for spec.Watch paths.
// trigger is called (debounced) when a non-ignored file changes.
func NewWatcher(spec *process.Spec, trigger func()) (*Watcher, error) {
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		spec:     spec,
		fs:       fs,
		stop:     make(chan struct{}),
		debounce: 500 * time.Millisecond,
		trigger:  trigger,
	}
	for _, p := range spec.Watch {
		path := p
		if !filepath.IsAbs(path) && spec.Cwd != "" {
			path = filepath.Join(spec.Cwd, path)
		}
		if err := addRecursive(fs, path); err != nil {
			log.Printf("watch %s: %v", path, err)
		}
	}
	go w.loop()
	return w, nil
}

func addRecursive(fs *fsnotify.Watcher, root string) error {
	st, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fs.Add(root)
	}
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		if info.IsDir() {
			// skip hidden + node_modules by convention
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "node_modules" {
				return filepath.SkipDir
			}
			return fs.Add(path)
		}
		return nil
	})
}

func (w *Watcher) loop() {
	var timer *time.Timer
	for {
		select {
		case <-w.stop:
			return
		case ev, ok := <-w.fs.Events:
			if !ok {
				return
			}
			if w.ignored(ev.Name) {
				continue
			}
			if timer == nil {
				timer = time.AfterFunc(w.debounce, w.trigger)
			} else {
				timer.Reset(w.debounce)
			}
		case err, ok := <-w.fs.Errors:
			if !ok {
				return
			}
			log.Printf("watch error: %v", err)
		}
	}
}

func (w *Watcher) ignored(path string) bool {
	for _, pat := range w.spec.IgnoreWatch {
		ok, _ := filepath.Match(pat, filepath.Base(path))
		if ok {
			return true
		}
		ok, _ = filepath.Match(pat, path)
		if ok {
			return true
		}
	}
	return false
}

func (w *Watcher) Close() {
	w.once.Do(func() {
		close(w.stop)
		_ = w.fs.Close()
	})
}
