package finder

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"
)

// Finder executes searches according to a Config.
type Finder struct {
	cfg *Config
}

// New constructs a Finder, compiling derived state (extension regexes) from the
// configuration. Patterns must already be compiled and assigned via
// SetPatterns or by the SDK helpers.
func New(cfg *Config) (*Finder, error) {
	if cfg.Threads < 1 {
		cfg.Threads = defaultThreads()
	}
	if cfg.ActualPathSeparator == "" {
		cfg.ActualPathSeparator = string(filepath.Separator)
	}
	// Compile extension regexes.
	for _, ext := range cfg.Extensions {
		e := ext
		for len(e) > 0 && e[0] == '.' {
			e = e[1:]
		}
		re, err := regexp.Compile("(?i)" + `.\.` + regexp.QuoteMeta(e) + `$`)
		if err != nil {
			return nil, err
		}
		cfg.extensionsRes = append(cfg.extensionsRes, re)
	}
	return &Finder{cfg: cfg}, nil
}

// SetPatterns assigns the compiled search patterns. All patterns must match for
// an entry to be reported.
func (f *Finder) SetPatterns(patterns []*regexp.Regexp) {
	f.cfg.patterns = patterns
}

// Config returns the underlying configuration.
func (f *Finder) Config() *Config { return f.cfg }

