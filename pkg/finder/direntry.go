package finder

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// DirEntry represents a single discovered filesystem entry. It lazily resolves
// metadata to avoid unnecessary syscalls.
type DirEntry struct {
	path  string
	depth int

	// brokenSymlink marks entries that are dangling symlinks.
	brokenSymlink bool

	dirEntry fs.DirEntry

	once     sync.Once
	info     os.FileInfo
	infoErr  error
	cachedFT *fs.FileMode
}

// newEntry constructs a DirEntry from a walk hit.
func newEntry(path string, depth int, de fs.DirEntry) *DirEntry {
	return &DirEntry{path: path, depth: depth, dirEntry: de}
}

// Path returns the entry's path as discovered.
func (e *DirEntry) Path() string { return e.path }

// Depth returns the traversal depth (root children are depth 1).
func (e *DirEntry) Depth() int { return e.depth }

// Info returns the (lazily-loaded) file info, using lstat semantics.
func (e *DirEntry) Info() (os.FileInfo, error) {
	e.once.Do(func() {
		if e.dirEntry != nil {
			e.info, e.infoErr = e.dirEntry.Info()
			return
		}
		e.info, e.infoErr = os.Lstat(e.path)
	})
	return e.info, e.infoErr
}

// Type returns the file mode type bits, or 0 if unavailable.
func (e *DirEntry) Type() fs.FileMode {
	if e.dirEntry != nil {
		return e.dirEntry.Type()
	}
	info, err := e.Info()
	if err != nil {
		return 0
	}
	return info.Mode().Type()
}

// IsDir reports whether the entry is a directory.
func (e *DirEntry) IsDir() bool {
	return e.Type().IsDir()
}

// StrippedPath returns the path as it should be displayed to the user.
func (e *DirEntry) StrippedPath(cfg *Config) string {
	if cfg.StripCwdPrefix {
		return stripCurrentDir(e.path)
	}
	return e.path
}

// stripCurrentDir removes a leading "./" from a path.
func stripCurrentDir(path string) string {
	if path == "." {
		return path
	}
	if len(path) >= 2 && path[0] == '.' && os.IsPathSeparator(path[1]) {
		return path[2:]
	}
	return path
}

// fileName returns the final path component.
func (e *DirEntry) fileName() string {
	return filepath.Base(e.path)
}
