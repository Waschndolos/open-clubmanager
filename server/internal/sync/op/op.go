// Package op defines the plaintext change operation of the sync protocol
// (docs/sync-protocol.md §10).
package op

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Op changes one row.
type Op struct {
	// T is the table name.
	T string `json:"t"`
	// PK holds the primary key columns and their values.
	PK map[string]any `json:"pk"`
	// HLC is the timestamp of the change (hlc.Timestamp.String()).
	HLC string `json:"hlc"`
	// Set holds the columns with their new values. It never contains
	// updated_at, which every device derives locally.
	Set map[string]any `json:"set"`
	// Purge lists columns to erase (§15.3). Not written yet.
	Purge []string `json:"purge,omitempty"`
}

// EncodePK returns the canonical text form of a primary key: a JSON object
// with sorted keys. It identifies a row in the local sync tables.
func EncodePK(pk map[string]any) string {
	b, err := json.Marshal(pk)
	if err != nil {
		// Primary keys are strings; this cannot happen.
		panic(fmt.Sprintf("op: encode pk %v: %v", pk, err))
	}
	return string(b)
}

// DecodePK parses the form written by EncodePK.
func DecodePK(s string) (map[string]any, error) {
	var pk map[string]any
	if err := decode(s, &pk); err != nil {
		return nil, fmt.Errorf("op: decode pk %q: %w", s, err)
	}
	for k, v := range pk {
		pk[k] = Normalize(v)
	}
	return pk, nil
}

// EncodeValue returns the JSON text of a column value.
func EncodeValue(v any) string {
	b, err := json.Marshal(Normalize(v))
	if err != nil {
		panic(fmt.Sprintf("op: encode value %v: %v", v, err))
	}
	return string(b)
}

// DecodeValue parses the form written by EncodeValue. Integral numbers come
// back as int64.
func DecodeValue(s string) (any, error) {
	var v any
	if err := decode(s, &v); err != nil {
		return nil, fmt.Errorf("op: decode value %q: %w", s, err)
	}
	return Normalize(v), nil
}

func decode(s string, out any) error {
	d := json.NewDecoder(bytes.NewReader([]byte(s)))
	d.UseNumber()
	return d.Decode(out)
}

// Normalize maps a Go or SQLite column value to the types ops carry: nil,
// string, int64, float64 or bool. Two values are equal in the protocol iff
// their normalized forms are ==.
func Normalize(v any) any {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case uint16:
		return int64(x)
	case uint32:
		return int64(x)
	case []byte:
		return string(x)
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i
		}
		f, _ := x.Float64()
		return f
	}
	return v
}
