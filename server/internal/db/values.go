package db

import (
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// TimeLayout is the storage format for all timestamps and dates: UTC with
// millisecond precision, which sorts lexicographically.
const TimeLayout = "2006-01-02T15:04:05.000Z"

// AdminUserID identifies the administrator configured in config.json, who has
// no row in the users table.
const AdminUserID = "00000000-0000-0000-0000-000000000000"

// NewID returns a new UUIDv7 primary key. UUIDv7 is time-ordered, which keeps
// SQLite's B-trees compact and gives a natural creation order across devices.
func NewID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// FormatTime formats t for storage.
func FormatTime(t time.Time) string {
	return t.UTC().Format(TimeLayout)
}

// Now returns the current time formatted for storage.
func Now() string {
	return FormatTime(time.Now())
}

var parseLayouts = []string{
	TimeLayout,
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// ParseTime parses a stored timestamp. It also accepts the formats written by
// earlier versions (RFC 3339 without milliseconds, SQLite CURRENT_TIMESTAMP,
// plain dates).
func ParseTime(s string) (time.Time, bool) {
	for _, layout := range parseLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

// ToCents converts a currency amount to integer cents.
func ToCents(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

// FromCents converts integer cents back to a currency amount.
func FromCents(cents int64) float64 {
	return float64(cents) / 100
}

// normalizeLegacyTime converts a timestamp as stored by Prisma (Unix epoch
// milliseconds) or by the previous Go handlers (text) into TimeLayout.
func normalizeLegacyTime(v any) (string, bool) {
	switch x := v.(type) {
	case nil:
		return "", false
	case int64:
		return FormatTime(time.UnixMilli(x)), true
	case float64:
		return FormatTime(time.UnixMilli(int64(x))), true
	case []byte:
		return normalizeLegacyTime(string(x))
	case time.Time:
		return FormatTime(x), true
	case string:
		if x == "" {
			return "", false
		}
		if ms, err := strconv.ParseInt(x, 10, 64); err == nil {
			return FormatTime(time.UnixMilli(ms)), true
		}
		if t, ok := ParseTime(x); ok {
			return FormatTime(t), true
		}
	}
	return "", false
}
