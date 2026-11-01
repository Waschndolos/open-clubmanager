package handlers

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Login throttling: after a number of failed attempts for one account (or
// from one source address) further attempts are refused for a lockout that
// doubles with every additional failure. Failures are forgotten once no
// attempt failed for loginFailureWindow.
const (
	loginAccountThreshold = 5
	loginSourceThreshold  = 20
	loginBaseLockout      = 30 * time.Second
	loginMaxLockout       = 15 * time.Minute
	loginFailureWindow    = 15 * time.Minute
)

type loginAttempts struct {
	failures    int
	lastFailure time.Time
	lockedUntil time.Time
}

// loginLimiter tracks failed logins in memory, keyed by account and source.
type loginLimiter struct {
	mu      sync.Mutex
	entries map[string]*loginAttempts
	now     func() time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{entries: map[string]*loginAttempts{}, now: time.Now}
}

func accountKey(email string) string { return "account:" + strings.ToLower(strings.TrimSpace(email)) }

func sourceKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return "source:" + host
}

// retryAfter returns how long the caller must wait before the next attempt
// for the given account and source, or 0 when an attempt is allowed.
func (l *loginLimiter) retryAfter(email string, r *http.Request) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	var wait time.Duration
	for _, key := range []string{accountKey(email), sourceKey(r)} {
		if e, ok := l.entries[key]; ok && e.lockedUntil.After(now) {
			wait = max(wait, e.lockedUntil.Sub(now))
		}
	}
	return wait
}

// failure records a failed attempt for the account and the source.
func (l *loginLimiter) failure(email string, r *http.Request) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.prune(now)
	l.record(accountKey(email), loginAccountThreshold, now)
	l.record(sourceKey(r), loginSourceThreshold, now)
}

// success clears the failures of the account (not of the source, so one
// valid login cannot be used to reset guessing against other accounts).
func (l *loginLimiter) success(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, accountKey(email))
}

func (l *loginLimiter) record(key string, threshold int, now time.Time) {
	e, ok := l.entries[key]
	if !ok || now.Sub(e.lastFailure) > loginFailureWindow {
		e = &loginAttempts{}
		l.entries[key] = e
	}
	e.failures++
	e.lastFailure = now
	if over := e.failures - threshold; over >= 0 {
		lockout := loginMaxLockout
		if over < 10 {
			lockout = min(loginBaseLockout<<over, loginMaxLockout)
		}
		e.lockedUntil = now.Add(lockout)
	}
}

// prune drops entries that neither lock nor count anymore.
func (l *loginLimiter) prune(now time.Time) {
	for key, e := range l.entries {
		if now.After(e.lockedUntil) && now.Sub(e.lastFailure) > loginFailureWindow {
			delete(l.entries, key)
		}
	}
}
