// Package api serves an authenticated REST + ndjson streaming surface over TCP
// for the web UI to consume. Token is read from ~/.pm2-go/api-token; address
// from ~/.pm2-go/api-port. Disabled if either file is missing.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/daemon"
	"github.com/huseinzol05/pm2-go/daemon/internal/ipc"
	"github.com/huseinzol05/pm2-go/daemon/internal/paths"
	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

type Server struct {
	sup    *daemon.Supervisor
	token  string
	addr   string
	server *http.Server
}

func Serve(ctx context.Context, sup *daemon.Supervisor) error {
	addr, token, err := readConfig()
	if err != nil {
		return err
	}
	if addr == "" || token == "" {
		// not configured — return without error so the daemon can still run
		return nil
	}
	s := &Server{sup: sup, token: token, addr: addr}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/v1/procs", s.auth(s.procs))
	mux.HandleFunc("/v1/specs", s.auth(s.specs))
	mux.HandleFunc("/v1/describe", s.auth(s.describe))
	mux.HandleFunc("/v1/start", s.auth(s.start))
	mux.HandleFunc("/v1/stop", s.auth(s.stop))
	mux.HandleFunc("/v1/restart", s.auth(s.restart))
	mux.HandleFunc("/v1/reload", s.auth(s.reload))
	mux.HandleFunc("/v1/delete", s.auth(s.del))
	mux.HandleFunc("/v1/logs", s.auth(s.tail))
	mux.HandleFunc("/v1/logs/stream", s.auth(s.stream))
	mux.HandleFunc("/v1/metrics", s.auth(s.metrics))
	mux.HandleFunc("/v1/save", s.auth(s.save))
	mux.HandleFunc("/v1/stop-all", s.auth(s.stopAll))
	mux.HandleFunc("/v1/start-all", s.auth(s.startAll))
	mux.HandleFunc("/v1/delete-all", s.auth(s.deleteAll))
	s.server = &http.Server{
		Addr:         addr,
		Handler:      withCORS(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() { errCh <- s.server.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func readConfig() (addr, token string, err error) {
	// Env vars win over files so docker-compose can wire this up without an
	// init step.
	if v := strings.TrimSpace(os.Getenv("PM2_GO_API_ADDR")); v != "" {
		addr = v
	} else if b, e := os.ReadFile(paths.APIPortFile()); e == nil {
		addr = strings.TrimSpace(string(b))
	}
	if v := strings.TrimSpace(os.Getenv("PM2_GO_API_TOKEN")); v != "" {
		token = v
	} else if b, e := os.ReadFile(paths.APIToken()); e == nil {
		token = strings.TrimSpace(string(b))
	}
	return addr, token, nil
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != s.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) procs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, ipc.ListResponse{Procs: s.sup.List()})
}

func (s *Server) specs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, ipc.SpecsResponse{Specs: s.sup.Specs()})
}

func (s *Server) describe(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	sp, procs, err := s.sup.Describe(name)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ipc.DescribeResponse{Spec: sp, Procs: procs})
}

func (s *Server) start(w http.ResponseWriter, r *http.Request) {
	// support both JSON spec body and ?name=<existing>
	if name := r.URL.Query().Get("name"); name != "" {
		if err := s.sup.Start(name); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	var spec process.Spec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeErr(w, err)
		return
	}
	if err := s.sup.Add(&spec); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"name": spec.Name})
}

func (s *Server) byName(w http.ResponseWriter, r *http.Request, do func(string) error) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}
	if err := do(name); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) stop(w http.ResponseWriter, r *http.Request)    { s.byName(w, r, s.sup.Stop) }
func (s *Server) restart(w http.ResponseWriter, r *http.Request) { s.byName(w, r, s.sup.Restart) }
func (s *Server) reload(w http.ResponseWriter, r *http.Request)  { s.byName(w, r, s.sup.Reload) }
func (s *Server) del(w http.ResponseWriter, r *http.Request)     { s.byName(w, r, s.sup.Delete) }

func (s *Server) tail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	n, _ := strconv.Atoi(q.Get("n"))
	if n <= 0 {
		n = 50
	}
	streams, err := s.sup.TailLogs(q.Get("name"), q.Get("stream"), n)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ipc.TailResponse{Streams: streams})
}

func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	hist, err := s.sup.MetricsHistory(name)
	if err != nil {
		writeErr(w, err)
		return
	}
	series := make(map[string][]ipc.MetricSample, len(hist))
	for id, samples := range hist {
		out := make([]ipc.MetricSample, len(samples))
		for i, sm := range samples {
			out[i] = ipc.MetricSample{Time: sm.Time.UnixMilli(), CPU: sm.CPU, Mem: sm.MemBytes}
		}
		series[id] = out
	}
	writeJSON(w, http.StatusOK, ipc.MetricsResponse{Series: series})
}

func (s *Server) save(w http.ResponseWriter, _ *http.Request) {
	if err := s.sup.Save(); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) stopAll(w http.ResponseWriter, _ *http.Request) {
	s.sup.StopAll()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) startAll(w http.ResponseWriter, _ *http.Request) {
	s.sup.StartAll()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) deleteAll(w http.ResponseWriter, _ *http.Request) {
	s.sup.DeleteAll()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	ch, cancel, err := s.sup.Subscribe(name)
	if err != nil {
		writeErr(w, err)
		return
	}
	defer cancel()
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	enc := json.NewEncoder(w)
	for {
		select {
		case <-r.Context().Done():
			return
		case line, ok := <-ch:
			if !ok {
				return
			}
			if err := enc.Encode(line); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	if errors.Is(err, daemon.ErrNotFound) {
		code = http.StatusNotFound
	}
	writeJSON(w, code, ipc.ErrorResponse{Error: err.Error()})
}

func withCORS(next http.Handler) http.Handler {
	allow := os.Getenv("PM2_GO_API_CORS")
	if allow == "" {
		allow = "*"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allow)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// unused but kept for callers wanting a prepared addr string
func _portStr(p int) string { return fmt.Sprintf(":%d", p) }
