// Package ignore implements gitignore-style pattern matching used by fd to
// honor .gitignore, .ignore, .fdignore and custom ignore files. It supports
// nested ignore files accumulated while descending a directory tree.
package ignore

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// Match is the result of testing a path against a set of ignore patterns.
type Match int

const (
	// None means no pattern matched.
	None Match = iota
	// Ignore means the path should be ignored.
	Ignore
	// Whitelist means the path was explicitly un-ignored (negated pattern).
	Whitelist
)

// pattern is a single compiled gitignore rule.
type pattern struct {
	re       *regexp.Regexp
	negated  bool
	dirOnly  bool
	original string
}

// Gitignore is a set of patterns rooted at a particular base directory.
type Gitignore struct {
	// root is the directory the patterns are anchored to (the directory
	// containing the ignore file).
	root     string
	patterns []pattern
}

// Root returns the base directory the patterns are anchored to.
func (g *Gitignore) Root() string { return g.root }

// NewFromFile parses an ignore file located at path. The patterns are anchored
// to the directory containing the file.
func NewFromFile(path, root string) (*Gitignore, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	g := &Gitignore{root: root}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		g.addLine(sc.Text())
	}
	return g, sc.Err()
}

// NewFromLines builds a Gitignore from raw lines anchored at root.
func NewFromLines(root string, lines []string) *Gitignore {
	g := &Gitignore{root: root}
	for _, l := range lines {
		g.addLine(l)
	}
	return g
}

func (g *Gitignore) addLine(line string) {
	p, ok := parsePattern(line)
	if ok {
		g.patterns = append(g.patterns, p)
	}
}

// parsePattern converts a single gitignore line into a compiled pattern.
func parsePattern(line string) (pattern, bool) {
	orig := line
	// Strip trailing CR.
	line = strings.TrimRight(line, "\r")

	if line == "" {
		return pattern{}, false
	}
	// Comments.
	if strings.HasPrefix(line, "#") {
		return pattern{}, false
	}

	// Trailing spaces are ignored unless escaped with backslash.
	line = trimTrailingSpaces(line)
	if line == "" {
		return pattern{}, false
	}

	negated := false
	if strings.HasPrefix(line, "!") {
		negated = true
		line = line[1:]
	} else if strings.HasPrefix(line, "\\!") || strings.HasPrefix(line, "\\#") {
		line = line[1:]
	}

	dirOnly := false
	if strings.HasSuffix(line, "/") {
		dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}
	if line == "" {
		return pattern{}, false
	}

	// Determine anchoring: a slash anywhere except a trailing one means the
	// pattern is anchored relative to the ignore file location.
	anchored := strings.HasPrefix(line, "/")
	if anchored {
		line = strings.TrimPrefix(line, "/")
	} else if strings.Contains(line, "/") {
		anchored = true
	}

	reStr := buildRegex(line, anchored)
	re, err := regexp.Compile(reStr)
	if err != nil {
		return pattern{}, false
	}
	return pattern{re: re, negated: negated, dirOnly: dirOnly, original: orig}, true
}

func trimTrailingSpaces(s string) string {
	// Remove trailing spaces unless the last space is escaped.
	i := len(s)
	for i > 0 && s[i-1] == ' ' {
		// count preceding backslashes
		bs := 0
		j := i - 2
		for j >= 0 && s[j] == '\\' {
			bs++
			j--
		}
		if bs%2 == 1 {
			break // escaped space, keep
		}
		i--
	}
	return s[:i]
}

// buildRegex translates a gitignore glob into a regex matched against a path
// relative to the ignore root (using forward slashes, no leading slash).
func buildRegex(glob string, anchored bool) string {
	var b strings.Builder
	b.WriteString("(?s)^")
	if !anchored {
		// Match at any directory depth: allow an optional leading path.
		b.WriteString("(?:.*/)?")
	}

	runes := []rune(glob)
	n := len(runes)
	i := 0
	for i < n {
		c := runes[i]
		switch c {
		case '*':
			if i+1 < n && runes[i+1] == '*' {
				j := i
				for j < n && runes[j] == '*' {
					j++
				}
				prevSep := i == 0 || runes[i-1] == '/'
				nextSepOrEnd := j >= n || runes[j] == '/'
				if prevSep && nextSepOrEnd {
					if j < n && runes[j] == '/' {
						b.WriteString("(?:.*/)?")
						i = j + 1
						continue
					}
					b.WriteString(".*")
					i = j
					continue
				}
				b.WriteString("[^/]*")
				i = j
				continue
			}
			b.WriteString("[^/]*")
			i++
		case '?':
			b.WriteString("[^/]")
			i++
		case '[':
			end, cls := parseClass(runes, i)
			b.WriteString(cls)
			i = end
		case '\\':
			if i+1 < n {
				b.WriteString(regexp.QuoteMeta(string(runes[i+1])))
				i += 2
			} else {
				b.WriteString("\\\\")
				i++
			}
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
			i++
		}
	}
	// Match the entry itself or anything beneath it.
	b.WriteString("(?:/.*)?$")
	return b.String()
}

func parseClass(runes []rune, i int) (int, string) {
	n := len(runes)
	var b strings.Builder
	b.WriteByte('[')
	j := i + 1
	if j < n && (runes[j] == '!' || runes[j] == '^') {
		b.WriteByte('^')
		j++
	}
	if j < n && runes[j] == ']' {
		b.WriteString("\\]")
		j++
	}
	for j < n && runes[j] != ']' {
		c := runes[j]
		switch c {
		case '\\':
			// In gitignore, \] inside a character class escapes the ]
			// (making it literal). \X for other X means literal X.
			if j+1 < n && runes[j+1] == ']' {
				b.WriteString("\\]")
				j += 2
				continue
			}
			if j+1 < n {
				b.WriteString(regexp.QuoteMeta(string(runes[j+1])))
				j += 2
				continue
			}
			// Trailing backslash: literal
			b.WriteString("\\\\")
		case '^', '[':
			b.WriteByte('\\')
			b.WriteRune(c)
		default:
			b.WriteRune(c)
		}
		j++
	}
	if j >= n {
		return i + 1, "\\["
	}
	b.WriteByte(']')
	return j + 1, b.String()
}

// Matches tests relPath (relative to this Gitignore's root, using forward
// slashes) against the patterns. isDir indicates whether the entry is a
// directory.
func (g *Gitignore) Matches(relPath string, isDir bool) Match {
	relPath = strings.TrimPrefix(relPath, "/")
	result := None
	// Last matching pattern wins.
	for i := range g.patterns {
		p := &g.patterns[i]
		if p.dirOnly && !isDir {
			continue
		}
		if p.re.MatchString(relPath) {
			if p.negated {
				result = Whitelist
			} else {
				result = Ignore
			}
		}
	}
	return result
}

// Empty reports whether the set has no patterns.
func (g *Gitignore) Empty() bool { return len(g.patterns) == 0 }
