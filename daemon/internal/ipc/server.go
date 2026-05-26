package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/daemon"
	"github.com/huseinzol05/pm2-go/daemon/internal/paths"
	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

type Server struct {
	sup      *daemon.Supervisor
	listener net.Listener
	server   *http.Server
	shutdown chan struct{}
}

func NewServer(sup *daemon.Supervisor) *Server {
	return &Server{sup: sup, shutdown: make(chan struct{})}
}

// Serve binds the Unix socket and serves requests until the context is cancelled
// or /v1/shutdown is called.
func (s *Server) Serve(ctx context.Context) error {
	sock := paths.RPCSocket()
	_ = os.Remove(sock)
	lis, err := net.Listen("unix", sock)
	if err != nil {
		return fmt.Errorf("listen %s: %w", sock, err)
	}
	if err := os.Chmod(sock, 0o600); err != nil {
		return err
	}
	s.listener = lis

	mux := http.NewServeMux()
	mux.HandleFunc(PathPing, s.handlePing)
	mux.HandleFunc(PathList, s.handleList)
	mux.HandleFunc(PathSpecs, s.handleSpecs)
	mux.HandleFunc(PathDescribe, s.handleDescribe)
	mux.HandleFunc(PathStart, s.handleStart)
	mux.HandleFunc(PathStartSpec, s.handleStartSpec)
	mux.HandleFunc(PathStop, s.handleStop)
	mux.HandleFunc(PathRestart, s.handleRestart)
	mux.HandleFunc(PathReload, s.handleReload)
	mux.HandleFunc(PathDelete, s.handleDelete)
	mux.HandleFunc(PathStopAll, s.handleStopAll)
	mux.HandleFunc(PathStartAll, s.handleStartAll)
	mux.HandleFunc(PathDeleteAll, s.handleDeleteAll)
	mux.HandleFunc(PathSave, s.handleSave)
	mux.HandleFunc(PathTail, s.handleTail)
	mux.HandleFunc(PathStreamLogs, s.handleStreamLogs)
	mux.HandleFunc(PathShutdown, s.handleShutdown)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // disabled for streaming endpoints
		IdleTimeout:  120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.Serve(lis)
	}()
	select {
	case <-ctx.Done():
	case <-s.shutdown:
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.server.Shutdown(shutdownCtx)
	_ = os.Remove(sock)
	return nil
}

func (s *Server) handlePing(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"pid":     os.Getpid(),
		"started": time.Now().Format(time.RFC3339),
		"ok":      true,
	})
}

func (s *Server) handleList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, ListResponse{Procs: s.sup.List()})
}

func (s *Server) handleSpecs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, SpecsResponse{Specs: s.sup.Specs()})
}

func (s *Server) handleDescribe(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	spec, procs, err := s.sup.Describe(name)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DescribeResponse{Spec: spec, Procs: procs})
}

func (s *Server) handleStartSpec(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) nameAction(w http.ResponseWriter, r *http.Request, fn func(string) error) {
	var req NameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	if err := fn(req.Name); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request)   { s.nameAction(w, r, s.sup.Start) }
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request)    { s.nameAction(w, r, s.sup.Stop) }
func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) { s.nameAction(w, r, s.sup.Restart) }
func (s *Server) handleReload(w http.ResponseWriter, r *http.Request)  { s.nameAction(w, r, s.sup.Reload) }
func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request)  { s.nameAction(w, r, s.sup.Delete) }

func (s *Server) handleStopAll(w http.ResponseWriter, _ *http.Request) {
	s.sup.StopAll()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func (s *Server) handleStartAll(w http.ResponseWriter, _ *http.Request) {
	s.sup.StartAll()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func (s *Server) handleDeleteAll(w http.ResponseWriter, _ *http.Request) {
	s.sup.DeleteAll()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleSave(w http.ResponseWriter, _ *http.Request) {
	if err := s.sup.Save(); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleTail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	name := q.Get("name")
	n, _ := strconv.Atoi(q.Get("n"))
	if n <= 0 {
		n = 20
	}
	streams, err := s.sup.TailLogs(name, q.Get("stream"), n)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, TailResponse{Streams: streams})
}

// handleStreamLogs sends newline-delimited JSON log lines until the client
// disconnects.
func (s *Server) handleStreamLogs(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	ch, cancel, err := s.sup.Subscribe(name)
	if err != nil {
		writeErr(w, err)
		return
	}
	defer cancel()

	flusher, _ := w.(http.Flusher)
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	notify := r.Context().Done()
	for {
		select {
		case <-notify:
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

func (s *Server) handleShutdown(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(s.shutdown)
	}()
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
	writeJSON(w, code, ErrorResponse{Error: err.Error()})
}
