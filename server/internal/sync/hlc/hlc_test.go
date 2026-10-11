package hlc

import (
	"sort"
	"testing"
	"time"
)

type fakeTime struct{ ms int64 }

func (f *fakeTime) now() time.Time              { return time.UnixMilli(f.ms) }
func (f *fakeTime) sleep(d time.Duration)       { f.ms += d.Milliseconds() }
func (f *fakeTime) clock(last Timestamp) *Clock { return NewWithSource(last, f.now, f.sleep) }

func TestEncoding(t *testing.T) {
	ts := Timestamp{MS: 0x19a1b2c, Counter: 3}
	if got := ts.String(); got != "0000019a1b2c-0003" {
		t.Errorf("String = %q", got)
	}
	back, err := Parse("0000019a1b2c-0003")
	if err != nil || back != ts {
		t.Errorf("Parse = %v, %v", back, err)
	}
	for _, bad := range []string{"", "0000019a1b2c0003", "0000019A1B2C-0003", "019a1b2c-0003", "0000019a1b2c-00003", "0000019a1b2c-000g"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) accepted", bad)
		}
	}
	if (Timestamp{}).String() != "000000000000-0000" {
		t.Errorf("zero = %q", Timestamp{}.String())
	}
}

func TestStringOrderEqualsTimeOrder(t *testing.T) {
	stamps := []Timestamp{{MS: 2, Counter: 0}, {MS: 1, Counter: 0xffff}, {MS: 0xabc, Counter: 1}, {MS: 1, Counter: 2}, {MS: 0xabc, Counter: 0}}
	byString := append([]Timestamp(nil), stamps...)
	sort.Slice(byString, func(i, j int) bool { return byString[i].String() < byString[j].String() })
	byTime := append([]Timestamp(nil), stamps...)
	sort.Slice(byTime, func(i, j int) bool { return byTime[i].Compare(byTime[j]) < 0 })
	for i := range stamps {
		if byString[i] != byTime[i] {
			t.Fatalf("orders differ: %v vs %v", byString, byTime)
		}
	}
}

func TestLessUsesDeviceAsTieBreaker(t *testing.T) {
	a := Timestamp{MS: 5, Counter: 1}
	if !Less(a, "z", Timestamp{MS: 5, Counter: 2}, "a") {
		t.Error("smaller hlc must win regardless of device")
	}
	if !Less(a, "a", a, "b") || Less(a, "b", a, "a") || Less(a, "a", a, "a") {
		t.Error("equal hlc must be ordered by device id")
	}
}

func TestLocalTicks(t *testing.T) {
	f := &fakeTime{ms: 1000}
	c := f.clock(Timestamp{})

	if got := c.Now(); got != (Timestamp{MS: 1000}) {
		t.Errorf("first tick = %v", got)
	}
	if got := c.Now(); got != (Timestamp{MS: 1000, Counter: 1}) {
		t.Errorf("same ms = %v", got)
	}
	f.ms = 1001
	if got := c.Now(); got != (Timestamp{MS: 1001}) {
		t.Errorf("next ms = %v", got)
	}
	// Wall clock going backwards does not move the HLC back.
	f.ms = 500
	if got := c.Now(); got != (Timestamp{MS: 1001, Counter: 1}) {
		t.Errorf("after clock step back = %v", got)
	}
}

func TestCounterOverflowWaitsForNextMillisecond(t *testing.T) {
	f := &fakeTime{ms: 1000}
	c := f.clock(Timestamp{MS: 1000, Counter: maxCounter})
	if got := c.Now(); got != (Timestamp{MS: 1001}) {
		t.Errorf("after overflow = %v", got)
	}
	if f.ms != 1001 {
		t.Errorf("clock did not wait: ms = %d", f.ms)
	}
}

func TestObserve(t *testing.T) {
	f := &fakeTime{ms: 10_000}
	c := f.clock(Timestamp{MS: 10_000})

	if !c.Observe(Timestamp{MS: 20_000, Counter: 7}) {
		t.Error("timestamp within drift rejected")
	}
	if got := c.Now(); got != (Timestamp{MS: 20_000, Counter: 8}) {
		t.Errorf("after observe = %v", got)
	}

	// Older timestamps do not move the clock back.
	c.Observe(Timestamp{MS: 1})
	if c.Last() != (Timestamp{MS: 20_000, Counter: 8}) {
		t.Errorf("observing older moved clock: %v", c.Last())
	}

	// Beyond MaxDrift the clock does not follow.
	far := Timestamp{MS: uint64(f.ms) + uint64(MaxDrift.Milliseconds()) + 1}
	if c.Observe(far) {
		t.Error("timestamp beyond drift accepted")
	}
	if c.Last().Compare(far) >= 0 {
		t.Error("clock followed a timestamp beyond MaxDrift")
	}
}

func TestRestore(t *testing.T) {
	f := &fakeTime{ms: 5}
	c := f.clock(Timestamp{})
	c.Restore(Timestamp{MS: 100, Counter: 2})
	if got := c.Now(); got != (Timestamp{MS: 100, Counter: 3}) {
		t.Errorf("after restore = %v", got)
	}
	c.Restore(Timestamp{MS: 1})
	if c.Last() != (Timestamp{MS: 100, Counter: 3}) {
		t.Errorf("restore moved clock back: %v", c.Last())
	}
}