// matches applies every configured filter to an entry.
func (f *Finder) matches(e *DirEntry) bool {
	cfg := f.cfg

	if cfg.MinDepth != nil && e.depth < *cfg.MinDepth {
		return false
	}

	// Build the string the pattern is matched against.
	searchStr := f.searchString(e)
	for _, pat := range cfg.patterns {
		if !pat.MatchString(searchStr) {
			return false
		}
	}

	// Extension filter.
	if len(cfg.extensionsRes) > 0 {
		name := e.fileName()
		ok := false
		for _, re := range cfg.extensionsRes {
			if re.MatchString(name) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// File type filter.
	if cfg.FileTypes != nil && cfg.FileTypes.shouldIgnore(e) {
		return false
	}

	// Owner filter.
	if cfg.OwnerConstraint != nil {
		info, err := e.Info()
		if err != nil || !cfg.OwnerConstraint.Matches(info) {
			return false
		}
	}

	// Size filter (files only).
	if len(cfg.SizeConstraints) > 0 {
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
		size := uint64(info.Size())
		for _, sc := range cfg.SizeConstraints {
			if !sc.IsWithin(size) {
				return false
			}
		}
	}

	// Time filter.
	if len(cfg.TimeConstraints) > 0 {
		info, err := e.Info()
		if err != nil {
			return false
		}
		mod := info.ModTime()
		for _, tc := range cfg.TimeConstraints {
			if !tc.AppliesTo(mod) {
				return false
			}
		}
	}

	return true
}

func (f *Finder) searchString(e *DirEntry) string {
	if f.cfg.FullPathBase != "" {
		p := e.Path()
		if filepath.IsAbs(p) {
			return p
		}
		p = stripCurrentDir(p)
		return filepath.Join(f.cfg.FullPathBase, p)
	}
	return e.fileName()
}

// Result is a single matched entry surfaced through the SDK.
type Result struct {
	Path  string
	Entry *DirEntry
}

// Find runs the search and returns all matching paths, sorted lexicographically.
// It is a convenience wrapper around Stream that collects results.
func (f *Finder) Find(ctx context.Context, paths []string) ([]string, error) {
	results, errs := f.Stream(ctx, paths)
	var out []string
	for r := range results {
		out = append(out, r.Path)
	}
	// Drain errors (non-fatal); return the first if any was recorded.
	var firstErr error
	for e := range errs {
		if firstErr == nil {
			firstErr = e
		}
	}
	sort.Strings(out)
	return out, firstErr
}

// Stream runs the search and streams results over a channel. A second channel
// surfaces non-fatal traversal errors. Both channels are closed when the search
// completes. Cancel ctx to stop early.
func (f *Finder) Stream(ctx context.Context, paths []string) (<-chan Result, <-chan error) {
	ctx, cancel := context.WithCancel(ctx)
	out := make(chan Result, 256)
	errs := make(chan error, 64)
	raw := make(chan workerResult, 1024)

	f.runWalk(ctx, paths, raw)

	go func() {
		defer cancel()
		defer close(out)
		defer close(errs)
		count := 0
		for wr := range raw {
			if wr.err != nil {
				select {
				case errs <- wr.err:
				default:
				}
				continue
			}
			select {
			case out <- Result{Path: wr.entry.StrippedPath(f.cfg), Entry: wr.entry}:
			case <-ctx.Done():
				return
			}
			count++
			if f.cfg.MaxResults != nil && count >= *f.cfg.MaxResults {
				return
			}
		}
	}()

	return out, errs
}

// ExitCode mirrors fd's process exit codes.
type ExitCode int

const (
	// ExitSuccess indicates success.
	ExitSuccess ExitCode = 0
	// ExitGeneralError indicates a general error.
	ExitGeneralError ExitCode = 1
)

// Run executes the search and performs fd's CLI-style output: printing results
// (with buffering/sorting and colorization), running commands (-x/-X), or
// reporting match presence in quiet mode. It returns a process exit code.
func (f *Finder) Run(ctx context.Context, paths []string) ExitCode {
	cfg := f.cfg
	raw := make(chan workerResult, 1024)
	f.runWalk(ctx, paths, raw)

	if cfg.Quiet {
		return f.runQuiet(raw)
	}
	if cfg.Command != nil {
		return f.runExec(ctx, raw)
	}
	return f.runPrint(ctx, raw)
}

func (f *Finder) runQuiet(raw chan workerResult) ExitCode {
	found := false
	for wr := range raw {
		if wr.err != nil {
			f.reportErr(wr.err)
			continue
		}
		found = true
	}
	if found {
		return ExitSuccess
	}
	return ExitGeneralError
}

// runPrint buffers initial results to allow sorting when the search is fast,
// then streams. This mirrors fd's ReceiverBuffer behavior.
func (f *Finder) runPrint(ctx context.Context, raw chan workerResult) ExitCode {
	cfg := f.cfg
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	const maxBuffer = 1000
	maxBufferTime := cfg.MaxBufferTime
	if maxBufferTime == 0 {
		maxBufferTime = 100 * time.Millisecond
	}

	buffer := make([]*DirEntry, 0, maxBuffer)
	streaming := false
	count := 0
	exit := ExitSuccess
	deadline := time.NewTimer(maxBufferTime)
	defer deadline.Stop()

	flushBuffer := func() {
		sort.Slice(buffer, func(i, j int) bool { return buffer[i].path < buffer[j].path })
		for _, e := range buffer {
			printEntry(w, e, cfg)
		}
		buffer = buffer[:0]
		w.Flush()
		streaming = true
	}

	for {
		select {
		case <-deadline.C:
			if !streaming {
				flushBuffer()
			}
		case <-ctx.Done():
			if !streaming {
				flushBuffer()
			}
			return exit
		case wr, ok := <-raw:
			if !ok {
				if !streaming {
					flushBuffer()
				} else {
					w.Flush()
				}
				return exit
			}
			if wr.err != nil {
				f.reportErr(wr.err)
				continue
			}
			if streaming {
				printEntry(w, wr.entry, cfg)
			} else {
				buffer = append(buffer, wr.entry)
				if len(buffer) > maxBuffer {
					flushBuffer()
				}
			}
			count++
			if cfg.MaxResults != nil && count >= *cfg.MaxResults {
				if !streaming {
					flushBuffer()
				}
				w.Flush()
				return exit
			}
		}
	}
}

func (f *Finder) runExec(ctx context.Context, raw chan workerResult) ExitCode {
	cfg := f.cfg
	exit := ExitSuccess

	if cfg.Command.InBatchMode() {
		var paths []string
		for wr := range raw {
			if wr.err != nil {
				f.reportErr(wr.err)
				continue
			}
			paths = append(paths, wr.entry.StrippedPath(cfg))
		}
		if !cfg.Command.ExecuteBatch(paths, cfg.BatchSize, cfg.PathSeparator) {
			exit = ExitGeneralError
		}
		return exit
	}

	// One-by-one mode: process results with a worker pool.
	threads := cfg.Threads
	if threads < 1 {
		threads = 1
	}
	bufferOutput := threads > 1
	var wg sync.WaitGroup
	var mu sync.Mutex
	jobs := make(chan *DirEntry, 256)

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for e := range jobs {
				ok := cfg.Command.Execute(e.StrippedPath(cfg), cfg.PathSeparator, bufferOutput)
				if !ok {
					mu.Lock()
					exit = ExitGeneralError
					mu.Unlock()
				}
			}
		}()
	}

	for wr := range raw {
		if wr.err != nil {
			f.reportErr(wr.err)
			continue
		}
		jobs <- wr.entry
	}
	close(jobs)
	wg.Wait()
	return exit
}

func (f *Finder) reportErr(err error) {
	if f.cfg.ShowFilesystemErrors {
		fmt.Fprintf(os.Stderr, "[fd error]: %v\n", err)
	}
}

func defaultThreads() int {
	n := numCPU()
	if n > 64 {
		n = 64
	}
	if n < 1 {
		n = 1
	}
	return n
}
