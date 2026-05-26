// Package store persists the canonical list of Specs to disk (~/.pm2-go/dump.json).
// Runtime state is rebuilt at daemon start from this file ("resurrect").
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/huseinzol05/pm2-go/daemon/internal/paths"
	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

type Dump struct {
	Version int               `json:"version"`
	Specs   []*process.Spec   `json:"specs"`
}

type Store struct {
	mu    sync.Mutex
	path  string
	cache *Dump
}

func New() *Store {
	return &Store{path: paths.DumpFile()}
}

func (s *Store) Load() (*Dump, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cache != nil {
		return s.cache, nil
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.cache = &Dump{Version: 1}
			return s.cache, nil
		}
		return nil, err
	}
	var d Dump
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("parse %s: %w", s.path, err)
	}
	if d.Version == 0 {
		d.Version = 1
	}
	s.cache = &d
	return s.cache, nil
}

func (s *Store) Save(specs []*process.Spec) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := &Dump{Version: 1, Specs: specs}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	s.cache = d
	return nil
}

func (s *Store) Path() string { return s.path }
