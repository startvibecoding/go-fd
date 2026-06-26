// Package format implements fd's format/exec placeholder templates, supporting
// the tokens {}, {/}, {//}, {.}, {/.} and literal brace escaping ({{ }}).
package format

import (
	"path/filepath"
	"strings"
)

// TokenKind identifies a placeholder type.
type TokenKind int

const (
	// TokenText is a literal text run.
	TokenText TokenKind = iota
	// TokenPlaceholder is `{}` (the full path).
	TokenPlaceholder
	// TokenBasename is `{/}`.
	TokenBasename
	// TokenParent is `{//}`.
	TokenParent
	// TokenNoExt is `{.}` (path without extension).
	TokenNoExt
	// TokenBasenameNoExt is `{/.}`.
	TokenBasenameNoExt
)

// Token is a single element of a parsed template.
type Token struct {
	Kind TokenKind
	Text string
}

// Template is a parsed format string. If it contains no placeholders, HasTokens
// returns false and it is treated as fixed text.
type Template struct {
	tokens   []Token
	hasToken bool
	text     string
}

var placeholderPatterns = []struct {
	lit  string
	kind TokenKind
}{
	{"{}", TokenPlaceholder},
	{"{//}", TokenParent},
	{"{/.}", TokenBasenameNoExt},
	{"{/}", TokenBasename},
	{"{.}", TokenNoExt},
}

// Parse parses a format template string.
func Parse(s string) *Template {
	var tokens []Token
	var buf strings.Builder
	hasToken := false

	i := 0
	n := len(s)
	for i < n {
		// Handle escapes {{ and }}.
		if i+1 < n && ((s[i] == '{' && s[i+1] == '{') || (s[i] == '}' && s[i+1] == '}')) {
			buf.WriteByte(s[i])
			i += 2
			continue
		}
		matched := false
		if s[i] == '{' {
			for _, p := range placeholderPatterns {
				if strings.HasPrefix(s[i:], p.lit) {
					if buf.Len() > 0 {
						tokens = append(tokens, Token{Kind: TokenText, Text: buf.String()})
						buf.Reset()
					}
					tokens = append(tokens, Token{Kind: p.kind})
					hasToken = true
					i += len(p.lit)
					matched = true
					break
				}
			}
		}
		if matched {
			continue
		}
		buf.WriteByte(s[i])
		i++
	}

	if !hasToken {
		return &Template{text: buf.String(), hasToken: false}
	}
	if buf.Len() > 0 {
		tokens = append(tokens, Token{Kind: TokenText, Text: buf.String()})
	}
	return &Template{tokens: tokens, hasToken: true}
}

// HasTokens reports whether the template contains any placeholder tokens.
func (t *Template) HasTokens() bool { return t.hasToken }

// NumTokens returns the number of placeholder (non-text) token groups. Used to
// validate exec-batch templates.
func (t *Template) NumTokens() int {
	if !t.hasToken {
		return 0
	}
	count := 0
	for _, tok := range t.tokens {
		if tok.Kind != TokenText {
			count++
			break
		}
	}
	return count
}

// PlaceholderTemplate constructs a template consisting of a single {} token.
func PlaceholderTemplate() *Template {
	return &Template{tokens: []Token{{Kind: TokenPlaceholder}}, hasToken: true}
}

// Generate renders the template for a given path. If pathSeparator is non-empty,
// the OS path separator within placeholder values is replaced with it.
func (t *Template) Generate(path string, pathSeparator string) string {
	if !t.hasToken {
		return t.text
	}
	var b strings.Builder
	for _, tok := range t.tokens {
		switch tok.Kind {
		case TokenText:
			b.WriteString(tok.Text)
		case TokenPlaceholder:
			b.WriteString(replaceSep(path, pathSeparator))
		case TokenBasename:
			b.WriteString(replaceSep(basename(path), pathSeparator))
		case TokenParent:
			b.WriteString(replaceSep(dirname(path), pathSeparator))
		case TokenNoExt:
			b.WriteString(replaceSep(removeExtension(path), pathSeparator))
		case TokenBasenameNoExt:
			b.WriteString(replaceSep(removeExtension(basename(path)), pathSeparator))
		}
	}
	return b.String()
}

func replaceSep(s, sep string) string {
	if sep == "" {
		return s
	}
	return strings.ReplaceAll(s, string(filepath.Separator), sep)
}

func basename(path string) string {
	if path == "" {
		return ""
	}
	b := filepath.Base(path)
	if b == "." || b == string(filepath.Separator) {
		return path
	}
	return b
}

func dirname(path string) string {
	d := filepath.Dir(path)
	return d
}

// removeExtension strips the final extension component from path.
func removeExtension(path string) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if dir == "." {
		return stem
	}
	if dir == string(filepath.Separator) {
		return dir + stem
	}
	return dir + string(filepath.Separator) + stem
}
