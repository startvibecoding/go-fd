package glob

import "testing"

func TestToRegexMatch(t *testing.T) {
	cases := []struct {
		pattern string
		input   string
		match   bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "main.rs", false},
		{"*.go", "sub/main.go", false}, // literal separator
		{"**/*.go", "sub/main.go", true},
		{"**/*.go", "main.go", true},
		{"foo?", "fool", true},
		{"foo?", "foo", false},
		{"[abc].txt", "a.txt", true},
		{"[abc].txt", "d.txt", false},
		{"[!abc].txt", "d.txt", true},
		{"{a,b}.txt", "a.txt", true},
		{"{a,b}.txt", "c.txt", false},
		{"src/*.go", "src/x.go", true},
	}
	for _, c := range cases {
		re, err := Compile(c.pattern, Options{LiteralSeparator: true})
		if err != nil {
			t.Fatalf("compile %q: %v", c.pattern, err)
		}
		if got := re.MatchString(c.input); got != c.match {
			t.Errorf("glob %q vs %q = %v, want %v", c.pattern, c.input, got, c.match)
		}
	}
}

func TestCaseInsensitive(t *testing.T) {
	re, err := Compile("*.GO", Options{LiteralSeparator: true, CaseInsensitive: true})
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("main.go") {
		t.Error("expected case-insensitive match")
	}
}

func TestCharacterClassEscaping(t *testing.T) {
	// \] inside a character class should match a literal ].
	re, err := Compile("[a\\]b].txt", Options{LiteralSeparator: true})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, input := range []string{"a.txt", "].txt", "b.txt"} {
		if !re.MatchString(input) {
			t.Errorf("expected %q to match [a\\]b].txt", input)
		}
	}
	if re.MatchString("c.txt") {
		t.Error("expected c.txt to not match [a\\]b].txt")
	}
}
