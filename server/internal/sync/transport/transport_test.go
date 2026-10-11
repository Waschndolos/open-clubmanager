package transport

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// conformance runs the behaviour every Transport must have.
func conformance(t *testing.T, tr Transport) {
	ctx := context.Background()
	if err := tr.Create(ctx, "acl/00000001-0123456789abcdef.json", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := tr.Create(ctx, "acl/00000001-0123456789abcdef.json", []byte("two")); !errors.Is(err, ErrExists) {
		t.Fatalf("second Create: %v", err)
	}
	got, err := tr.Get(ctx, "acl/00000001-0123456789abcdef.json")
	if err != nil || string(got) != "one" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	if _, err := tr.Get(ctx, "acl/missing.json"); !errors.Is(err, ErrNotExist) {
		t.Fatalf("Get missing: %v", err)
	}

	for _, p := range []string{"club.json", "keys/finance/1/root.age", "segments/x/finance/000000000001.seg"} {
		if err := tr.Create(ctx, p, []byte(p)); err != nil {
			t.Fatal(err)
		}
	}
	paths := func(prefix string) []string {
		files, err := tr.List(ctx, prefix)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, f := range files {
			if strings.Contains(f.Path, TempPrefix) {
				t.Errorf("temp file %s left behind", f.Path)
			}
			out = append(out, f.Path)
		}
		slices.Sort(out)
		return out
	}
	if got := paths("acl/"); !slices.Equal(got, []string{"acl/00000001-0123456789abcdef.json"}) {
		t.Errorf("List(acl/) = %v", got)
	}
	if got := paths(""); len(got) != 4 {
		t.Errorf("List() = %v", got)
	}
	if got := paths("keys/"); !slices.Equal(got, []string{"keys/finance/1/root.age"}) {
		t.Errorf("List(keys/) = %v", got)
	}
	if got := paths("requests/"); len(got) != 0 {
		t.Errorf("List(requests/) = %v", got)
	}

	if err := tr.Delete(ctx, "club.json"); err != nil {
		t.Fatal(err)
	}
	if err := tr.Delete(ctx, "club.json"); err != nil {
		t.Errorf("deleting a missing file: %v", err)
	}
	if _, err := tr.Get(ctx, "club.json"); !errors.Is(err, ErrNotExist) {
		t.Errorf("Get after Delete: %v", err)
	}

	for _, bad := range []string{"../x", "/etc/passwd", "a/../../x", `a\b`, "", ".", "a//b"} {
		if err := tr.Create(ctx, bad, nil); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("Create(%q): %v", bad, err)
		}
		if _, err := tr.Get(ctx, bad); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("Get(%q): %v", bad, err)
		}
	}

	// Concurrent creators of one name: exactly one wins.
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tr.Create(ctx, "requests/race.json", []byte{byte(i)}) == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Errorf("%d concurrent creates succeeded", wins)
	}
}

func TestMemory(t *testing.T) { conformance(t, NewMemory()) }

func TestLocalFolder(t *testing.T) {
	dir := t.TempDir()
	lf, err := OpenLocalFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer lf.Close()
	conformance(t, lf)

	// Symbolic links must not lead out of the sync root.
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "secret"), []byte("x"), 0o600)
	if err := os.Symlink(outside, filepath.Join(dir, "escape")); err == nil {
		if _, err := lf.Get(context.Background(), "escape/secret"); err == nil {
			t.Error("read through a symlink out of the root")
		}
		if err := lf.Create(context.Background(), "escape/new", []byte("x")); err == nil {
			t.Error("wrote through a symlink out of the root")
		}
	}
}

func TestLocalFolderRenameFallback(t *testing.T) {
	lf, err := OpenLocalFolder(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer lf.Close()
	lf.noLink = true
	ctx := context.Background()
	if err := lf.Create(ctx, "a/b.json", []byte("1")); err != nil {
		t.Fatal(err)
	}
	if err := lf.Create(ctx, "a/b.json", []byte("2")); !errors.Is(err, ErrExists) {
		t.Fatalf("second Create: %v", err)
	}
	got, _ := lf.Get(ctx, "a/b.json")
	files, _ := lf.List(ctx, "a/")
	if string(got) != "1" || len(files) != 1 {
		t.Errorf("content %q, files %v", got, files)
	}
}

func TestLocalFolderMissingDir(t *testing.T) {
	if _, err := OpenLocalFolder(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("opened a missing folder")
	}
}
