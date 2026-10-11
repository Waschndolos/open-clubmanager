// Package transport moves immutable protocol files between devices
// (docs/sync-protocol.md §18). Transports know nothing about the content;
// readers filter names with package layout and verify every file.
package transport

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"
)

var (
	// ErrExists is returned by Create when the path already exists.
	ErrExists = errors.New("transport: file exists")
	// ErrNotExist is returned by Get when the path does not exist.
	ErrNotExist = errors.New("transport: file does not exist")
	// ErrInvalidPath is returned for paths that are not clean, relative,
	// slash-separated names inside the sync root.
	ErrInvalidPath = errors.New("transport: invalid path")
)

// FileInfo describes one file.
type FileInfo struct {
	// Path is relative to the sync root, separated by '/'.
	Path    string
	Size    int64
	ModTime time.Time
}

// Transport is the interface of §18.1.
type Transport interface {
	// List returns the files under prefix, in any order. The prefix is matched
	// against the full relative path, so "acl/" lists the ACL directory and
	// "" lists everything.
	List(ctx context.Context, prefix string) ([]FileInfo, error)
	Get(ctx context.Context, path string) ([]byte, error)
	// Create writes a new file. It MUST fail with ErrExists if the path exists.
	Create(ctx context.Context, path string, data []byte) error
	// Delete is used only for compaction and join requests. Deleting a path
	// that does not exist is not an error.
	Delete(ctx context.Context, path string) error
}

// CheckPath validates a path given to a transport.
func CheckPath(path string) error {
	if !fs.ValidPath(path) || path == "." || strings.Contains(path, `\`) {
		return fmt.Errorf("%w: %q", ErrInvalidPath, path)
	}
	return nil
}

// Memory is an in-memory transport for tests.
type Memory struct {
	mu    sync.Mutex
	files map[string][]byte
}

// NewMemory returns an empty in-memory transport.
func NewMemory() *Memory { return &Memory{files: map[string][]byte{}} }

func (m *Memory) List(_ context.Context, prefix string) ([]FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []FileInfo
	for _, p := range slices.Sorted(maps.Keys(m.files)) {
		if strings.HasPrefix(p, prefix) {
			out = append(out, FileInfo{Path: p, Size: int64(len(m.files[p]))})
		}
	}
	return out, nil
}

func (m *Memory) Get(_ context.Context, path string) ([]byte, error) {
	if err := CheckPath(path); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.files[path]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotExist, path)
	}
	return slices.Clone(b), nil
}

func (m *Memory) Create(_ context.Context, path string, data []byte) error {
	if err := CheckPath(path); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.files[path]; ok {
		return fmt.Errorf("%w: %s", ErrExists, path)
	}
	m.files[path] = slices.Clone(data)
	return nil
}

func (m *Memory) Delete(_ context.Context, path string) error {
	if err := CheckPath(path); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, path)
	return nil
}

// Put stores data at path, replacing any existing file. It simulates a sync
// client or an attacker overwriting a file and exists for tests only.
func (m *Memory) Put(path string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[path] = slices.Clone(data)
}
