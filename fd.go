// Package gofd is a pure-Go port of the `fd` file finder. It exposes a friendly
// SDK for embedding fd-style search in Go programs, while cmd/fd provides a CLI
// compatible with the original tool.
//
// Typical SDK usage:
//
//	import gofd "github.com/startvibecoding/go-fd"
//
//	results, err := gofd.Find(context.Background(), gofd.Options{
//	    Pattern: "\\.go$",
//	    Paths:   []string{"."},
//	})
package gofd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"unicode"

	"github.com/startvibecoding/go-fd/pkg/exec"
	"github.com/startvibecoding/go-fd/pkg/filter"
	"github.com/startvibecoding/go-fd/pkg/finder"
	"github.com/startvibecoding/go-fd/pkg/format"
	"github.com/startvibecoding/go-fd/pkg/glob"
)

// Result re-exports finder.Result for SDK consumers.
type Result = finder.Result

// ExitCode re-exports finder.ExitCode for SDK/CLI consumers.
type ExitCode = finder.ExitCode

// Process exit codes.
const (
	ExitSuccess      = finder.ExitSuccess
	ExitGeneralError = finder.ExitGeneralError
)

// Options is the high-level, user-facing configuration for a search. Sensible
// fd defaults (smart case, respecting ignore files, skipping hidden entries)
// are applied automatically.
type Options struct {
	// Pattern is the primary search pattern. Empty matches everything.
	Pattern string
	// Exprs are additional patterns that must all match (fd's --and).
	Exprs []string
	// Paths are the search roots. Defaults to the current directory.
	Paths []string

	// Pattern interpretation.
	Glob         bool // treat patterns as globs
	FixedStrings bool // treat patterns as literal substrings
	Exact        bool // match the whole filename literally

	// Case handling. By default smart-case is used.
	CaseSensitive bool
	IgnoreCase    bool

	// Path matching.
	FullPath     bool // match against the full path, not just the file name
	AbsolutePath bool // emit absolute paths

	// Ignore handling.
	Hidden         bool // include hidden files
	NoIgnore       bool // disable all ignore files
	NoIgnoreVcs    bool // disable .gitignore only
	NoIgnoreParent bool // disable ignore files in parent directories
	NoGlobalIgnore bool // disable the global ignore file
	NoRequireGit   bool // respect gitignore even outside a git repo
	Unrestricted   bool // alias for NoIgnore + Hidden

	// Traversal.
	FollowLinks   bool
	OneFileSystem bool
	MaxDepth      int // 0 = unlimited
	MinDepth      int // 0 = none
	ExactDepth    int // 0 = unset; sets both min and max
	Prune         bool
	Threads       int // 0 = auto

	// Filters.
	Types         []string // f,d,l,x,e,s,p,c,b (or long names)
	Extensions    []string
	Exclude       []string
	Sizes         []string // e.g. "+1m", "-500k"
	ChangedWithin string
	ChangedBefore string
	Owner         string // [user|uid][:group|gid]
	IgnoreFiles   []string
	IgnoreContain []string

	// Output.
	NullSeparator  bool
	PathSeparator  string
	MaxResults     int // 0 = unlimited
	Format         string
	StripCwdPrefix *bool // nil = auto

	// Color: "auto", "always", "never".
	Color     string
	Hyperlink string // "auto", "always", "never"

	// Command execution (mutually exclusive with Format/output).
	Exec      []string // -x command template (terminated logically by caller)
	ExecBatch []string // -X command template
	BatchSize int

	ShowErrors bool
	Quiet      bool

	// ListDetails emulates --list-details (ls -l style listing).
	ListDetails bool
}

// Compile validates the options, builds the finder and resolves search paths.
func Compile(opts Options) (*finder.Finder, []string, error) {
	if opts.Unrestricted {
		opts.NoIgnore = true
		opts.Hidden = true
	}

	patterns := append([]string{}, opts.Exprs...)
	patterns = append(patterns, opts.Pattern)

	// Build pattern regex strings.
	regexStrs := make([]string, 0, len(patterns))
	for _, p := range patterns {
		s, err := buildPatternRegex(p, opts)
		if err != nil {
			return nil, nil, err
		}
		regexStrs = append(regexStrs, s)
	}

	// Smart-case detection.
	caseSensitive := !opts.IgnoreCase && (opts.CaseSensitive || anyHasUppercase(patterns))

	cfg, err := buildConfig(opts, caseSensitive)
	if err != nil {
		return nil, nil, err
	}

	// Compile regexes.
	compiled := make([]*regexp.Regexp, 0, len(regexStrs))
	for _, s := range regexStrs {
		flags := "(?s)"
		if !caseSensitive {
			flags = "(?is)"
		}
		re, err := regexp.Compile(flags + s)
		if err != nil {
			return nil, nil, fmt.Errorf("%w\n\nNote: You can search for literal substrings with FixedStrings or Exact options, or use Glob matching.", err)
		}
		compiled = append(compiled, re)
	}

	f, err := finder.New(cfg)
	if err != nil {
		return nil, nil, err
	}
	f.SetPatterns(compiled)

	paths, err := searchPaths(opts)
	if err != nil {
		return nil, nil, err
	}
	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("No valid search paths given.")
	}
	return f, paths, nil
}

