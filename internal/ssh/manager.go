package ssh

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

type SessionFactory func() (*Session, error)

type SessionManager struct {
	mu sync.Mutex
	session *Session
	factory SessionFactory
	policy RetryPolicy
}

func NewSessionManager(factory SessionFactory) *SessionManager { return &SessionManager{factory: factory, policy: DefaultRetryPolicy()} }
func (m *SessionManager) Connect(ctx context.Context) error { m.mu.Lock(); defer m.mu.Unlock(); if m.session != nil { return nil }; var s *Session; err := Retry(ctx, m.policy, func() error { var e error; s, e = m.factory(); return e }); if err != nil { return err }; m.session = s; return nil }
func (m *SessionManager) Session() *Session { m.mu.Lock(); defer m.mu.Unlock(); return m.session }
func (m *SessionManager) Close() error { m.mu.Lock(); defer m.mu.Unlock(); if m.session == nil { return nil }; err := m.session.Close(); m.session = nil; return err }

// Ensure reconnects when the TCP channel is stale. It intentionally does not
// guess at application-level liveness; callers can run a harmless probe.
func (m *SessionManager) Ensure(ctx context.Context, probe func(*Session) error) error {
	m.mu.Lock(); s := m.session; m.mu.Unlock()
	if s != nil && probe != nil { if err := probe(s); err == nil { return nil } }
	_ = m.Close(); return m.Connect(ctx)
}

func KeepAlive(ctx context.Context, conn net.Conn, interval time.Duration) error {
	if interval <= 0 { interval = 30 * time.Second }; ticker := time.NewTicker(interval); defer ticker.Stop()
	for { select { case <-ctx.Done(): return ctx.Err(); case <-ticker.C: if _, err := conn.Write([]byte{}); err != nil { return fmt.Errorf("keepalive: %w", err) } } }
}
