package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Store struct { mu sync.Mutex; path string; Hosts map[string][]string `json:"hosts"` }

func New(path string) *Store { return &Store{path: path, Hosts: map[string][]string{}} }
func Default() *Store { home, _ := os.UserHomeDir(); return New(filepath.Join(home, ".config", "vecna", "history.json")) }

func (s *Store) Load() error { s.mu.Lock(); defer s.mu.Unlock(); b, err := os.ReadFile(s.path); if os.IsNotExist(err) { return nil }; if err != nil { return err }; var v struct{ Hosts map[string][]string `json:"hosts"` }; if err := json.Unmarshal(b, &v); err != nil { return err }; if v.Hosts != nil { s.Hosts = v.Hosts }; return nil }
func (s *Store) Save() error { s.mu.Lock(); defer s.mu.Unlock(); if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil { return err }; b, err := json.MarshalIndent(struct{ Hosts map[string][]string `json:"hosts"` }{s.Hosts}, "", "  "); if err != nil { return err }; return os.WriteFile(s.path, b, 0600) }
func (s *Store) Add(host, command string) error { command = strings.TrimSpace(command); if command == "" { return nil }; s.mu.Lock(); items := s.Hosts[host]; if len(items) > 0 && items[len(items)-1] == command { s.mu.Unlock(); return nil }; filtered := items[:0]; for _, item := range items { if item != command { filtered = append(filtered, item) } }; s.Hosts[host] = append(filtered, command); if len(s.Hosts[host]) > 200 { s.Hosts[host] = s.Hosts[host][len(s.Hosts[host])-200:] }; s.mu.Unlock(); return s.Save() }
func (s *Store) Get(host string) []string { s.mu.Lock(); defer s.mu.Unlock(); return append([]string(nil), s.Hosts[host]...) }
