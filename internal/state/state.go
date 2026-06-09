package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Store struct {
	path     string
	seen     map[string]struct{}
	mu       sync.Mutex
	loaded   bool
	modified bool
}

func New(path string) (*Store, error) {
	expanded, err := expandPath(path)
	if err != nil {
		return nil, err
	}
	return &Store{
		path: expanded,
		seen: make(map[string]struct{}),
	}, nil
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.loaded = true
			return nil
		}
		return fmt.Errorf("read state: %w", err)
	}

	var keys []string
	if err := json.Unmarshal(data, &keys); err != nil {
		return fmt.Errorf("parse state: %w", err)
	}
	s.seen = make(map[string]struct{}, len(keys))
	for _, key := range keys {
		s.seen[key] = struct{}{}
	}
	s.loaded = true
	return nil
}

func (s *Store) Seen(mailbox string, uid uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.seen[key(mailbox, uid)]
	return ok
}

func (s *Store) Mark(mailbox string, uid uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.seen[key(mailbox, uid)]; ok {
		return
	}
	s.seen[key(mailbox, uid)] = struct{}{}
	s.modified = true
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.modified {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	keys := make([]string, 0, len(s.seen))
	for key := range s.seen {
		keys = append(keys, key)
	}

	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	s.modified = false
	return nil
}

func key(mailbox string, uid uint32) string {
	return fmt.Sprintf("%s:%d", mailbox, uid)
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home dir: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}
	return filepath.Clean(path), nil
}