// Find runs a search and returns matching paths sorted lexicographically.
func Find(ctx context.Context, opts Options) ([]string, error) {
	f, paths, err := Compile(opts)
	if err != nil {
		return nil, err
	}
	return f.Find(ctx, paths)
}

// Stream runs a search and streams results over a channel.
func Stream(ctx context.Context, opts Options) (<-chan Result, <-chan error, error) {
	f, paths, err := Compile(opts)
	if err != nil {
		return nil, nil, err
	}
	results, errs := f.Stream(ctx, paths)
	return results, errs, nil
}

func buildPatternRegex(pattern string, opts Options) (string, error) {
	switch {
	case opts.Glob && pattern != "":
		return glob.ToRegex(pattern, glob.Options{LiteralSeparator: true})
	case opts.Exact:
		return "^" + regexp.QuoteMeta(pattern) + "$", nil
	case opts.FixedStrings:
		return regexp.QuoteMeta(pattern), nil
	default:
		return pattern, nil
	}
}

func anyHasUppercase(patterns []string) bool {
	for _, p := range patterns {
		for _, r := range p {
			if unicode.IsUpper(r) {
				return true
			}
		}
	}
	return false
}

func buildConfig(opts Options, caseSensitive bool) (*finder.Config, error) {
	cfg := &finder.Config{
		CaseSensitive:        caseSensitive,
		IgnoreHidden:         !opts.Hidden,
		ReadFdignore:         !opts.NoIgnore,
		ReadVcsignore:        !(opts.NoIgnore || opts.NoIgnoreVcs),
		RequireGit:           !opts.NoRequireGit,
		ReadParentIgnore:     !opts.NoIgnoreParent,
		ReadGlobalIgnore:     !(opts.NoIgnore || opts.NoGlobalIgnore),
		FollowLinks:          opts.FollowLinks,
		OneFileSystem:        opts.OneFileSystem,
		NullSeparator:        opts.NullSeparator,
		Prune:                opts.Prune,
		Threads:              opts.Threads,
		Quiet:                opts.Quiet,
		ShowFilesystemErrors: opts.ShowErrors,
		BatchSize:            opts.BatchSize,
		ExcludePatterns:      opts.Exclude,
		IgnoreFiles:          opts.IgnoreFiles,
		IgnoreContain:        opts.IgnoreContain,
		Extensions:           opts.Extensions,
		AbsolutePath:         opts.AbsolutePath,
	}

	// Depth.
	if opts.ExactDepth > 0 {
		d := opts.ExactDepth
		cfg.MaxDepth = &d
		md := opts.ExactDepth
		cfg.MinDepth = &md
	} else {
		if opts.MaxDepth > 0 {
			d := opts.MaxDepth
			cfg.MaxDepth = &d
		}
		if opts.MinDepth > 0 {
			d := opts.MinDepth
			cfg.MinDepth = &d
		}
	}

	if opts.MaxResults > 0 {
		m := opts.MaxResults
		cfg.MaxResults = &m
	}

	// Path separator.
	cfg.PathSeparator = opts.PathSeparator
	if opts.PathSeparator != "" {
		cfg.ActualPathSeparator = opts.PathSeparator
	} else {
		cfg.ActualPathSeparator = string(filepath.Separator)
	}

	// Full path matching.
	if opts.FullPath {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("Could not determine current directory. This is required for --full-path.")
		}
		cfg.FullPathBase = cwd
	}

	// File types.
	if len(opts.Types) > 0 {
		ft, err := parseFileTypes(opts.Types)
		if err != nil {
			return nil, err
		}
		cfg.FileTypes = ft
	}

	// Size constraints.
	for _, s := range opts.Sizes {
		sf, err := filter.ParseSize(s)
		if err != nil {
			return nil, err
		}
		cfg.SizeConstraints = append(cfg.SizeConstraints, sf)
	}

	// Time constraints.
	if opts.ChangedWithin != "" {
		tf, err := filter.After(opts.ChangedWithin)
		if err != nil {
			return nil, err
		}
		cfg.TimeConstraints = append(cfg.TimeConstraints, tf)
	}
	if opts.ChangedBefore != "" {
		tf, err := filter.Before(opts.ChangedBefore)
		if err != nil {
			return nil, err
		}
		cfg.TimeConstraints = append(cfg.TimeConstraints, tf)
	}

	// Owner.
	if opts.Owner != "" {
		of, ok, err := filter.ParseOwner(opts.Owner)
		if err != nil {
			return nil, err
		}
		if ok {
			cfg.OwnerConstraint = &of
		}
	}

	// Format template.
	if opts.Format != "" {
		cfg.Format = format.Parse(opts.Format)
	}

	// Commands.
	if len(opts.Exec) > 0 {
		cs, err := exec.NewCommandSet([][]string{opts.Exec})
		if err != nil {
			return nil, err
		}
		cfg.Command = cs
	} else if len(opts.ExecBatch) > 0 {
		cs, err := exec.NewBatchCommandSet([][]string{opts.ExecBatch})
		if err != nil {
			return nil, err
		}
		cfg.Command = cs
	} else if opts.ListDetails {
		cs, err := exec.NewBatchCommandSet([][]string{lsCommand(opts.Color)})
		if err != nil {
			return nil, err
		}
		cfg.Command = cs
	}

	hasCommand := cfg.Command != nil

	// Colors.
	interactive := isTerminal(os.Stdout)
	cfg.InteractiveTerminal = interactive
	colored := resolveColor(opts.Color, interactive)
	cfg.Colored = colored
	if colored && cfg.Format == nil && cfg.Command == nil {
		cfg.LsColors = loadLsColors()
	}
	cfg.Hyperlink = resolveHyperlink(opts.Hyperlink, colored)

	// Strip cwd prefix.
	noSearchPaths := len(opts.Paths) == 0
	if opts.StripCwdPrefix != nil {
		cfg.StripCwdPrefix = noSearchPaths && *opts.StripCwdPrefix
	} else {
		cfg.StripCwdPrefix = noSearchPaths && !(opts.NullSeparator || hasCommand)
	}
	if opts.AbsolutePath {
		cfg.StripCwdPrefix = false
	}

	return cfg, nil
}

