package finder

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// printEntry writes a single entry to w according to the configuration.
func printEntry(w *bufio.Writer, e *DirEntry, cfg *Config) error {
	hyperlink := false
	if cfg.Hyperlink {
		if url, ok := pathURL(e.Path()); ok {
			w.WriteString("\x1B]8;;")
			w.WriteString(url)
			w.WriteString("\x1B\\")
			hyperlink = true
		}
	}

	switch {
	case cfg.Format != nil:
		printFormat(w, e, cfg)
	case cfg.LsColors != nil:
		printColorized(w, e, cfg)
	default:
		printPlain(w, e, cfg)
	}

	if hyperlink {
		w.WriteString("\x1B]8;;\x1B\\")
	}

	if cfg.NullSeparator {
		return w.WriteByte(0)
	}
	return w.WriteByte('\n')
}

func replacePathSeparator(path, sep string) string {
	if sep == "" {
		return path
	}
	path = strings.ReplaceAll(path, "/", sep)
	return strings.ReplaceAll(path, `\`, sep)
}

func printFormat(w *bufio.Writer, e *DirEntry, cfg *Config) {
	out := cfg.Format.Generate(e.StrippedPath(cfg), cfg.PathSeparator)
	w.WriteString(out)
}

func printPlain(w *bufio.Writer, e *DirEntry, cfg *Config) {
	path := e.StrippedPath(cfg)
	if cfg.PathSeparator != "" {
		path = replacePathSeparator(path, cfg.PathSeparator)
	}
	w.WriteString(path)
	printTrailingSlash(w, e, cfg, "")
}

func printColorized(w *bufio.Writer, e *DirEntry, cfg *Config) {
	path := e.StrippedPath(cfg)
	lc := cfg.LsColors

	// Split between parent directory and final component.
	offset := 0
	if idx := strings.LastIndexByte(path, filepath.Separator); idx >= 0 {
		offset = idx + 1
	}

	if offset > 0 {
		parent := path[:offset]
		if cfg.PathSeparator != "" {
			parent = replacePathSeparator(parent, cfg.PathSeparator)
		}
		w.WriteString(paint(lc.dirStyle(), parent))
	}

	last := path[offset:]
	if cfg.PathSeparator != "" {
		last = replacePathSeparator(last, cfg.PathSeparator)
	}
	w.WriteString(paint(lc.styleFor(e), last))

	printTrailingSlash(w, e, cfg, lc.dirStyle())
}

func printTrailingSlash(w *bufio.Writer, e *DirEntry, cfg *Config, code string) {
	if e.IsDir() {
		w.WriteString(paint(code, cfg.ActualPathSeparator))
	}
}

// pathURL builds a file:// URL for hyperlink output.
func pathURL(path string) (string, bool) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	var b strings.Builder
	b.WriteString("file://")
	b.WriteString(hostname())
	for i := 0; i < len(abs); i++ {
		c := abs[i]
		switch {
		case (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			c == '/' || c == ':' || c == '-' || c == '.' || c == '_' || c == '~':
			b.WriteByte(c)
		default:
			const hex = "0123456789ABCDEF"
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0xf])
		}
	}
	return b.String(), true
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}
