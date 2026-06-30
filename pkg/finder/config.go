// Package finder is the core engine of go-fd. It performs a parallel,
// gitignore-aware filesystem walk and applies fd's filters, mirroring the
// behavior of the original Rust implementation. It also exposes the SDK entry
// points (Finder, Config, Find, FindStream).
package finder

import (
	"regexp"
	"time"

	"github.com/startvibecoding/go-fd/pkg/exec"
	"github.com/startvibecoding/go-fd/pkg/filter"
	"github.com/startvibecoding/go-fd/pkg/format"
)

// FileType enumerates the entry kinds that can be filtered with --type.
type FileType int

const (
	// TypeFile matches regular files.
	TypeFile FileType = iota
	// TypeDirectory matches directories.
	TypeDirectory
	// TypeSymlink matches symbolic links.
	TypeSymlink
	// TypeBlockDevice matches block devices.
	TypeBlockDevice
	// TypeCharDevice matches character devices.
	TypeCharDevice
	// TypeExecutable matches executable files.
	TypeExecutable
	// TypeEmpty matches empty files or directories.
	TypeEmpty
	// TypeSocket matches sockets.
	TypeSocket
	// TypePipe matches named pipes (FIFOs).
	TypePipe
)

// FileTypes describes which entry kinds should be shown.
type FileTypes struct {
	Files           bool
	Directories     bool
	Symlinks        bool
	BlockDevices    bool
	CharDevices     bool
	Sockets         bool
	Pipes           bool
	ExecutablesOnly bool
	EmptyOnly       bool
}

// Config holds every option controlling a search. The zero value is not valid;
// build it through the CLI layer or the SDK helpers which apply defaults.
type Config struct {
	// CaseSensitive controls case sensitivity of the pattern match.
	CaseSensitive bool

	// FullPathBase, when non-empty, makes patterns match against the absolute
	// path (rooted at this directory) instead of the file name only.
	FullPathBase string

	// IgnoreHidden skips dotfiles/dotdirs when true.
	IgnoreHidden bool

	// ReadFdignore respects .fdignore and .ignore files.
	ReadFdignore bool
	// ReadParentIgnore respects ignore files in parent directories.
	ReadParentIgnore bool
	// ReadVcsignore respects .gitignore files.
	ReadVcsignore bool
	// RequireGit only respects gitignore inside a git repository.
	RequireGit bool
	// ReadGlobalIgnore respects the global ignore file.
	ReadGlobalIgnore bool

	// FollowLinks traverses symlinked directories.
	FollowLinks bool
	// OneFileSystem prevents descending into other filesystems.
	OneFileSystem bool

	// NullSeparator separates results with NUL instead of newline.
	NullSeparator bool

	// MaxDepth/MinDepth limit traversal depth. nil means unbounded.
	MaxDepth *int
	MinDepth *int

	// Prune stops descending into matching directories.
	Prune bool

	// Threads is the worker count (defaults applied by the caller).
	Threads int

	// Quiet suppresses output; the search reports only whether a match exists.
	Quiet bool

	// MaxBufferTime is the duration to buffer results for sorting before
	// streaming. Zero uses the default.
	MaxBufferTime time.Duration

	// Colored enables LS_COLORS-based colorization.
	Colored  bool
	LsColors *LsColors

	// Hyperlink wraps each path in an OSC 8 terminal hyperlink.
	Hyperlink bool

	// InteractiveTerminal reports whether stdout is a TTY.
	InteractiveTerminal bool

	// FileTypes restricts results by entry kind (nil = all).
	FileTypes *FileTypes

	// Extensions restricts results to matching file extensions (nil = all).
	Extensions []string

	// Format renders results with a template (nil = plain path).
	Format *format.Template

	// Command runs a command per result / batch (nil = print).
	Command *exec.CommandSet
	// BatchSize bounds arguments per batch command (0 = unlimited).
	BatchSize int

	// ExcludePatterns are gitignore-style globs that exclude entries.
	ExcludePatterns []string

	// IgnoreFiles are custom ignore files in gitignore format.
	IgnoreFiles []string

	// SizeConstraints restrict file sizes.
	SizeConstraints []filter.SizeFilter
	// TimeConstraints restrict modification times.
	TimeConstraints []filter.TimeFilter
	// OwnerConstraint restricts ownership (unix only). nil = no constraint.
	OwnerConstraint *filter.OwnerFilter

	// ShowFilesystemErrors prints traversal errors to stderr.
	ShowFilesystemErrors bool

	// PathSeparator overrides the path separator in printed output.
	PathSeparator string
	// ActualPathSeparator is the effective separator (default or override).
	ActualPathSeparator string

	// MaxResults limits the number of results. nil = unlimited.
	MaxResults *int

	// StripCwdPrefix removes the leading "./" of relative results.
	StripCwdPrefix bool

	// IgnoreContain skips directories that contain a named entry.
	IgnoreContain []string

	// AbsolutePath makes results absolute.
	AbsolutePath bool

	// compiled fields (populated by the finder)
	extensionsRes []*regexp.Regexp
}

// IsPrinting reports whether results are being printed (no command set).
func (c *Config) IsPrinting() bool { return c.Command == nil }