func parseFileTypes(values []string) (*finder.FileTypes, error) {
	ft := &finder.FileTypes{}
	for _, v := range values {
		switch v {
		case "f", "file":
			ft.Files = true
		case "d", "dir", "directory":
			ft.Directories = true
		case "l", "symlink":
			ft.Symlinks = true
		case "x", "executable":
			ft.ExecutablesOnly = true
			ft.Files = true
		case "e", "empty":
			ft.EmptyOnly = true
		case "b", "block-device":
			ft.BlockDevices = true
		case "c", "char-device":
			ft.CharDevices = true
		case "s", "socket":
			ft.Sockets = true
		case "p", "pipe":
			ft.Pipes = true
		default:
			return nil, fmt.Errorf("'%s' is not a valid file type", v)
		}
	}
	if ft.EmptyOnly && !(ft.Files || ft.Directories) {
		ft.Files = true
		ft.Directories = true
	}
	return ft, nil
}

func searchPaths(opts Options) ([]string, error) {
	paths := opts.Paths
	if len(paths) == 0 {
		cwd := "./"
		if !isExistingDir(cwd) {
			return nil, fmt.Errorf("Could not retrieve current directory (has it been deleted?).")
		}
		return []string{normalizePath(cwd, opts.AbsolutePath)}, nil
	}
	var out []string
	for _, p := range paths {
		if isExistingDir(p) {
			out = append(out, normalizePath(p, opts.AbsolutePath))
		} else {
			fmt.Fprintf(os.Stderr, "[fd error]: Search path '%s' is not a directory.\n", p)
		}
	}
	return out, nil
}

func normalizePath(path string, absolute bool) string {
	if absolute {
		abs, err := filepath.Abs(path)
		if err == nil {
			return abs
		}
	}
	if path == "." {
		return "./"
	}
	return path
}

func isExistingDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func lsCommand(color string) []string {
	colorArg := "--color=never"
	if color == "always" || color == "auto" {
		colorArg = "--color=always"
	}
	return []string{"ls", "-l", "-h", "-d", colorArg}
}

func resolveColor(when string, interactive bool) bool {
	switch when {
	case "always":
		return true
	case "never":
		return false
	default: // auto
		noColor := os.Getenv("NO_COLOR")
		return interactive && noColor == ""
	}
}

func resolveHyperlink(when string, colored bool) bool {
	switch when {
	case "always":
		return true
	case "never", "":
		return false
	default: // auto
		return colored
	}
}

func loadLsColors() *finder.LsColors {
	if env := os.Getenv("LS_COLORS"); env != "" {
		return finder.ParseLsColors(env)
	}
	return finder.ParseLsColors(finder.DefaultLsColors)
}
