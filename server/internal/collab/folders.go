package collab

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
)

var (
	// ErrFolderMissing is returned for a path that does not exist or is not
	// a folder.
	ErrFolderMissing = errors.New("collab: folder does not exist")
	// ErrNotWritable is returned for a folder this process cannot write to.
	ErrNotWritable = errors.New("collab: folder is not writable")
	// ErrNoClubInFolder is returned when joining from a folder without a
	// club (or with several, so the user has to pick one).
	ErrNoClubInFolder = errors.New("collab: folder does not contain a club")
)

// Folder kinds, shown to the user after choosing a folder (UX F2 step 1).
const (
	KindSync    = "sync"    // inside a folder of a known sync client
	KindNetwork = "network" // network drive or mounted share
	KindLocal   = "local"   // local folder that does not seem to be synced
)

// ClubRef names a club found in a folder.
type ClubRef struct {
	Path string
	Name string
}

// FolderInfo describes a folder the user chose or the app found.
type FolderInfo struct {
	Path     string
	Writable bool
	Kind     string
	// Provider is the sync client, e.g. "Nextcloud", for KindSync.
	Provider string
	// Club is set if the folder itself holds a club (club.json).
	Club *ClubRef
	// ClubsInside lists clubs in direct subfolders, so the user can pick
	// the shared parent folder when joining.
	ClubsInside []ClubRef
}

// providers maps folder name prefixes of sync clients to display names.
var providers = []struct{ prefix, name string }{
	{"nextcloud", "Nextcloud"},
	{"owncloud", "ownCloud"},
	{"onedrive", "OneDrive"},
	{"dropbox", "Dropbox"},
	{"google drive", "Google Drive"},
	{"googledrive", "Google Drive"},
	{"my drive", "Google Drive"},
	{"seafile", "Seafile"},
	{"pcloud", "pCloud"},
	{"icloud", "iCloud Drive"},
	{"magentacloud", "MagentaCLOUD"},
}

func providerOf(path string) string {
	for _, part := range strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' }) {
		p := strings.ToLower(part)
		for _, pr := range providers {
			if strings.HasPrefix(p, pr.prefix) {
				return pr.name
			}
		}
	}
	return ""
}

func isNetworkPath(path string) bool {
	switch {
	case strings.HasPrefix(path, `\\`), strings.HasPrefix(path, "//"):
		return true
	case runtime.GOOS == "darwin" && strings.HasPrefix(path, "/Volumes/"):
		return true
	case runtime.GOOS == "linux":
		for _, p := range []string{"/mnt/", "/media/", "/net/"} {
			if strings.HasPrefix(path, p) {
				return true
			}
		}
		return strings.Contains(path, "/gvfs/")
	}
	return false
}

// expandHome replaces a leading "~" with the home directory.
func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

// CheckFolder inspects a folder chosen or typed by the user (UX F2 step 1).
func CheckFolder(path string) (*FolderInfo, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, ErrFolderMissing
	}
	path = filepath.Clean(expandHome(path))
	st, err := os.Stat(path)
	if err != nil || !st.IsDir() {
		return nil, ErrFolderMissing
	}
	info := &FolderInfo{Path: path, Kind: KindLocal}
	if p := providerOf(path); p != "" {
		info.Kind, info.Provider = KindSync, p
	} else if isNetworkPath(path) {
		info.Kind = KindNetwork
	}
	if f, err := os.CreateTemp(path, ".ocm-write-test-*"); err == nil {
		name := f.Name()
		_ = f.Close()
		_ = os.Remove(name)
		info.Writable = true
	}
	info.Club = readClub(path)
	if info.Club == nil {
		entries, _ := os.ReadDir(path)
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			if c := readClub(filepath.Join(path, e.Name())); c != nil {
				info.ClubsInside = append(info.ClubsInside, *c)
			}
		}
	}
	return info, nil
}

// readClub returns the club whose valid, root-signed club.json is in dir.
func readClub(dir string) *ClubRef {
	raw, err := os.ReadFile(filepath.Join(dir, layout.Genesis))
	if err != nil {
		return nil
	}
	sig, err := os.ReadFile(filepath.Join(dir, layout.GenesisSig))
	if err != nil {
		return nil
	}
	g, err := acl.ParseGenesis(raw, sig)
	if err != nil {
		return nil
	}
	return &ClubRef{Path: dir, Name: g.Name}
}

// DetectFolders returns folders of sync clients found in the home directory,
// as suggestions for the storage location (UX F2 step 1, F4 step 2).
func DetectFolders() []FolderInfo {
	home, err := os.UserHomeDir()
	if err != nil {
		return []FolderInfo{}
	}
	roots := []string{home, filepath.Join(home, "Library", "CloudStorage")}
	seen := map[string]bool{}
	out := []FolderInfo{}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			p := filepath.Join(root, e.Name())
			if providerOf(e.Name()) == "" || seen[p] {
				continue
			}
			seen[p] = true
			if info, err := CheckFolder(p); err == nil {
				out = append(out, *info)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
