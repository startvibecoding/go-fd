// Package glob translates shell-style glob patterns into Go regular
// expressions. It mirrors the behavior of the `globset` crate used by fd,
// including support for `*`, `?`, `**`, character classes and `{a,b}`
// alternations.
package glob

import (
	"regexp"
	"strings"
)

// Options control how a glob is translated.
type Options struct {
	// LiteralSeparator, when true, makes `*` and `?` not match the path
	// separator `/`. This matches fd's GlobBuilder::literal_separator(true).
	LiteralSeparator bool
	// CaseInsensitive makes the resulting regex case-insensitive.
	CaseInsensitive bool
}

// ToRegex converts a glob pattern into an anchored regular expression string.
// The returned expression matches the entire input (it is anchored with ^ and $).
func ToRegex(pattern string, opts Options) (string, error) {
	var b strings.Builder
	b.WriteString("(?s)") // dot matches newline, consistent with fd
	if opts.CaseInsensitive {
		b.WriteString("(?i)")
	}
	b.WriteByte('^')
	if err := translate(pattern, opts, &b); err != nil {
		return "", err
	}
	b.WriteByte('$')
	return b.String(), nil
}

// Compile builds a compiled *regexp.Regexp from a glob pattern.
func Compile(pattern string, opts Options) (*regexp.Regexp, error) {
	re, err := ToRegex(pattern, opts)
	if err != nil {
		return nil, err
	}
	return regexp.Compile(re)
}

func translate(pattern string, opts Options, b *strings.Builder) error {
	sep := "/"
	runes := []rune(pattern)
	n := len(runes)
	i := 0
	for i < n {
		c := runes[i]
		switch c {
		case '*':
			// Check for ** (recursive)
			if i+1 < n && runes[i+1] == '*' {
				// Handle **, **/ and /** forms.
				// Consume consecutive stars.
				j := i
				for j < n && runes[j] == '*' {
					j++
				}
				// A "**" must be its own path component to be recursive.
				prevIsSep := i == 0 || runes[i-1] == '/'
				nextIsSepOrEnd := j >= n || runes[j] == '/'
				if opts.LiteralSeparator && prevIsSep && nextIsSepOrEnd {
					if j < n && runes[j] == '/' {
						// `**/` matches any number of directories (including none)
						b.WriteString("(?:[^/]*(?:/|$))*")
						i = j + 1
						continue
					}
					// trailing `**` matches everything
					b.WriteString(".*")
					i = j
					continue
				}
				// Not a real recursive glob; treat as .*
				b.WriteString(".*")
				i = j
				continue
			}
			if opts.LiteralSeparator {
				b.WriteString("[^" + sep + "]*")
			} else {
				b.WriteString(".*")
			}
			i++
		case '?':
			if opts.LiteralSeparator {
				b.WriteString("[^" + sep + "]")
			} else {
				b.WriteString(".")
			}
			i++
		case '[':
			// Character class
			end, cls, err := parseClass(runes, i)
			if err != nil {
				return err
			}
			b.WriteString(cls)
			i = end
		case '{':
			// Alternation {a,b,c}
			end, alt, err := parseAlternation(runes, i, opts)
			if err != nil {
				return err
			}
			b.WriteString(alt)
			i = end
		case '\\':
			// Escape next character literally
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
	return nil
}

// parseClass handles a [...] character class starting at index i (runes[i]=='[').
// Returns the index just past the class and the regex fragment.
func parseClass(runes []rune, i int) (int, string, error) {
	n := len(runes)
	var b strings.Builder
	b.WriteByte('[')
	j := i + 1
	if j < n && (runes[j] == '!' || runes[j] == '^') {
		b.WriteByte('^')
		j++
	}
	// A ] immediately after [ or [^ is a literal ]
	if j < n && runes[j] == ']' {
		b.WriteString("\\]")
		j++
	}
	for j < n && runes[j] != ']' {
		c := runes[j]
		switch c {
		case '\\':
			// \] inside a character class escapes the ] (making it literal).
			// \X for other X means literal X.
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
		// Unterminated class: treat '[' as literal
		return i + 1, "\\[", nil
	}
	b.WriteByte(']')
	return j + 1, b.String(), nil
}

// parseAlternation handles {a,b,c} starting at runes[i]=='{'.
func parseAlternation(runes []rune, i int, opts Options) (int, string, error) {
	n := len(runes)
	// Find matching close brace, tracking nesting.
	depth := 0
	j := i
	for j < n {
		if runes[j] == '{' {
			depth++
		} else if runes[j] == '}' {
			depth--
			if depth == 0 {
				break
			}
		}
		j++
	}
	if j >= n {
		// No closing brace: literal '{'
		return i + 1, "\\{", nil
	}
	inner := runes[i+1 : j]
	// Split on top-level commas.
	var parts []string
	depth = 0
	start := 0
	for k, c := range inner {
		switch c {
		case '{':
			depth++
		case '}':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, string(inner[start:k]))
				start = k + 1
			}
		}
	}
	parts = append(parts, string(inner[start:]))

	var b strings.Builder
	b.WriteString("(?:")
	for idx, p := range parts {
		if idx > 0 {
			b.WriteByte('|')
		}
		if err := translate(p, opts, &b); err != nil {
			return 0, "", err
		}
	}
	b.WriteByte(')')
	return j + 1, b.String(), nil
}
