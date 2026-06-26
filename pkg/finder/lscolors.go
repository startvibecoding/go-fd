package finder

import (
	"io/fs"
	"strings"
)

// DefaultLsColors is the molokai-derived default palette used by fd when
// LS_COLORS is not present in the environment.
const DefaultLsColors = "ow=0:or=0;38;5;16;48;5;203:no=0:ex=1;38;5;203:cd=0;38;5;203;48;5;236:mi=0;38;5;16;48;5;203:*~=0;38;5;243:st=0:pi=0;38;5;16;48;5;81:fi=0:di=0;38;5;81:so=0;38;5;16;48;5;203:bd=0;38;5;81;48;5;236:tw=0:ln=0;38;5;203:*.md=0;38;5;185:*.go=0;38;5;48:*.rs=0;38;5;48:*.py=0;38;5;48:*.c=0;38;5;48:*.h=0;38;5;48:*.cpp=0;38;5;48:*.js=0;38;5;48:*.ts=0;38;5;48:*.json=0;38;5;149:*.yml=0;38;5;149:*.yaml=0;38;5;149:*.toml=0;38;5;149:*.zip=4;38;5;203:*.tar=4;38;5;203:*.gz=4;38;5;203:*.png=0;38;5;208:*.jpg=0;38;5;208:*.mp3=0;38;5;208:*.mp4=0;38;5;208"

// LsColors maps filesystem indicators and filename patterns to ANSI styles.
type LsColors struct {
	indicators map[string]string // e.g. "di", "ln", "ex"
	extensions map[string]string // lowercase extension -> code, key includes leading dot
	names      map[string]string // exact filename match
}

// ParseLsColors parses an LS_COLORS-format string.
func ParseLsColors(s string) *LsColors {
	lc := &LsColors{
		indicators: map[string]string{},
		extensions: map[string]string{},
		names:      map[string]string{},
	}
	for _, entry := range strings.Split(s, ":") {
		if entry == "" {
			continue
		}
		eq := strings.IndexByte(entry, '=')
		if eq < 0 {
			continue
		}
		key := entry[:eq]
		val := entry[eq+1:]
		switch {
		case strings.HasPrefix(key, "*."):
			lc.extensions[strings.ToLower(key[1:])] = val
		case strings.HasPrefix(key, "*"):
			lc.names[key[1:]] = val
		default:
			lc.indicators[key] = val
		}
	}
	return lc
}

// styleFor returns the ANSI code (without escape framing) for an entry.
func (lc *LsColors) styleFor(e *DirEntry) string {
	mode := e.Type()
	switch {
	case mode&fs.ModeSymlink != 0:
		if c, ok := lc.indicators["ln"]; ok {
			return c
		}
	case mode.IsDir():
		if c, ok := lc.indicators["di"]; ok {
			return c
		}
	case mode&fs.ModeNamedPipe != 0:
		return lc.indicators["pi"]
	case mode&fs.ModeSocket != 0:
		return lc.indicators["so"]
	case mode&fs.ModeCharDevice != 0:
		return lc.indicators["cd"]
	case mode&fs.ModeDevice != 0:
		return lc.indicators["bd"]
	}

	name := e.fileName()
	if c, ok := lc.names[name]; ok {
		return c
	}
	lower := strings.ToLower(name)
	if dot := strings.LastIndexByte(lower, '.'); dot >= 0 {
		if c, ok := lc.extensions[lower[dot:]]; ok {
			return c
		}
	}
	// Executable files.
	if mode.IsRegular() {
		if info, err := e.Info(); err == nil && info.Mode().Perm()&0111 != 0 {
			if c, ok := lc.indicators["ex"]; ok {
				return c
			}
		}
		return lc.indicators["fi"]
	}
	return ""
}

// dirStyle returns the style code used for parent directory components.
func (lc *LsColors) dirStyle() string {
	return lc.indicators["di"]
}

// paint wraps text in ANSI escape codes for the given code; if code is empty
// the text is returned unchanged.
func paint(code, text string) string {
	if code == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}
