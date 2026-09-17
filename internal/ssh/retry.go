package ssh

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"time"
)

// RetryPolicy controls transient SSH/network retry behavior.
type RetryPolicy struct {
	Attempts  int
	BaseDelay time.Duration
	MaxDelay  time.Duration
	Jitter    time.Duration
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{Attempts: 4, BaseDelay: 250 * time.Millisecond, MaxDelay: 4 * time.Second, Jitter: 150 * time.Millisecond}
}

// Retry executes fn until it succeeds, the context is cancelled, or Attempts
// is exhausted. Authentication and host-key failures are returned immediately.
func Retry(ctx context.Context, policy RetryPolicy, fn func() error) error {
	if policy.Attempts < 1 {
		policy.Attempts = 1
	}
	if policy.BaseDelay <= 0 {
		policy.BaseDelay = 250 * time.Millisecond
	}
	if policy.MaxDelay <= 0 {
		policy.MaxDelay = 4 * time.Second
	}

	var last error
	for attempt := 0; attempt < policy.Attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := fn(); err == nil {
			return nil
		} else {
			last = err
			var unknown *UnknownHostKeyError
			var changed *ChangedHostKeyError
			if errors.As(err, &unknown) || errors.As(err, &changed) || !isTransient(err) {
				return err
			}
		}
		if attempt == policy.Attempts-1 {
			break
		}
		delay := policy.BaseDelay << attempt
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}
		if policy.Jitter > 0 {
			delay += time.Duration(rand.Int63n(int64(policy.Jitter)))
		}
		t := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
	return fmt.Errorf("operation failed after %d attempts: %w", policy.Attempts, last)
}

func isTransient(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) {
		return ne.Timeout() || ne.Temporary()
	}
	return true
}

// ContextDialer creates a TCP connection with a hard deadline.
func ContextDialer(ctx context.Context, address string, timeout time.Duration) (net.Conn, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	d := net.Dialer{Timeout: timeout}
	return d.DialContext(ctx, "tcp", address)
}
