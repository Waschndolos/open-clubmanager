// Package dsar assembles a data subject access report for one member
// (GDPR Art. 15 access, Art. 20 portability).
//
// Every module that stores member data registers a Provider; the report is
// the output of all registered providers. A new module only has to call
// Register (usually from an init function) — nothing here is hard-wired.
package dsar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

// Field is one stored value. Label is shown in the PDF, Key is the JSON name.
type Field struct {
	Key   string
	Label string
	Value any
}

// Record is one stored entity (a fee, a loan, ...), as ordered fields.
type Record []Field

// MarshalJSON writes the record as an object that keeps the field order.
func (r Record) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, f := range r {
		if i > 0 {
			buf.WriteByte(',')
		}
		k, err := json.Marshal(f.Key)
		if err != nil {
			return nil, err
		}
		v, err := json.Marshal(f.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(k)
		buf.WriteByte(':')
		buf.Write(v)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// Section is the output of one provider.
type Section struct {
	Key     string   `json:"key"`
	Title   string   `json:"title"`
	Records []Record `json:"records"`
}

// Report is the complete export for one member.
type Report struct {
	GeneratedAt time.Time `json:"generatedAt"`
	GeneratedBy string    `json:"generatedBy"`
	MemberID    string    `json:"memberId"`
	Sections    []Section `json:"sections"`
}

// Provider contributes one section of the report.
type Provider struct {
	// Key is the stable JSON key of the section.
	Key string
	// Title is the heading in the readable PDF.
	Title string
	// Scope is the data scope the provider reads. A report is only created
	// for callers who may read every registered scope.
	Scope db.Scope
	// Collect returns all records of the member; it returns none if the
	// member has no data in this module.
	Collect func(ctx context.Context, st *store.Store, memberID string) ([]Record, error)
}

var (
	mu        sync.RWMutex
	providers []Provider
)

// Register adds a provider. Registering a key twice replaces the first.
func Register(p Provider) {
	mu.Lock()
	defer mu.Unlock()
	for i := range providers {
		if providers[i].Key == p.Key {
			providers[i] = p
			return
		}
	}
	providers = append(providers, p)
}

// Providers returns the registered providers in registration order.
func Providers() []Provider {
	mu.RLock()
	defer mu.RUnlock()
	return append([]Provider(nil), providers...)
}

// Scopes returns the distinct scopes the registered providers read.
func Scopes() []db.Scope {
	seen := map[db.Scope]bool{}
	var out []db.Scope
	for _, p := range Providers() {
		if !seen[p.Scope] {
			seen[p.Scope] = true
			out = append(out, p.Scope)
		}
	}
	return out
}

// Build runs every provider for the member. It fails with store.ErrNotFound
// if the member does not exist.
func Build(ctx context.Context, st *store.Store, memberID, generatedBy string) (*Report, error) {
	if _, err := st.GetMember(ctx, memberID); err != nil {
		return nil, err
	}
	rep := &Report{GeneratedAt: time.Now().UTC(), GeneratedBy: generatedBy, MemberID: memberID, Sections: []Section{}}
	for _, p := range Providers() {
		recs, err := p.Collect(ctx, st, memberID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.Key, err)
		}
		if recs == nil {
			recs = []Record{}
		}
		rep.Sections = append(rep.Sections, Section{Key: p.Key, Title: p.Title, Records: recs})
	}
	return rep, nil
}
