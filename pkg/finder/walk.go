package finder

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/startvibecoding/go-fd/pkg/ignore"
)

// workerResult carries either a matched entry or a traversal error.
type workerResult struct {
	entry *DirEntry
	err   error
}

// walker performs the parallel, ignore-aware traversal.
type walker struct {
	cfg     *Config
	results chan workerResult
	ctx     context.Context
	match   func(*DirEntry) bool

	excludes     *ignore.Gitignore
	customIgnore []*ignore.Gitignore
	globalIgnore *ignore.Gitignore
}

type dirJob struct {
	path        string
	depth       int
	ignoreStack []*ignore.Gitignore
}

// runWalk traverses the given search paths, applying filters, and emits matches
// to the results channel. The channel is closed when traversal finishes.
func (f *Finder) runWalk(ctx context.Context, paths []string, results chan workerResult) {
	w := &walker{cfg: f.cfg, results: results, ctx: ctx, match: f.matches}

	// Exclude patterns become a gitignore set rooted at "." (matched against
	// paths relative to each search root).
	if len(f.cfg.ExcludePatterns) > 0 {
		w.excludes = ignore.NewFromLines("", f.cfg.ExcludePatterns)
	}

	// Custom ignore files.
	for _, p := range f.cfg.IgnoreFiles {
		if gi, err := ignore.NewFromFile(p, ""); err == nil {
			w.customIgnore = append(w.customIgnore, gi)
		}
	}

	// Global ignore file.
	if f.cfg.ReadGlobalIgnore {
		if p := globalIgnorePath(); p != "" {
			if gi, err := ignore.NewFromFile(p, ""); err == nil {
				w.globalIgnore = gi
			}
		}
	}

	threads := f.cfg.Threads
	if threads < 1 {
		threads = 1
	}
	sem := make(chan struct{}, threads)
	var wg sync.WaitGroup

	for _, root := range paths {
		stack := w.initialStack(root)
		wg.Add(1)
		go w.process(dirJob{path: root, depth: 0, ignoreStack: stack}, sem, &wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()
}

// initialStack builds the ignore stack for a search root, optionally pulling in
// ignore files from parent directories.
func (w *walker) initialStack(root string) []*ignore.Gitignore {
	var stack []*ignore.Gitignore
	if w.globalIgnore != nil {
		stack = append(stack, w.globalIgnore)
	}
	stack = append(stack, w.customIgnore...)

	if w.cfg.ReadParentIgnore {
		abs, err := filepath.Abs(root)
		if err == nil {
			var parents []string
			dir := filepath.Dir(abs)
			for {
				parents = append(parents, dir)
				parent := filepath.Dir(dir)
				if parent == dir {
					break
				}
				dir = parent
			}
			// Outermost first.
			for i := len(parents) - 1; i >= 0; i-- {
				stack = append(stack, w.loadIgnores(parents[i])...)
			}
		}
	}
	return stack
}

// loadIgnores reads ignore files present in dir and returns matchers anchored
// to dir.
func (w *walker) loadIgnores(dir string) []*ignore.Gitignore {
	var out []*ignore.Gitignore
	add := func(name string) {
		p := filepath.Join(dir, name)
		if gi, err := ignore.NewFromFile(p, dir); err == nil && !gi.Empty() {
			out = append(out, gi)
		}
	}
	if w.cfg.ReadVcsignore {
		add(".gitignore")
		// .git/info/exclude
		p := filepath.Join(dir, ".git", "info", "exclude")
		if gi, err := ignore.NewFromFile(p, dir); err == nil && !gi.Empty() {
			out = append(out, gi)
		}
	}
	if w.cfg.ReadFdignore {
		add(".ignore")
		add(".fdignore")
	}
	return out
}

func (w *walker) process(job dirJob, sem chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	select {
	case <-w.ctx.Done():
		return
	case sem <- struct{}{}:
	}

	subdirs, stack := w.processDir(job)
	<-sem

	for _, sd := range subdirs {
		wg.Add(1)
		go w.process(dirJob{path: sd, depth: job.depth + 1, ignoreStack: stack}, sem, wg)
	}
}

// processDir reads a directory, emits matching children, and returns the list
// of subdirectories to descend into along with the ignore stack for them.
func (w *walker) processDir(job dirJob) ([]string, []*ignore.Gitignore) {
	cfg := w.cfg

	// Respect max depth: don't read beyond it.
	if cfg.MaxDepth != nil && job.depth >= *cfg.MaxDepth {
		// We still may emit the directory entry itself; that was handled by
		// the parent. Just stop descending.
		return nil, job.ignoreStack
	}

	entries, err := os.ReadDir(job.path)
	if err != nil {
		w.emitErr(err)
		return nil, job.ignoreStack
	}

	// Load ignore files in this directory and extend the stack.
	stack := job.ignoreStack
	if newIgnores := w.loadIgnores(job.path); len(newIgnores) > 0 {
		stack = append(append([]*ignore.Gitignore{}, job.ignoreStack...), newIgnores...)
	}

	var subdirs []string
	for _, de := range entries {
		select {
		case <-w.ctx.Done():
			return nil, stack
		default:
		}

		name := de.Name()
		childPath := joinPath(job.path, name)
		childDepth := job.depth + 1

		// Hidden files.
		if cfg.IgnoreHidden && strings.HasPrefix(name, ".") {
			continue
		}

		isDir := de.IsDir()
		// Resolve symlinks to directories when following.
		if de.Type()&os.ModeSymlink != 0 && cfg.FollowLinks {
			if info, err := os.Stat(childPath); err == nil {
				isDir = info.IsDir()
			}
		}

		// Ignore rules.
		if w.isIgnored(childPath, isDir, stack) {
			continue
		}
		// Exclude patterns.
		if w.excludes != nil && w.matchExclude(childPath, isDir) {
			continue
		}

		entry := newEntry(childPath, childDepth, de)

		matched := w.match(entry)

		if matched {
			select {
			case w.results <- workerResult{entry: entry}:
			case <-w.ctx.Done():
				return nil, stack
			}
		}

		if isDir {
			// ignore-contain: skip directories containing a named entry.
			if len(cfg.IgnoreContain) > 0 && containsAny(childPath, cfg.IgnoreContain) {
				continue
			}
			// Pruning: if the directory matched and prune is on, don't descend.
			if cfg.Prune && matched {
				continue
			}
			// Don't follow symlinked dirs unless configured.
			if de.Type()&os.ModeSymlink != 0 && !cfg.FollowLinks {
				continue
			}
			subdirs = append(subdirs, childPath)
		}
	}
	return subdirs, stack
}

func (w *walker) emitErr(err error) {
	select {
	case w.results <- workerResult{err: err}:
	case <-w.ctx.Done():
	}
}

// isIgnored evaluates the ignore stack for a child path. Deeper (later) and
// later-matching patterns take precedence; a whitelist overrides an ignore.
func (w *walker) isIgnored(path string, isDir bool, stack []*ignore.Gitignore) bool {
	decision := ignore.None
	for _, gi := range stack {
		rel := relTo(gi.Root(), path)
		if rel == "" {
			continue
		}
		switch gi.Matches(rel, isDir) {
		case ignore.Ignore:
			decision = ignore.Ignore
		case ignore.Whitelist:
			decision = ignore.Whitelist
		}
	}
	return decision == ignore.Ignore
}

func (w *walker) matchExclude(path string, isDir bool) bool {
	// Match against both the full path and the basename.
	if w.excludes.Matches(path, isDir) == ignore.Ignore {
		return true
	}
	return w.excludes.Matches(filepath.Base(path), isDir) == ignore.Ignore
}

func containsAny(dir string, names []string) bool {
	for _, n := range names {
		if _, err := os.Lstat(filepath.Join(dir, n)); err == nil {
			return true
		}
	}
	return false
}

// relTo returns path relative to root using forward slashes. If root is empty,
// the path is returned cleaned. Returns "" if path is not under root.
func relTo(root, path string) string {
	if root == "" {
		return filepath.ToSlash(stripCurrentDir(path))
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	return filepath.ToSlash(rel)
}

func joinPath(dir, name string) string {
	if dir == "" {
		return name
	}
	// Preserve the "./" prefix style used for the current directory.
	if strings.HasSuffix(dir, string(os.PathSeparator)) {
		return dir + name
	}
	return dir + string(os.PathSeparator) + name
}

func globalIgnorePath() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "fd", "ignore")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "fd", "ignore")
}
