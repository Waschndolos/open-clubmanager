// Package hlc implements the hybrid logical clock of the sync protocol
// (docs/sync-protocol.md §12): a 48-bit physical time in milliseconds plus a
// 16-bit counter, encoded so that string order equals time order.
package hlc

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

const (
	maxMS      = 1<<48 - 1
	maxCounter = 1<<16 - 1

	// MaxDrift is how far a received timestamp may lie ahead of the local
	// wall clock for the local clock to still follow it.
	MaxDrift = time.Hour
)

// Timestamp is one HLC value. The zero value sorts before every timestamp a
// clock produces.
type Timestamp struct {
	MS      uint64 // physical time, Unix milliseconds (48 bits)
	Counter uint16
}

// String encodes ts as "<12 hex digits ms>-<4 hex digits counter>".
func (ts Timestamp) String() string {
	return fmt.Sprintf("%012x-%04x", ts.MS&maxMS, ts.Counter)
}

// Time returns the physical part as a time.
func (ts Timestamp) Time() time.Time {
	return time.UnixMilli(int64(ts.MS)).UTC()
}

// Compare returns -1, 0 or +1 as ts is before, equal to or after o.
func (ts Timestamp) Compare(o Timestamp) int {
	switch {
	case ts.MS < o.MS:
		return -1
	case ts.MS > o.MS:
		return 1
	case ts.Counter < o.Counter:
		return -1
	case ts.Counter > o.Counter:
		return 1
	}
	return 0
}

// Parse decodes the String form. Only lower-case hex is accepted, so that
// every timestamp has exactly one encoding.
func Parse(s string) (Timestamp, error) {
	if len(s) != 17 || s[12] != '-' || !lowerHex(s[:12]) || !lowerHex(s[13:]) {
		return Timestamp{}, fmt.Errorf("hlc: malformed timestamp %q", s)
	}
	ms, err := strconv.ParseUint(s[:12], 16, 64)
	if err != nil {
		return Timestamp{}, fmt.Errorf("hlc: malformed timestamp %q", s)
	}
	c, err := strconv.ParseUint(s[13:], 16, 16)
	if err != nil {
		return Timestamp{}, fmt.Errorf("hlc: malformed timestamp %q", s)
	}
	return Timestamp{MS: ms, Counter: uint16(c)}, nil
}

func lowerHex(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// Less reports whether change a (with its author device) is ordered before
// change b in the protocol's total order: by timestamp, then by device id.
func Less(a Timestamp, aDevice string, b Timestamp, bDevice string) bool {
	if c := a.Compare(b); c != 0 {
		return c < 0
	}
	return aDevice < bDevice
}

// Clock hands out strictly increasing timestamps. It is safe for concurrent
// use.
type Clock struct {
	mu    sync.Mutex
	last  Timestamp
	now   func() time.Time
	sleep func(time.Duration)
}

// New returns a clock driven by the system time that continues after last
// (the persisted value, or the zero Timestamp).
func New(last Timestamp) *Clock {
	return &Clock{last: last, now: time.Now, sleep: time.Sleep}
}

// NewWithSource is New with an injectable time source, for tests.
func NewWithSource(last Timestamp, now func() time.Time, sleep func(time.Duration)) *Clock {
	return &Clock{last: last, now: now, sleep: sleep}
}

func (c *Clock) nowMS() uint64 {
	ms := c.now().UnixMilli()
	if ms < 0 {
		return 0
	}
	return uint64(ms) & maxMS
}

// Now ticks the clock for a local change and returns the new timestamp. If the
// counter would overflow, it waits for the next millisecond.
func (c *Clock) Now() Timestamp {
	c.mu.Lock()
	defer c.mu.Unlock()
	for {
		if pt := c.nowMS(); pt > c.last.MS {
			c.last = Timestamp{MS: pt}
			return c.last
		}
		if c.last.Counter < maxCounter {
			c.last.Counter++
			return c.last
		}
		c.sleep(time.Millisecond)
	}
}

// Last returns the most recent timestamp without ticking.
func (c *Clock) Last() Timestamp {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.last
}

// Observe merges a timestamp received from another device. The clock follows
// r unless r lies more than MaxDrift ahead of the local wall clock; in that
// case it returns false so the caller can warn about the author's clock. The
// op is applied either way.
func (c *Clock) Observe(r Timestamp) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r.MS > c.nowMS()+uint64(MaxDrift.Milliseconds()) {
		return false
	}
	if r.Compare(c.last) > 0 {
		c.last = r
	}
	return true
}

// Restore moves the clock forward to ts if ts is later, without the drift
// check. It is used to resume from the persisted local value.
func (c *Clock) Restore(ts Timestamp) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ts.Compare(c.last) > 0 {
		c.last = ts
	}
}
