package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
)

// TempPrefix starts the names of files LocalFolder is still writing. Readers
// ignore them because they match no pattern of §6.
const TempPrefix = ".tmp-"

// LocalFolder is the transport for a folder synced by a cloud client
// (§18.2). All access goes through an os.Root, so paths cannot escape the
// folder, also not through symbolic links.
type LocalFolder struct {
	root *os.Root
	// noLink forces the rename fallback of Create, for tests.
	noLink bool
}

// OpenLocalFolder opens dir, which must exist, as a sync root.
func OpenLocalFolder(dir string) (*LocalFolder, error) {
	r, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open sync folder: %w", err)
	}
	return &LocalFolder{root: r}, nil
}

// Close releases the folder.
func (l *LocalFolder) Close() error { return l.root.Close() }

func (l *LocalFolder) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	// Walk the deepest directory named by the prefix, then filter by the
	// whole prefix.
	dir := "."
	if i := strings.LastIndex(prefix, "/"); i >= 0 {
		dir = prefix[:i]
		if err := CheckPath(dir); err != nil {
			return nil, err
		}
	}
	var out []FileInfo
	err := fs.WalkDir(l.root.FS(), dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && p == dir {
				return fs.SkipAll
			}
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !d.Type().IsRegular() || !strings.HasPrefix(p, prefix) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil // removed while listing
			}
			return err
		}
		out = append(out, FileInfo{Path: p, Size: info.Size(), ModTime: info.ModTime()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (l *LocalFolder) Get(_ context.Context, p string) ([]byte, error) {
	if err := CheckPath(p); err != nil {
		return nil, err
	}
	b, err := l.root.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrNotExist, p)
	}
	return b, err
}

// Create writes the file as required by §18.2: into a ".tmp-<random>" file in
// the target directory, fsync, then hard-link to the target name (which fails
// if it exists), then remove the temp file. If the file system does not
// support hard links it falls back to a rename after an existence check; that
// fallback is not atomic against a concurrent writer of the same name.
func (l *LocalFolder) Create(_ context.Context, p string, data []byte) error {
	if err := CheckPath(p); err != nil {
		return err
	}
	dir := path.Dir(p)
	if err := l.root.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if _, err := l.root.Lstat(p); err == nil {
		return fmt.Errorf("%w: %s", ErrExists, p)
	}

	var rnd [8]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		return err
	}
	tmp := path.Join(dir, TempPrefix+hex.EncodeToString(rnd[:]))
	f, err := l.root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer l.root.Remove(tmp)
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	linkErr := errors.New("hard links disabled")
	if !l.noLink {
		linkErr = l.root.Link(tmp, p)
	}
	switch {
	case linkErr == nil:
	case errors.Is(linkErr, fs.ErrExist):
		return fmt.Errorf("%w: %s", ErrExists, p)
	default:
		// Hard links are not supported (e.g. FAT, some network or FUSE file
		// systems): rename with an existence check.
		if _, err := l.root.Lstat(p); err == nil {
			return fmt.Errorf("%w: %s", ErrExists, p)
		}
		if err := l.root.Rename(tmp, p); err != nil {
			return err
		}
	}
	l.syncDir(dir)
	return nil
}

// syncDir makes the new directory entry durable where the platform allows
// it. Failures are ignored: the file content itself is already synced.
func (l *LocalFolder) syncDir(dir string) {
	if d, err := l.root.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
}

func (l *LocalFolder) Delete(_ context.Context, p string) error {
	if err := CheckPath(p); err != nil {
		return err
	}
	if err := l.root.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
